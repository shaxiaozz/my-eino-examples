/*
* https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch04/main.go
 */
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/shaxiaozz/my-eino-examples/common/model"
	"github.com/shaxiaozz/my-eino-examples/global"

	"github.com/google/uuid"

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
)

func main() {
	var sessionID string
	var instruction string
	flag.StringVar(&sessionID, "session", "", "session ID (creates new if empty)")
	flag.StringVar(&instruction, "instruction", "", "custom instruction (empty for default)")
	flag.Parse()

	ctx := context.Background()
	cm, err := model.NewChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectRoot = cwd
		}
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}

	defaultInstruction := fmt.Sprintf(`You are a helpful assistant that helps users learn the Eino framework.

IMPORTANT: When using filesystem tools (ls, read_file, glob, grep, etc.), you MUST use absolute paths.

The project root directory is: %s

- When the user asks to list files in "current directory", use path: %s
- When the user asks to read a file with a relative path, convert it to absolute path by prepending %s
- Example: if user says "read main.go", you should call read_file with file_path: "%s/main.go"

Always use absolute paths when calling filesystem tools.`, projectRoot, projectRoot, projectRoot, projectRoot)

	agentInstruction := defaultInstruction
	if instruction != "" {
		// 如果用户通过命令行提供了自定义 instruction，使用用户的
		agentInstruction = instruction
	}

	// 创建 LocalBackend
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 创建 DeepAgent,自动注册文件系统工具
	agent, err := deep.New(ctx, &deep.Config{
		Name:           "Ch04ToolAgent",
		Description:    "ChatWithDoc agent with filesystem access via LocalBackend.",
		ChatModel:      cm,
		Instruction:    agentInstruction,
		Backend:        backend, // 提供文件系统操作能力
		StreamingShell: backend, // 提供命令执行能力
		MaxIteration:   50,      // 最多循环 50 次工具调用
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 创建 Store
	if global.Config.Model.SessionDir == "" {
		_, _ = fmt.Fprintln(os.Stderr, "请在.env配置MODEL_SESSION_DIR变量~")
		os.Exit(1)
	}
	store, err := mem.NewStore(global.Config.Model.SessionDir)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if sessionID == "" {
		sessionID = uuid.New().String()
		fmt.Printf("Created new session: %s\n", sessionID)
	} else {
		fmt.Printf("Resuming session: %s\n", sessionID)
	}

	session, err := store.GetOrCreate(sessionID)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Session title: %s\n", session.Title())
	fmt.Printf("Project root: %s\n", projectRoot)
	fmt.Println("Enter your message (empty line to exit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		userMsg := schema.UserMessage(line)
		if err := session.Append(userMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		history := session.GetMessages()
		events := runner.Run(ctx, history)
		content, err := printAndCollectAssistantFromEvents(events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		assistantMsg := schema.AssistantMessage(content, nil)
		if err := session.Append(assistantMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
	fmt.Printf("Resume with: go run ./cmd/ch04 --session %s\n", sessionID)
}

func printAndCollectAssistantFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder
	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput

			// 当角色为tool时输出工具调用的结果
			if mv.Role == schema.Tool {
				content := drainToolResult(mv)
				fmt.Printf("[tool result] %s\n", truncate(content, 200))
				continue
			}

			// 当角色不是Assistant时并且不为空时跳过
			if mv.Role != schema.Assistant && mv.Role != "" {
				continue
			}

			if mv.IsStreaming {
				mv.MessageStream.SetAutomaticClose()
				// 按 index 合并流式 ToolCall 分片
				mergedToolCalls := make(map[int]*schema.ToolCall)
				for {
					frame, err := mv.MessageStream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						return "", err
					}
					if frame != nil {
						if frame.Content != "" {
							sb.WriteString(frame.Content)
							_, _ = fmt.Fprint(os.Stdout, frame.Content)
						}
						// 按 index 合并 ToolCall 分片
						for _, tc := range frame.ToolCalls {
							idx := 0
							if tc.Index != nil {
								idx = *tc.Index
							}
							existing, ok := mergedToolCalls[idx]
							if !ok {
								copied := tc
								mergedToolCalls[idx] = &copied
							} else {
								if tc.ID != "" {
									existing.ID = tc.ID
								}
								if tc.Function.Name != "" {
									existing.Function.Name += tc.Function.Name
								}
								if tc.Function.Arguments != "" {
									existing.Function.Arguments += tc.Function.Arguments
								}
							}
						}
					}
				}

				// 流结束后打印完整的 ToolCalls
				for _, tc := range mergedToolCalls {
					fmt.Printf("\n[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
				}
				_, _ = fmt.Fprintln(os.Stdout)
				continue
			}

			if mv.Message != nil {
				sb.WriteString(mv.Message.Content)
				_, _ = fmt.Fprintln(os.Stdout, mv.Message.Content)
				for _, tc := range mv.Message.ToolCalls {
					fmt.Printf("[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
		}
	}

	return sb.String(), nil
}

func drainToolResult(mo *adk.MessageVariant) string {
	if mo.IsStreaming && mo.MessageStream != nil {
		var sb strings.Builder
		for {
			chunk, err := mo.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				sb.WriteString(chunk.Content)
			}
		}
		return sb.String()
	}
	if mo.Message != nil {
		return mo.Message.Content
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	var result bytes.Buffer
	if err := json.Compact(&result, []byte(s)); err == nil {
		s = result.String()
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

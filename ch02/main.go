/*
* https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_02_chatmodelagent_runner_agentevent/
 */
package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/shaxiaozz/my-eino-examples/common/model"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func main() {
	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	ctx := context.Background()
	cm, err := model.NewChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Ch02ChatModelAgent",
		Description: "A minimal ChatModelAgent with in-memory multi-turn history.",
		Instruction: instruction,
		Model:       cm,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	history := make([]*schema.Message, 0, 16)
	scanner := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("[log] 当前history: %v", history))

		// 1、获取用户输入
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		// 2、追加用户消息至history
		history = append(history, schema.UserMessage(line))
		_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("[log] 追加用户消息至history后: %v", history))

		// 3、调用 Runner 执行 Agent
		events := runner.Run(ctx, history)

		// 4、消费事件流，收集 assistant 回复
		content, err := printAndCollectAssistantFromEvents(events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// 5、追加 assistant 消息到 history
		history = append(history, schema.AssistantMessage(content, nil))
		_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("[log] 追加assistant消息至history后: %v", history))
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role != schema.Assistant {
			continue
		}

		if mv.IsStreaming {
			mv.MessageStream.SetAutomaticClose()
			_, _ = fmt.Fprint(os.Stdout, "assistant> ")
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if frame != nil && frame.Content != "" {
					sb.WriteString(frame.Content)
					_, _ = fmt.Fprint(os.Stdout, frame.Content)
				}
			}
			_, _ = fmt.Fprintln(os.Stdout)
			continue
		}

		if mv.Message != nil {
			_, _ = fmt.Fprint(os.Stdout, "assistant> ")
			sb.WriteString(mv.Message.Content)
			_, _ = fmt.Fprintln(os.Stdout, mv.Message.Content)
		} else {
			_, _ = fmt.Fprintln(os.Stdout)
		}
	}

	return sb.String(), nil
}

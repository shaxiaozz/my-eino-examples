package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/shaxiaozz/my-eino-examples/common/model"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"
)

// InspectionConfig 巡检配置
type InspectionConfig struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Items       []InspectionItem `json:"items"`
}

// InspectionItem 巡检项
type InspectionItem struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	Promql      string `json:"promql"`
	Description string `json:"description"`
}

func main() {
	ctx := context.Background()

	// 加载巡检配置
	inspectionCfg, err := loadInspectionConfig("agent/inspection/inspection.json")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "加载巡检配置失败:", err)
		os.Exit(1)
	}

	// 初始化模型
	cm, err := model.NewChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 构建系统指令
	instruction := buildInstruction(inspectionCfg)

	// 创建 DeepAgent
	agent, err := deep.New(ctx, &deep.Config{
		Name:         "InspectionAgent",
		Description:  "服务器巡检Agent，执行巡检命令并生成Markdown报告",
		ChatModel:    cm,
		Instruction:  instruction,
		MaxIteration: 50,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 构造用户消息，触发巡检
	now := time.Now().Format("2006-01-02 15:04:05")
	userContent := fmt.Sprintf("请执行巡检，当前时间: %s", now)
	fmt.Printf(">>> %s\n\n", userContent)

	history := []*schema.Message{schema.UserMessage(userContent)}
	events := runner.Run(ctx, history)

	report, err := collectReport(events)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "巡检执行失败:", err)
		os.Exit(1)
	}

	// 保存报告到文件
	reportFile := fmt.Sprintf("inspection_report_%s.md", time.Now().Format("20060102_150405"))
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "保存报告失败:", err)
		os.Exit(1)
	}

	fmt.Printf("\n\n巡检报告已保存至: %s\n", reportFile)
}

// loadInspectionConfig 加载巡检配置文件
func loadInspectionConfig(path string) (*InspectionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg InspectionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// buildInstruction 根据巡检配置构建系统指令
func buildInstruction(cfg *InspectionConfig) string {
	var sb strings.Builder

	sb.WriteString(`你是一个专业的服务器巡检Agent。你的任务是按照巡检项逐一执行命令，收集数据，最后生成一份美观的 Markdown 巡检报告。

## 工作流程

1. 按照下方巡检项列表，使用 execute 工具逐一执行对应的命令
2. 记录每个命令的执行结果
3. 对结果进行分析，判断是否存在异常（如CPU > 80%、内存 > 85%、磁盘 > 90% 等）
4. 最后汇总所有结果，生成一份结构清晰的 Markdown 巡检报告

## 巡检报告格式要求

请严格按照以下格式生成报告：

` + "```markdown" + `
# 🖥️ 服务器巡检报告

> **巡检时间**: YYYY-MM-DD HH:MM:SS
> **主机名**: xxx
> **巡检结果**: ✅ 正常 / ⚠️ 存在告警 / 🔴 存在严重问题

---

## 📊 巡检总览

| 分类 | 巡检项 | 状态 | 说明 |
|------|--------|------|------|
| CPU | CPU使用率 | ✅/⚠️/🔴 | 具体数值和说明 |
| ... | ... | ... | ... |

---

## 📋 详细数据

### 1. CPU

#### CPU使用率
- **状态**: ✅ 正常
- **数值**: xx%
- **原始数据**:
（命令输出）

（每个巡检项依次列出）

---

## 🔍 问题与建议

（如有异常项，列出问题和优化建议；如无异常则注明系统运行正常）
` + "```" + `

## 状态判定规则

- ✅ 正常：各指标在安全范围内
- ⚠️ 告警：CPU > 80%、内存 > 85%、磁盘 > 85%、TIME_WAIT > 5000、僵尸进程 > 0
- 🔴 严重：CPU > 95%、内存 > 95%、磁盘 > 95%、存在内核错误

## 巡检项列表

`)

	// 按 category 分组输出
	categories := make(map[string][]InspectionItem)
	var order []string
	for _, item := range cfg.Items {
		if _, exists := categories[item.Category]; !exists {
			order = append(order, item.Category)
		}
		categories[item.Category] = append(categories[item.Category], item)
	}

	for _, cat := range order {
		sb.WriteString(fmt.Sprintf("### %s\n", cat))
		for _, item := range categories[cat] {
			sb.WriteString(fmt.Sprintf("- **%s**: `%s`\n  说明: %s\n", item.Name, item.Promql, item.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("请开始执行巡检。\n")
	return sb.String()
}

// collectReport 从Agent事件流中收集最终报告内容
func collectReport(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
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

		// 打印工具调用过程
		if mv.Role == schema.Tool {
			content := drainStream(mv)
			fmt.Printf("[tool result] %s\n", truncate(content, 200))
			continue
		}

		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}

		if mv.IsStreaming {
			mv.MessageStream.SetAutomaticClose()
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
					for _, tc := range frame.ToolCalls {
						fmt.Printf("\n[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
					}
				}
			}
			continue
		}

		if mv.Message != nil {
			sb.WriteString(mv.Message.Content)
			_, _ = fmt.Fprint(os.Stdout, mv.Message.Content)
			for _, tc := range mv.Message.ToolCalls {
				fmt.Printf("[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
			}
		}
	}

	return sb.String(), nil
}

func drainStream(mo *adk.MessageVariant) string {
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
	return s[:maxLen] + "..."
}

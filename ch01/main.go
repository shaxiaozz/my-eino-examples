/*
 * https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch01/main.go
 */

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/cloudwego/eino/schema"
	"github.com/shaxiaozz/my-eino-examples/common/model"
	"io"
	"os"
	"strings"
	"time"
)

func main() {
	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: go run ./ch01/main.go -- \"your question\"")
		os.Exit(2)
	}

	ctx := context.Background()
	cm, err := model.NewChatModel(ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	messages := []*schema.Message{
		schema.SystemMessage(instruction),
		schema.UserMessage(query),
	}

	// 流式响应
	_, _ = fmt.Fprintln(os.Stdout, "===== start streaming =====")
	_, _ = fmt.Fprint(os.Stdout, "[assistant] ")
	start := time.Now()
	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer stream.Close()

	// 在流式中通常只有最后一个 frame 才会携带完整的 ResponseMeta，包含 finish_reason 和 usage 等信息
	var lastMeta *schema.ResponseMeta
	for {
		frame, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if frame != nil {
			_, _ = fmt.Fprint(os.Stdout, frame.Content)
			if frame.ResponseMeta != nil {
				lastMeta = frame.ResponseMeta
			}
		}
	}
	_, _ = fmt.Fprintln(os.Stdout)

	elapsed := time.Since(start)
	if lastMeta != nil {
		_, _ = fmt.Fprint(os.Stdout,
			fmt.Sprintf("===== response meta =====\nfinish_reason: %s\nusage_prompt_tokens: %d\nusage_completion_tokens: %d\nusage_total_tokens: %d\nresponse_time: %s\n",
				lastMeta.FinishReason,
				lastMeta.Usage.PromptTokens,
				lastMeta.Usage.CompletionTokens,
				lastMeta.Usage.TotalTokens,
				elapsed,
			),
		)
	}

	//// 等待响应
	//_, _ = fmt.Fprintln(os.Stdout, "===== start generate =====")
	//start := time.Now()
	//result, err := cm.Generate(ctx, messages)
	//if err != nil {
	//	_, _ = fmt.Fprintln(os.Stderr, fmt.Sprintf("[assistant] generate err: %v", err))
	//	os.Exit(1)
	//}
	//elapsed := time.Since(start)
	//_, _ = fmt.Fprint(os.Stdout,
	//	fmt.Sprintf("[assistant] : %s\n\n===== response meta =====\nfinish_reason: %s\nusage_prompt_tokens: %d\nusage_completion_tokens: %d\nusage_total_tokens: %d\nresponse_time: %s\n",
	//		result.Content,
	//		result.ResponseMeta.FinishReason,
	//		result.ResponseMeta.Usage.PromptTokens,
	//		result.ResponseMeta.Usage.CompletionTokens,
	//		result.ResponseMeta.Usage.TotalTokens,
	//		elapsed,
	//	),
	//)

}

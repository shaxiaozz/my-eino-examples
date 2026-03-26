/*
 * https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_01_chatmodel_and_message/
 */

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/shaxiaozz/my-eino-examples/config"
)

func main() {
	cfg := config.InitConfig()
	if cfg == nil {
		_, _ = fmt.Fprintln(os.Stderr, "加载.env配置失败")
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout,
		fmt.Sprintf("===== Model Info =====\ntype: %s\nmodel: %s\nbase_url: %s\nthinking: %t\nproxy: %t\n",
			cfg.Model.Type,
			cfg.Model.Name,
			cfg.Model.BaseUrl,
			cfg.Model.EnableThinking,
			cfg.Proxy.Enable,
		),
	)

	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	query := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if query == "" {
		_, _ = fmt.Fprintln(os.Stderr, "usage: go run ./ch01/main.go -- \"your question\"")
		os.Exit(2)
	}

	ctx := context.Background()
	cm, err := newChatModel(ctx, cfg)
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

func newChatModel(ctx context.Context, cfg *config.App) (model.ToolCallingChatModel, error) {
	// proxy
	proxyUrl, _ := url.Parse(cfg.Proxy.Host)
	if cfg.Proxy.UserName != "" {
		proxyUrl.User = url.UserPassword(cfg.Proxy.UserName, cfg.Proxy.Password)
	}
	proxy := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
		},
	}

	switch cfg.Model.Type {
	case "ark":
		chatModelConfig := &ark.ChatModelConfig{
			APIKey:  cfg.Model.ApiKey,
			Model:   cfg.Model.Name,
			BaseURL: cfg.Model.BaseUrl,
		}
		if cfg.Proxy.Enable {
			chatModelConfig.HTTPClient = proxy
		}
		return ark.NewChatModel(ctx, chatModelConfig)
	case "openai":
		chatModelConfig := &openai.ChatModelConfig{
			APIKey:  cfg.Model.ApiKey,
			Model:   cfg.Model.Name,
			BaseURL: cfg.Model.BaseUrl,
		}
		if cfg.Proxy.Enable {
			chatModelConfig.HTTPClient = proxy
		}
		return openai.NewChatModel(ctx, chatModelConfig)
	case "deepseek":
		chatModelConfig := &deepseek.ChatModelConfig{
			APIKey:  cfg.Model.ApiKey,
			Model:   cfg.Model.Name,
			BaseURL: cfg.Model.BaseUrl,
		}
		if cfg.Proxy.Enable {
			chatModelConfig.HTTPClient = proxy
		}
		return deepseek.NewChatModel(ctx, chatModelConfig)
	case "qwen":
		chatModelConfig := &qwen.ChatModelConfig{
			APIKey:  cfg.Model.ApiKey,
			Model:   cfg.Model.Name,
			BaseURL: cfg.Model.BaseUrl,
		}
		if cfg.Proxy.Enable {
			chatModelConfig.HTTPClient = proxy
		}
		chatModelConfig.EnableThinking = &cfg.Model.EnableThinking

		return qwen.NewChatModel(ctx, chatModelConfig)
	default:
		return nil, fmt.Errorf("不支持的模型类型: %s", cfg.Model.Type)
	}
}

package model

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/shaxiaozz/my-eino-examples/config"
	"github.com/shaxiaozz/my-eino-examples/global"
	"net/http"
	"net/url"
	"os"
)

func NewChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	cfg := config.InitConfig()
	if cfg == nil {
		_, _ = fmt.Fprintln(os.Stderr, "加载.env配置失败")
		os.Exit(1)
	}
	_, _ = fmt.Fprintln(os.Stdout,
		fmt.Sprintf("===== Model Info =====\ntype: %s\nmodel: %s\nbase_url: %s\nthinking: %t\nproxy: %t\nsession_dir: %s\n",
			cfg.Model.Type,
			cfg.Model.Name,
			cfg.Model.BaseUrl,
			cfg.Model.EnableThinking,
			cfg.Proxy.Enable,
			cfg.Model.SessionDir,
		),
	)
	global.Config = cfg

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

# Ch01 - ChatModel 与 Message

> 对应教程：[Eino 快速开始 - 第一章：ChatModel 与 Message](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_01_chatmodel_and_message/)

本示例演示如何使用 Eino 框架的 ChatModel 组件，通过**流式响应**与大语言模型进行对话，并在流结束后输出响应元信息（token 用量、完成原因、响应时间等）。

## 支持的模型

| 类型 | MODEL_TYPE | 说明 |
|------|-----------|------|
| OpenAI | `openai` | OpenAI 兼容接口（默认） |
| 火山方舟 | `ark` | 字节跳动火山方舟 |
| DeepSeek | `deepseek` | DeepSeek |
| 通义千问 | `qwen` | 阿里云通义千问（支持 thinking 模式） |

## 环境配置

在项目根目录创建 `.env` 文件（参考 `.env.example`）：

```bash
# 模型配置
MODEL_NAME=gpt-4o
MODEL_TYPE=openai          # openai / ark / deepseek / qwen
MODEL_BASE_URL=
MODEL_API_KEY=

# 思考模式（仅 qwen 生效）
MODEL_ENABLE_THINKING=false

# 代理配置（可选）
PROXY_ENABLE=false
PROXY_HOST=http://127.0.0.1:7890
PROXY_USERNAME=
PROXY_PASSWORD=
```

## 运行

```bash
go run ./ch01/main.go -- "你的问题"
```

可通过 `-instruction` 自定义系统提示词：

```bash
go run ./ch01/main.go -instruction "你是一个翻译助手" -- "把这段话翻译成英文：你好世界"
```

## 输出示例

```
===== Model Info =====
type: openai
model: gpt-4o
base_url: https://api.openai.com/v1
thinking: false
proxy: false

===== start streaming =====
[assistant] 你好！有什么我可以帮助你的吗？

===== response meta =====
finish_reason: stop
usage_prompt_tokens: 19
usage_completion_tokens: 12
usage_total_tokens: 31
response_time: 1.234s
```

## 代码结构

- `main.go` — 入口文件，包含流式调用逻辑和模型初始化
- `../config/` — 配置加载（从 `.env` 或环境变量读取）

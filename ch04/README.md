# Ch04 - Tool Use：让 Agent 使用工具

> 对应教程：[Eino 快速开始 - 第四章：Tool Use](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_04_tool_and_filesystem/)

本示例在第三章持久化会话的基础上，引入 **DeepAgent + LocalBackend**，使 Agent 具备文件系统操作和命令执行能力。LLM 可以自主决定调用哪个工具、传什么参数，实现真正的 Agent 交互。

## 核心概念

### Ch03 vs Ch04

| 维度 | Ch03（持久化会话） | Ch04（Tool Use） |
|------|-------------------|-----------------|
| Agent 类型 | `ChatModelAgent`（纯对话） | `DeepAgent`（带工具调用） |
| 能力范围 | 只能生成文本 | 可读写文件、搜索代码、执行命令 |
| 交互模式 | 用户问 → 模型答 | 用户问 → 模型决策 → 调用工具 → 模型总结 |
| 迭代次数 | 单轮生成 | 最多 50 次工具调用循环 |

### Tool Use 工作原理

```
用户: "列出当前目录的文件"
         ↓
┌─────────────────────────────────┐
│ LLM 收到消息 + 可用工具列表      │
│ 分析意图 → 决定调用 ls 工具      │
│ 返回: tool_call {name:"ls", ..} │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ 框架执行 ls 工具                 │
│ LocalBackend.LsInfo()           │
│ → os.ReadDir() → 文件列表       │
└─────────────────────────────────┘
         ↓
┌─────────────────────────────────┐
│ LLM 收到工具结果                 │
│ 生成最终回复给用户               │
└─────────────────────────────────┘
```

关键点：**工具选择由 LLM 完成**，不是代码中的 if-else 规则。框架只负责注册工具和编排调用循环。

### 内置工具列表

通过 `LocalBackend` 自动注册以下工具：

| 工具名 | 功能 | 来源 |
|--------|------|------|
| `ls` | 列出目录文件 | `Backend` |
| `read_file` | 读取文件内容 | `Backend` |
| `write_file` | 写入文件 | `Backend` |
| `edit_file` | 编辑文件（文本替换） | `Backend` |
| `glob` | 按模式匹配文件 | `Backend` |
| `grep` | 搜索文件内容 | `Backend` |
| `execute` | 执行 Shell 命令 | `StreamingShell` |

### 工具注册链路

```
LocalBackend（实现 filesystem.Backend 接口）
    ↓
newLsTool() / newReadFileTool() ...（工厂函数）
    ↓  InferTool: 从 Go 结构体自动生成 JSON Schema
    ↓  例如 lsArgs{Path string} → {"name":"ls","parameters":{"path":"string"}}
    ↓
filesystem Middleware 收集所有 tool.BaseTool
    ↓
DeepAgent.New() 注入 Middleware
    ↓
ChatModelAgent 将工具 Schema 随 API 请求发给 LLM
```

## 环境配置

在项目根目录的 `.env` 文件中配置：

```bash
# 模型配置
MODEL_NAME=gpt-4o
MODEL_TYPE=openai
MODEL_BASE_URL=
MODEL_API_KEY=

# 会话存储目录
MODEL_SESSION_DIR=./data/sessions
```

可选：通过 `PROJECT_ROOT` 环境变量指定工具操作的根目录（默认为当前工作目录）。

## 运行

```bash
# 创建新会话
go run ./ch04

# 恢复已有会话
go run ./ch04 --session <session-id>
```

可通过 `-instruction` 自定义系统提示词：

```bash
go run ./ch04 -instruction "你是一个代码审查助手，帮用户分析代码质量"
```

## 输出示例

```
Created new session: c683947e-c05c-4cb6-a591-e6023bbd35b6
Session title: New Session
Project root: /Users/fengjin/Desktop/GitHub/my-eino-examples
Enter your message (empty line to exit):
you> go version

[tool call] execute({"command": "go version"})
[tool result] go version go1.25.8 darwin/arm64

Go 版本是 **go1.25.8**，运行在 darwin/arm64 平台上（macOS Apple Silicon）。
you> 列出当前目录的文件

[tool call] ls({"path": "/Users/fengjin/Desktop/GitHub/my-eino-examples"})
[tool result] ch01 ch02 ch03 ch04 common config go.mod go.sum ...

当前目录包含以下文件和目录：
- ch01 ~ ch04：各章节示例代码
- common：公共模块
- go.mod / go.sum：Go 模块文件
you>

Session saved: c683947e-c05c-4cb6-a591-e6023bbd35b6
Resume with: go run ./ch04 --session c683947e-c05c-4cb6-a591-e6023bbd35b6
```

## 代码结构

- `main.go` — 入口文件，创建 DeepAgent、管理 Session、消费流式事件（含 ToolCall 分片合并）
- `../common/mem/store.go` — Session 和 Store 实现，JSONL 文件读写逻辑
- `../common/model/` — ChatModel 初始化（支持 OpenAI / Ark / DeepSeek / Qwen）
- `../config/` — 配置加载（从 `.env` 或环境变量读取）

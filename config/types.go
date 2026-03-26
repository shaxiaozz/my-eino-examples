package config

type App struct {
	Model Model
	Proxy Proxy
}

type Model struct {
	Name           string // 模型名称
	Type           string // 模型类型(openai/ark/deepseek/qwen)
	BaseUrl        string
	ApiKey         string
	EnableThinking bool
}

type Proxy struct {
	Enable   bool
	Host     string
	UserName string
	Password string
}

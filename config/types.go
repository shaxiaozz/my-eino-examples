package config

type App struct {
	Model Model
	Proxy Proxy
}

type Model struct {
	Name           string
	Type           string
	BaseUrl        string
	ApiKey         string
	EnableThinking bool
	SessionDir     string
}

type Proxy struct {
	Enable   bool
	Host     string
	UserName string
	Password string
}

package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func InitConfig() *App {
	log.Println("使用.env加载环境变量,如果手动传入环境变量,优先级比.env高~")
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("加载.env文件失败: %v", err)
		return nil
	}

	return &App{
		Model: Model{
			Name:           getEnv("MODEL_NAME", ""),
			Type:           getEnv("MODEL_TYPE", "openai"),
			BaseUrl:        getEnv("MODEL_BASE_URL", ""),
			ApiKey:         getEnv("MODEL_API_KEY", ""),
			EnableThinking: getEnv("MODEL_ENABLE_THINKING", "false") == "true",
		},
		Proxy: Proxy{
			Enable:   getEnv("PROXY_ENABLE", "false") == "true",
			Host:     getEnv("PROXY_HOST", ""),
			UserName: getEnv("PROXY_USERNAME", ""),
			Password: getEnv("PROXY_PASSWORD", ""),
		},
	}
}

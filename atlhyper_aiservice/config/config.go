package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// AIServerConfig 定义 AI Service 的 HTTP 相关配置
type AIServerConfig struct {
	Port    int           // 监听端口
	Timeout time.Duration // 超时时间
	Enable  bool          // 是否启用服务（可用于测试关闭）
}

// GeminiConfig 模型服务配置（供 client 使用）
type GeminiConfig struct {
	APIKey    string
	ModelName string
	Timeout   time.Duration
}

// Config 模块总配置
type Config struct {
	Server AIServerConfig
	Gemini GeminiConfig
}

var C Config // 全局配置实例

// 默认值
const (
	defaultPort       = 8089
	defaultTimeout    = 10 * time.Second
	defaultModelName  = "gemini-2.5-flash"
	defaultServerOn   = true
	envGeminiKey      = "GEMINI_API_KEY"
	envGeminiModel    = "GEMINI_MODEL"
	envServerPort     = "AI_HTTP_PORT"
	envServerEnable   = "AI_ENABLE"
	envGeminiTimeout  = "GEMINI_TIMEOUT"
)

// getenvOr 读取环境变量或返回默认
func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load 从环境变量加载配置
func Load() error {
	var cfg Config

	// Server.Enable
	enable := getenvOr(envServerEnable, "true")
	cfg.Server.Enable = (enable == "true" || enable == "1")

	// Server.Port
	if val := os.Getenv(envServerPort); val != "" {
		var p int
		_, err := fmt.Sscanf(val, "%d", &p)
		if err == nil && p > 0 {
			cfg.Server.Port = p
		} else {
			cfg.Server.Port = defaultPort
		}
	} else {
		cfg.Server.Port = defaultPort
	}
	cfg.Server.Timeout = defaultTimeout

	// Gemini.APIKey
	key := os.Getenv(envGeminiKey)
	if key == "" {
		return errors.New("GEMINI_API_KEY 未设置")
	}
	cfg.Gemini.APIKey = key

	// Gemini.Model
	model := getenvOr(envGeminiModel, defaultModelName)
	cfg.Gemini.ModelName = model

	// Gemini.Timeout
	if val := os.Getenv(envGeminiTimeout); val != "" {
		d, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("GEMINI_TIMEOUT 格式错误: %v", err)
		}
		cfg.Gemini.Timeout = d
	} else {
		cfg.Gemini.Timeout = defaultTimeout
	}

	C = cfg
	return nil
}

// MustLoad 同 Load，但失败时 panic
func MustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

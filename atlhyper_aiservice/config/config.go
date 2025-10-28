// atlhyper_aiservice/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// =========================================
// 🌐 统一配置结构定义
// =========================================

// AI 服务 HTTP 配置
type AIServerConfig struct {
	Port    int           // 监听端口
	Timeout time.Duration // 超时时间
}

// Gemini 模型配置
type GeminiConfig struct {
	APIKey    string        // API 密钥（从环境变量）
	ModelName string        // 模型名称
	Timeout   time.Duration // 请求超时
}

// Master 接口配置
type MasterAPIConfig struct {
	BaseURL string        // 主控服务地址
	Timeout time.Duration // 请求超时
}

// 总配置体
type Config struct {
	Server AIServerConfig
	Gemini GeminiConfig
	Master MasterAPIConfig
}

// 全局实例
var C Config

// =========================================
// ⚙️ 默认值与环境变量键名定义（含注释）
// =========================================
const (
	// ------------------------------
	// 🖥️ AI Service (HTTP 服务)
	// ------------------------------

	defaultPort = 8089
	// 默认监听端口（当 AI_HTTP_PORT 未设置时使用）

	defaultTimeout = 10 * time.Second
	// 默认请求或处理超时时间，适用于所有外部请求

	envServerPort = "AI_HTTP_PORT"
	// 可选：AI Service 对外 HTTP 端口（默认 8089）

	envServerTimeout = "AI_TIMEOUT"
	// 可选：AI Service 自身操作超时（格式示例：10s、30s、1m）

	// ------------------------------
	// 🤖 Gemini 模型服务
	// ------------------------------

	defaultModelName = "gemini-2.5-flash"
	// 默认使用的 Gemini 模型名称，可通过 GEMINI_MODEL 覆盖

	envGeminiKey = "GEMINI_API_KEY"
	// [必填] Gemini 模型调用所需的 API Key

	envGeminiModel = "GEMINI_MODEL"
	// 可选：指定 Gemini 模型名称（例如 gemini-2.0-pro、gemini-2.5-flash）

	envGeminiTimeout = "GEMINI_TIMEOUT"
	// 可选：Gemini 请求超时（格式示例：10s、30s、1m）

	// ------------------------------
	// 🛰️ Master API (主控端接口)
	// ------------------------------

	defaultMasterURL = "http://127.0.0.1:8081"
	// 默认的 Master 服务地址（用于 /ai/context/fetch 调用）

	envMasterURL = "MASTER_API_URL"
	// 可选：主控服务的访问 URL（例如 http://atlhyper-master:8081）

	envMasterTimeout = "MASTER_API_TIMEOUT"
	// 可选：访问 Master 的超时（格式同上，默认 8s）
)

// =========================================
// 🧩 加载逻辑
// =========================================

// getenvOr 返回默认值
func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load 从环境变量加载配置
func Load() error {
	var cfg Config

	// ---------- Server ----------
	if val := os.Getenv(envServerPort); val != "" {
		var p int
		if _, err := fmt.Sscanf(val, "%d", &p); err == nil && p > 0 {
			cfg.Server.Port = p
		} else {
			cfg.Server.Port = defaultPort
		}
	} else {
		cfg.Server.Port = defaultPort
	}

	if val := os.Getenv(envServerTimeout); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Server.Timeout = d
		} else {
			cfg.Server.Timeout = defaultTimeout
		}
	} else {
		cfg.Server.Timeout = defaultTimeout
	}

	// ---------- Gemini ----------
	key := os.Getenv(envGeminiKey)
	if key == "" {
		return errors.New("GEMINI_API_KEY 未设置")
	}
	cfg.Gemini.APIKey = key
	cfg.Gemini.ModelName = getenvOr(envGeminiModel, defaultModelName)

	if val := os.Getenv(envGeminiTimeout); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Gemini.Timeout = d
		} else {
			cfg.Gemini.Timeout = defaultTimeout
		}
	} else {
		cfg.Gemini.Timeout = defaultTimeout
	}

	// ---------- Master ----------
	cfg.Master.BaseURL = getenvOr(envMasterURL, defaultMasterURL)
	if val := os.Getenv(envMasterTimeout); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			cfg.Master.Timeout = d
		} else {
			cfg.Master.Timeout = 8 * time.Second
		}
	} else {
		cfg.Master.Timeout = 8 * time.Second
	}

	// ✅ 赋值全局变量
	C = cfg
	return nil
}

// MustLoad —— 启动时加载配置，失败即 panic
func MustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

// =========================================
// 🔍 Getter（供外部使用）
// =========================================
func GetServerConfig() *AIServerConfig { return &C.Server }
func GetGeminiConfig() *GeminiConfig   { return &C.Gemini }
func GetMasterAPI() *MasterAPIConfig   { return &C.Master }

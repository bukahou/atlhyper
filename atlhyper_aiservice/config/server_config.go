package config

import (
	"fmt"
	"os"
	"time"
)

type AIServerConfig struct {
	Port    int
	Timeout time.Duration
}

// ========================================
// 🧩 默认值定义（Default Values）
// ========================================
const (
	defaultPort    = 8089
	defaultTimeout = 120 * time.Second
)

// ========================================
// 🌿 环境变量键名定义（Environment Keys）
// ========================================
const (
	envServerPort    = "AI_HTTP_PORT"
	envServerTimeout = "AI_TIMEOUT"
)

// ========================================
// ⚙️ 加载逻辑
// ========================================
func loadServerConfig() AIServerConfig {
	var c AIServerConfig

	// ---------- Port ----------
	if val := os.Getenv(envServerPort); val != "" {
		var p int
		if _, err := fmt.Sscanf(val, "%d", &p); err == nil && p > 0 {
			c.Port = p
		} else {
			c.Port = defaultPort
		}
	} else {
		c.Port = defaultPort
	}

	// ---------- Timeout ----------
	if val := os.Getenv(envServerTimeout); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			c.Timeout = d
		} else {
			c.Timeout = defaultTimeout
		}
	} else {
		c.Timeout = defaultTimeout
	}

	return c
}

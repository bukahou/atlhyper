package config

import (
	"os"
	"time"
)

type MasterAPIConfig struct {
	BaseURL string
	Timeout time.Duration
}

// ========================================
// 🧩 默认值定义（Default Values）
// ========================================
const (
	defaultMasterURL      = "http://127.0.0.1:8081"
	defaultMasterTimeout   = 10 * time.Second
)

// ========================================
// 🌿 环境变量键名定义（Environment Keys）
// ========================================
const (
	envMasterURL     = "MASTER_API_URL"
	envMasterTimeout = "MASTER_API_TIMEOUT"
)

// ========================================
// ⚙️ 加载逻辑
// ========================================
func loadMasterConfig() MasterAPIConfig {
	var c MasterAPIConfig

	// ---------- Base URL ----------
	if v := os.Getenv(envMasterURL); v != "" {
		c.BaseURL = v
	} else {
		c.BaseURL = defaultMasterURL
	}

	// ---------- Timeout ----------
	if v := os.Getenv(envMasterTimeout); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Timeout = d
		} else {
			c.Timeout = defaultMasterTimeout
		}
	} else {
		c.Timeout = defaultMasterTimeout
	}

	return c
}

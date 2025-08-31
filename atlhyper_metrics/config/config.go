package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// PushConfig 主动上报相关配置
type PushConfig struct {
	Enable   bool          // 是否启用主动上报
	URL      string        // Agent 接收端 URL
	Token    string        // 可选：Bearer Token
	Interval time.Duration // 上报间隔
	Timeout  time.Duration // HTTP 超时
}

// Config 模块总配置
type Config struct {
	Push PushConfig
}

var C Config // 全局配置

// 默认值
const (
	defaultEnable   = true
	defaultInterval = 5 * time.Second
	defaultTimeout  = 5 * time.Second

	// atlhyper-agent-service
	// 192.168.0.117
	defaultAgentSvcName = "127.0.0.1"
	defaultAgentSvcPort = 8082
	defaultIngestPath   = "/ingest/metrics/v1/snapshot"
)

// getenvOr 读取环境变量，空则返回默认
func getenvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvIntOr 读取整型环境变量，非法或空则返回默认
func getenvIntOr(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

// Load 从环境变量读取配置并设置默认值
func Load() error {
	var cfg Config

	// ENABLE
	if val := os.Getenv("PUSH_ENABLE"); val != "" {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("PUSH_ENABLE 格式错误: %w", err)
		}
		cfg.Push.Enable = b
	} else {
		cfg.Push.Enable = defaultEnable
	}

	// URL（优先使用显式 PUSH_URL）
	cfg.Push.URL = os.Getenv("PUSH_URL")

	// TOKEN
	cfg.Push.Token = os.Getenv("PUSH_TOKEN")

	// INTERVAL
	if val := os.Getenv("PUSH_INTERVAL"); val != "" {
		d, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("PUSH_INTERVAL 格式错误: %w", err)
		}
		cfg.Push.Interval = d
	} else {
		cfg.Push.Interval = defaultInterval
	}

	// TIMEOUT
	if val := os.Getenv("PUSH_TIMEOUT"); val != "" {
		d, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("PUSH_TIMEOUT 格式错误: %w", err)
		}
		cfg.Push.Timeout = d
	} else {
		cfg.Push.Timeout = defaultTimeout
	}

	// ===== 当启用且未给出 PUSH_URL 时，按默认规则推导 =====
	if cfg.Push.Enable && cfg.Push.URL == "" {
		svcName := getenvOr("AGENT_SVC_NAME", defaultAgentSvcName)
		svcNS := os.Getenv("AGENT_SVC_NAMESPACE") // 可选：为空表示同 ns，使用短名
		svcPort := getenvIntOr("AGENT_SVC_PORT", defaultAgentSvcPort)
		path := getenvOr("INGEST_PATH", defaultIngestPath)

		host := svcName
		if svcNS != "" {
			// 跨命名空间或想更稳妥时使用 FQDN
			host = fmt.Sprintf("%s.%s.svc", svcName, svcNS)
		}
		cfg.Push.URL = fmt.Sprintf("http://%s:%d%s", host, svcPort, path)
	}

	// 校验
	if cfg.Push.Enable && cfg.Push.URL == "" {
		return errors.New("PUSH_ENABLE=true 但无法确定 PUSH_URL（也未能根据默认规则推导）")
	}

	C = cfg
	return nil
}

// MustLoad 同 Load，但失败时直接 panic
func MustLoad() {
	if err := Load(); err != nil {
		panic(err)
	}
}

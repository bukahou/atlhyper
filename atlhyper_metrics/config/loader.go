// atlhyper_metrics/config/loader.go
// 配置加载逻辑
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Load 从环境变量读取配置并设置默认值
func Load() error {
	var cfg Config

	// ENABLE
	cfg.Push.Enable = getBool("PUSH_ENABLE")

	// URL（优先使用显式 PUSH_URL）
	cfg.Push.URL = getString("PUSH_URL")

	// TOKEN
	cfg.Push.Token = getString("PUSH_TOKEN")

	// INTERVAL
	cfg.Push.Interval = getDuration("PUSH_INTERVAL")

	// TIMEOUT
	cfg.Push.Timeout = getDuration("PUSH_TIMEOUT")

	// 当启用且未给出 PUSH_URL 时，按默认规则推导
	if cfg.Push.Enable && cfg.Push.URL == "" {
		svcName := getString("AGENT_SVC_NAME")
		svcNS := getString("AGENT_SVC_NAMESPACE")
		svcPort := getInt("AGENT_SVC_PORT")
		path := getString("INGEST_PATH")

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

	// -------------------- 指标采集配置 --------------------
	cfg.Collect.NodeName = getString("NODE_NAME")
	cfg.Collect.ProcRoot = getString("PROC_ROOT")
	cfg.Collect.SysRoot = getString("SYS_ROOT")

	// NodeName 为空时自动获取 hostname
	if cfg.Collect.NodeName == "" {
		cfg.Collect.NodeName, _ = os.Hostname()
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

// ==================== 工具函数 ====================

func getBool(envKey string) bool {
	if val := os.Getenv(envKey); val != "" {
		val = strings.ToLower(val)
		return val == "true" || val == "1" || val == "yes"
	}
	return defaultBools[envKey]
}

func getString(envKey string) string {
	if val := os.Getenv(envKey); val != "" {
		return val
	}
	return defaultStrings[envKey]
}

func getInt(envKey string) int {
	if val := os.Getenv(envKey); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return defaultInts[envKey]
}

func getDuration(envKey string) time.Duration {
	if val := os.Getenv(envKey); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	def := defaultDurations[envKey]
	d, _ := time.ParseDuration(def)
	return d
}

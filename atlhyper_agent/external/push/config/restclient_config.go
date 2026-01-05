// atlhyper_agent/external/push/config/restclient_config.go
package config

import (
	"strings"
	"time"

	agentconfig "AtlHyper/atlhyper_agent/config"
)

// RestClientConfig 定义 RESTful 客户端的配置。
// Path 不再提供默认值，调用时必须指定。
type RestClientConfig struct {
	BaseURL        string            // 基础地址，例如 http://master:8080
	Path           string            // 上报路径，例如 /ingest/events/cleaned（必须显式设置）
	Timeout        time.Duration     // 单次请求超时
	Gzip           bool              // 是否对请求体启用 gzip
	MaxRespBytes   int64             // 响应体读取上限
	DefaultHeaders map[string]string // 默认请求头
}

// NewDefaultRestClientConfig 从全局配置创建 REST 客户端配置
func NewDefaultRestClientConfig() RestClientConfig {
	base := strings.TrimRight(agentconfig.GlobalConfig.RestClient.BaseURL, "/")

	return RestClientConfig{
		BaseURL:        base,
		Path:           "",
		Timeout:        agentconfig.GlobalConfig.RestClient.Timeout,
		Gzip:           agentconfig.GlobalConfig.RestClient.Gzip,
		MaxRespBytes:   agentconfig.GlobalConfig.RestClient.MaxRespBytes,
		DefaultHeaders: map[string]string{},
	}
}

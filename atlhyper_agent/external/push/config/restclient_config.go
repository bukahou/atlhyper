// internal/push/config/restclient_config.go
package config

import (
	"os"
	"strings"
	"time"
)

// RestClientConfig 定义 RESTful 客户端的配置。
// Path 不再提供默认值，调用时必须指定。
type RestClientConfig struct {
	BaseURL        string            // 基础地址，例如 http://master:8081/ingest/v1
	Path           string            // 上报路径，例如 /events/cleaned（必须显式设置）
	Timeout        time.Duration     // 单次请求超时
	Gzip           bool              // 是否对请求体启用 gzip
	MaxRespBytes   int64             // 响应体读取上限
	DefaultHeaders map[string]string // 默认请求头
}

// func NewDefaultRestClientConfig() RestClientConfig {
// 	return RestClientConfig{
// 		BaseURL:        "http://localhost:8081",
// 		Path:           "",           
// 		Timeout:        8 * time.Second,
// 		Gzip:           true,
// 		MaxRespBytes:   1 << 20, // 1MB
// 		DefaultHeaders: map[string]string{},
// 	}
// }


const defaultBaseURL = "http://localhost:8081" // 默认值

func NewDefaultRestClientConfig() RestClientConfig {
	base := strings.TrimSpace(os.Getenv("AGENT_API_BASE_URL"))
	if base == "" {
		base = defaultBaseURL
	}
	base = strings.TrimRight(base, "/") // 规范化，去掉末尾的 "/"

	return RestClientConfig{
		BaseURL:        base,
		Path:           "",
		Timeout:        8 * time.Second,
		Gzip:           true,
		MaxRespBytes:   1 << 20, // 1MB
		DefaultHeaders: map[string]string{},
	}
}
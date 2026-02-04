// atlhyper_agent_v2/sdk/interfaces_metrics.go
// 指标采集接口定义
package sdk

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
)

// MetricsScraper 指标采集器接口
//
// 负责从 Prometheus 格式的端点采集指标数据。
// 支持手动配置 URL 和自动发现两种模式。
type MetricsScraper interface {
	// Scrape 从指定 URL 采集指标
	// 返回解析后的 IngressMetrics
	Scrape(ctx context.Context, url string) (*model.IngressMetrics, error)

	// DiscoverIngressURL 自动发现 Ingress Controller 的指标 URL
	// 扫描所有命名空间，通过标签识别 Ingress Controller
	// 返回: url, ingressType (nginx/traefik/kong), error
	DiscoverIngressURL(ctx context.Context) (string, string, error)

	// SetIngressType 设置 Ingress 类型（用于自动发现后更新）
	SetIngressType(ingressType string)
}

// IngressRouteCollector IngressRoute 采集器接口
//
// 负责采集 Traefik IngressRoute CRD 配置信息。
// 用于建立 Traefik service 名称与实际域名/路径的映射关系。
type IngressRouteCollector interface {
	// CollectRoutes 采集所有 IngressRoute 配置
	// 解析 match 规则，提取 Host() 和 PathPrefix() 信息
	// 返回解析后的路由信息列表
	CollectRoutes(ctx context.Context) ([]model.IngressRouteInfo, error)
}

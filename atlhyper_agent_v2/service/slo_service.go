// atlhyper_agent_v2/service/slo_service.go
// SLO 指标采集服务
package service

import (
	"context"
	"sync"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var sloLog = logger.Module("SLOService")

// SLOService SLO 指标采集服务接口
type SLOService interface {
	// Collect 采集 Ingress 指标
	Collect(ctx context.Context) (*model.IngressMetrics, error)

	// CollectRoutes 采集 IngressRoute 配置
	// 返回 Traefik service 名称到域名/路径的映射信息
	CollectRoutes(ctx context.Context) ([]model.IngressRouteInfo, error)

	// SetMetricsURL 设置指标 URL（用于自动发现后更新）
	SetMetricsURL(url string)

	// GetMetricsURL 获取当前指标 URL
	GetMetricsURL() string
}

// sloService SLO 服务实现
type sloService struct {
	scraper        sdk.MetricsScraper
	routeCollector sdk.IngressRouteCollector
	metricsURL     string
	autoDiscover   bool

	mu sync.RWMutex
}

// SLOServiceConfig SLO 服务配置
type SLOServiceConfig struct {
	MetricsURL   string // 手动配置的指标 URL
	AutoDiscover bool   // 是否自动发现（扫描所有命名空间）
}

// NewSLOService 创建 SLO 服务
func NewSLOService(scraper sdk.MetricsScraper, routeCollector sdk.IngressRouteCollector, config SLOServiceConfig) SLOService {
	return &sloService{
		scraper:        scraper,
		routeCollector: routeCollector,
		metricsURL:     config.MetricsURL,
		autoDiscover:   config.AutoDiscover,
	}
}

// Collect 采集 Ingress 指标
func (s *sloService) Collect(ctx context.Context) (*model.IngressMetrics, error) {
	url := s.GetMetricsURL()

	// 如果没有 URL 且启用了自动发现，尝试发现
	if url == "" && s.autoDiscover {
		discoveredURL, ingressType, err := s.scraper.DiscoverIngressURL(ctx)
		if err != nil {
			sloLog.Warn("自动发现 Ingress 失败", "err", err)
			return nil, err
		}
		s.SetMetricsURL(discoveredURL)
		// 同时更新 Ingress 类型，确保解析器使用正确的格式
		if ingressType != "" {
			s.scraper.SetIngressType(ingressType)
			sloLog.Info("自动发现 Ingress Controller", "url", discoveredURL, "type", ingressType)
		}
		url = discoveredURL
	}

	if url == "" {
		return nil, nil // 没有配置 URL
	}

	// 采集指标
	metrics, err := s.scraper.Scrape(ctx, url)
	if err != nil {
		sloLog.Warn("采集指标失败", "url", url, "err", err)
		return nil, err
	}

	sloLog.Debug("采集指标成功",
		"url", url,
		"counters", len(metrics.Counters),
		"histograms", len(metrics.Histograms),
	)

	return metrics, nil
}

// SetMetricsURL 设置指标 URL
func (s *sloService) SetMetricsURL(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metricsURL = url
}

// GetMetricsURL 获取当前指标 URL
func (s *sloService) GetMetricsURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metricsURL
}

// CollectRoutes 采集 IngressRoute 配置
func (s *sloService) CollectRoutes(ctx context.Context) ([]model.IngressRouteInfo, error) {
	if s.routeCollector == nil {
		return nil, nil
	}
	return s.routeCollector.CollectRoutes(ctx)
}

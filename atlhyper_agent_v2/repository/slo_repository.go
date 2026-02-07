// Package repository 数据访问层
//
// slo_repository.go - SLO 数据仓库实现
//
// 负责从 Ingress Controller 采集 SLO 指标数据:
//   - 调用 sdk.IngressClient 采集原始指标
//   - 通过 SLOSnapshotManager 计算增量
//   - 组装为 SLOSnapshot 返回
//
// 架构位置:
//
//	Service (SnapshotService)
//	    ↓ 调用
//	SLORepository (本文件) ← 数据采集 + 增量计算
//	    ↓ 调用
//	SDK (IngressClient)    ← HTTP 采集
//	    ↓
//	Ingress Controller
package repository

import (
	"context"
	"sync"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var sloLog = logger.Module("SLORepository")

// sloRepository SLO 数据仓库实现
type sloRepository struct {
	ingressClient sdk.IngressClient
	snapshot      *SLOSnapshotManager

	// URL 管理
	mu           sync.RWMutex
	metricsURL   string
	autoDiscover bool
}

// NewSLORepository 创建 SLO 数据仓库
//
// 参数:
//   - ingressClient: Ingress Controller 客户端
//   - metricsURL: 手动配置的指标 URL（为空则自动发现）
//   - autoDiscover: 是否启用自动发现
func NewSLORepository(ingressClient sdk.IngressClient, metricsURL string, autoDiscover bool) SLORepository {
	return &sloRepository{
		ingressClient: ingressClient,
		snapshot:      NewSLOSnapshotManager(),
		metricsURL:    metricsURL,
		autoDiscover:  autoDiscover,
	}
}

// Collect 采集 SLO 指标数据
func (r *sloRepository) Collect(ctx context.Context) (*model_v2.SLOSnapshot, error) {
	url := r.getMetricsURL()

	// 自动发现
	if url == "" && r.autoDiscover {
		discoveredURL, ingressType, err := r.ingressClient.DiscoverURL(ctx)
		if err != nil {
			sloLog.Warn("自动发现 Ingress 失败", "err", err)
			return nil, err
		}
		r.setMetricsURL(discoveredURL)
		if ingressType != "" {
			r.ingressClient.SetIngressType(ingressType)
			sloLog.Info("自动发现 Ingress Controller", "url", discoveredURL, "type", ingressType)
		}
		url = discoveredURL
	}

	if url == "" {
		return nil, nil
	}

	// 采集原始指标
	rawMetrics, err := r.ingressClient.ScrapeMetrics(ctx, url)
	if err != nil {
		sloLog.Warn("采集指标失败", "url", url, "err", err)
		return nil, err
	}

	// 计算增量
	counterDeltas := r.snapshot.CalculateCounterDeltas(rawMetrics.Counters)
	histogramDeltas := r.snapshot.CalculateHistogramDeltas(rawMetrics.Histograms)

	// 组装增量后的 IngressMetrics
	metrics := model_v2.IngressMetrics{
		Timestamp: rawMetrics.Timestamp,
	}

	for _, d := range counterDeltas {
		metrics.Counters = append(metrics.Counters, model_v2.IngressCounterMetric{
			Host:       d.Host,
			Status:     d.Status,
			MetricType: d.MetricType,
			Value:      d.Delta,
		})
	}

	for _, d := range histogramDeltas {
		buckets := make(map[string]int64)
		for le, delta := range d.BucketDeltas {
			buckets[le] = delta
		}
		metrics.Histograms = append(metrics.Histograms, model_v2.IngressHistogramMetric{
			Host:    d.Host,
			Buckets: buckets,
			Sum:     d.SumDelta,
			Count:   d.CountDelta,
		})
	}

	sloLog.Debug("采集指标成功",
		"url", url,
		"counters", len(metrics.Counters),
		"histograms", len(metrics.Histograms),
	)

	return &model_v2.SLOSnapshot{
		Metrics: metrics,
	}, nil
}

// CollectRoutes 采集 IngressRoute 配置
func (r *sloRepository) CollectRoutes(ctx context.Context) ([]model_v2.IngressRouteInfo, error) {
	sdkRoutes, err := r.ingressClient.CollectRoutes(ctx)
	if err != nil {
		return nil, err
	}

	// 转换 sdk.IngressRouteInfo → model_v2.IngressRouteInfo
	routes := make([]model_v2.IngressRouteInfo, 0, len(sdkRoutes))
	for _, r := range sdkRoutes {
		routes = append(routes, model_v2.IngressRouteInfo{
			Name:        r.Name,
			Namespace:   r.Namespace,
			Domain:      r.Domain,
			PathPrefix:  r.PathPrefix,
			ServiceKey:  r.ServiceKey,
			ServiceName: r.ServiceName,
			ServicePort: r.ServicePort,
			TLS:         r.TLS,
		})
	}
	return routes, nil
}

func (r *sloRepository) getMetricsURL() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metricsURL
}

func (r *sloRepository) setMetricsURL(url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metricsURL = url
}

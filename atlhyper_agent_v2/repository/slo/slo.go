package slo

import (
	"context"
	"sync"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var sloLog = logger.Module("SLORepository")

// =============================================================================
// SLO Repository
// =============================================================================

// sloRepository SLO 数据仓库实现
type sloRepository struct {
	ingressClient sdk.IngressClient
	snapshot      *sloSnapshotManager

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
func NewSLORepository(ingressClient sdk.IngressClient, metricsURL string, autoDiscover bool) repository.SLORepository {
	return &sloRepository{
		ingressClient: ingressClient,
		snapshot:      newSLOSnapshotManager(),
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
	counterDeltas := r.snapshot.calculateCounterDeltas(rawMetrics.Counters)
	histogramDeltas := r.snapshot.calculateHistogramDeltas(rawMetrics.Histograms)

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

// =============================================================================
// Counter 快照管理（增量计算）
// =============================================================================

// counterDelta Counter 增量数据
type counterDelta struct {
	Host       string
	Status     string
	MetricType string
	Delta      int64
}

// histogramDelta Histogram 增量数据
type histogramDelta struct {
	Host         string
	BucketDeltas map[string]int64
	SumDelta     float64
	CountDelta   int64
}

// sloSnapshotManager Counter 快照管理器
//
// 维护上一次采集的 Counter 累计值，用于计算增量。
// Counter 类型指标 (如 requests_total) 是累计值，
// 需要 delta = current - previous 才能得到时间段内的增量。
//
// 如果 delta < 0，说明 Ingress Controller 重启导致 Counter 重置，
// 此时直接使用当前值作为增量。
type sloSnapshotManager struct {
	mu    sync.RWMutex
	prev  map[string]int64   // counter key -> previous value
	hPrev map[string]int64   // histogram bucket key -> previous value
	sPrev map[string]float64 // histogram sum key -> previous value
	cPrev map[string]int64   // histogram count key -> previous value
}

func newSLOSnapshotManager() *sloSnapshotManager {
	return &sloSnapshotManager{
		prev:  make(map[string]int64),
		hPrev: make(map[string]int64),
		sPrev: make(map[string]float64),
		cPrev: make(map[string]int64),
	}
}

// calculateCounterDeltas 计算 Counter 增量
func (m *sloSnapshotManager) calculateCounterDeltas(counters []sdk.IngressCounterMetric) []counterDelta {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deltas []counterDelta
	newPrev := make(map[string]int64)

	for _, c := range counters {
		key := counterKey(c.Host, c.Status, c.MetricType)
		newPrev[key] = c.Value

		prevVal, hasPrev := m.prev[key]
		delta := c.Value
		if hasPrev {
			delta = c.Value - prevVal
			if delta < 0 {
				delta = c.Value // Counter 重置
			}
		}

		if delta > 0 {
			deltas = append(deltas, counterDelta{
				Host:       c.Host,
				Status:     c.Status,
				MetricType: c.MetricType,
				Delta:      delta,
			})
		}
	}

	m.prev = newPrev
	return deltas
}

// calculateHistogramDeltas 计算 Histogram 增量
func (m *sloSnapshotManager) calculateHistogramDeltas(histograms []sdk.IngressHistogramMetric) []histogramDelta {
	m.mu.Lock()
	defer m.mu.Unlock()

	var deltas []histogramDelta
	newHPrev := make(map[string]int64)
	newSPrev := make(map[string]float64)
	newCPrev := make(map[string]int64)

	for _, h := range histograms {
		hd := histogramDelta{
			Host:         h.Host,
			BucketDeltas: make(map[string]int64),
		}

		// Bucket 增量
		for le, count := range h.Buckets {
			bKey := bucketKey(h.Host, le)
			newHPrev[bKey] = count

			prevVal, hasPrev := m.hPrev[bKey]
			delta := count
			if hasPrev {
				delta = count - prevVal
				if delta < 0 {
					delta = count
				}
			}
			if delta > 0 {
				hd.BucketDeltas[le] = delta
			}
		}

		// Sum 增量
		sKey := h.Host + "|sum"
		newSPrev[sKey] = h.Sum
		prevSum, hasPrev := m.sPrev[sKey]
		hd.SumDelta = h.Sum
		if hasPrev {
			hd.SumDelta = h.Sum - prevSum
			if hd.SumDelta < 0 {
				hd.SumDelta = h.Sum
			}
		}

		// Count 增量
		cKey := h.Host + "|count"
		newCPrev[cKey] = h.Count
		prevCount, hasPrev := m.cPrev[cKey]
		hd.CountDelta = h.Count
		if hasPrev {
			hd.CountDelta = h.Count - prevCount
			if hd.CountDelta < 0 {
				hd.CountDelta = h.Count
			}
		}

		deltas = append(deltas, hd)
	}

	m.hPrev = newHPrev
	m.sPrev = newSPrev
	m.cPrev = newCPrev
	return deltas
}

// counterKey 生成 Counter 唯一键
func counterKey(host, status, metricType string) string {
	return host + "|" + status + "|" + metricType
}

// bucketKey 生成 Bucket 唯一键
func bucketKey(host, le string) string {
	return host + "|" + le
}

// atlhyper_master_v2/slo/processor.go
// SLO 数据处理器
// 接收 Agent 上报的 Ingress Metrics，计算增量，写入数据库
package slo

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v2"
)

// Processor SLO 数据处理器
type Processor struct {
	repo   database.SLORepository
	mu     sync.Mutex
}

// NewProcessor 创建 SLO Processor
func NewProcessor(repo database.SLORepository) *Processor {
	return &Processor{
		repo: repo,
	}
}

// ProcessIngressMetrics 处理 Agent 上报的 Ingress 指标
// 由 master.go 回调调用，直接传入数据（不查 datahub）
func (p *Processor) ProcessIngressMetrics(ctx context.Context, clusterID string, metrics *model_v2.IngressMetrics) error {
	if metrics == nil {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// 1. 处理 Counter 类型指标
	if err := p.processCounterMetrics(ctx, clusterID, metrics.Counters, now); err != nil {
		log.Printf("[SLO] 处理 Counter 指标失败: %v", err)
		return err
	}

	// 2. 处理 Histogram 类型指标
	if err := p.processHistogramMetrics(ctx, clusterID, metrics.Histograms, now); err != nil {
		log.Printf("[SLO] 处理 Histogram 指标失败: %v", err)
		return err
	}

	return nil
}

// ProcessIngressRoutes 处理 Agent 上报的 IngressRoute 映射信息
// 更新 service_key -> domain/path 的映射关系
func (p *Processor) ProcessIngressRoutes(ctx context.Context, clusterID string, routes []model_v2.IngressRouteInfo) error {
	if len(routes) == 0 {
		return nil
	}

	now := time.Now()

	for _, route := range routes {
		mapping := &database.SLORouteMapping{
			ClusterID:   clusterID,
			Domain:      route.Domain,
			PathPrefix:  route.PathPrefix,
			IngressName: route.Name,
			Namespace:   route.Namespace,
			TLS:         route.TLS,
			ServiceKey:  route.ServiceKey,
			ServiceName: route.ServiceName,
			ServicePort: route.ServicePort,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := p.repo.UpsertRouteMapping(ctx, mapping); err != nil {
			log.Printf("[SLO] 更新路由映射失败: %v", err)
			// 继续处理其他映射
		}
	}

	return nil
}

// resolveServiceKey 将 service key 解析为 domain/path
// 返回 domain, pathPrefix
func (p *Processor) resolveServiceKey(ctx context.Context, clusterID, serviceKey string) (string, string) {
	mapping, err := p.repo.GetRouteMappingByServiceKey(ctx, clusterID, serviceKey)
	if err != nil || mapping == nil {
		return "", "/" // 未找到映射，返回空 domain
	}
	return mapping.Domain, mapping.PathPrefix
}

// processCounterMetrics 处理 Counter 类型指标
func (p *Processor) processCounterMetrics(ctx context.Context, clusterID string, counters []model_v2.IngressCounterMetric, now time.Time) error {
	// 按 host 分组聚合
	hostMetrics := make(map[string]*hostCounterData)

	for _, c := range counters {
		if c.Host == "" {
			continue
		}

		// 获取或创建 host 数据
		hd, ok := hostMetrics[c.Host]
		if !ok {
			hd = &hostCounterData{
				ingressName:  c.IngressName,
				ingressClass: c.IngressClass,
				namespace:    c.Namespace,
				service:      c.Service,
				tls:          c.TLS,
			}
			hostMetrics[c.Host] = hd
		}

		// 更新 snapshot 并计算增量
		snapshot := &database.IngressCounterSnapshot{
			ClusterID:    clusterID,
			Host:         c.Host,
			IngressName:  c.IngressName,
			IngressClass: c.IngressClass,
			Namespace:    c.Namespace,
			Service:      c.Service,
			TLS:          c.TLS,
			Method:       c.Method,
			Status:       c.Status,
			CounterValue: c.Value,
			UpdatedAt:    now,
		}

		// 获取之前的值
		prevSnapshots, err := p.repo.GetCounterSnapshot(ctx, clusterID, c.Host)
		if err != nil {
			log.Printf("[SLO] 获取 Counter Snapshot 失败: %v", err)
		}

		var prevValue int64
		var foundPrev bool
		for _, ps := range prevSnapshots {
			if ps.Method == c.Method && ps.Status == c.Status {
				prevValue = ps.CounterValue
				foundPrev = true
				break
			}
		}

		// 只有找到 prevSnapshot 时才计算增量
		// 第一次采集时只记录 snapshot，不写入 raw（无法计算正确增量）
		if foundPrev {
			delta := CalculateDelta(c.Value, prevValue)
			hd.totalRequests += delta
			if isErrorStatus(c.Status) {
				hd.errorRequests += delta
			}
		}

		// 更新 snapshot（总是更新，为下次采集准备 prevValue）
		if err := p.repo.UpsertCounterSnapshot(ctx, snapshot); err != nil {
			log.Printf("[SLO] 更新 Counter Snapshot 失败: %v", err)
		}
	}

	// 写入 raw metrics
	for host, hd := range hostMetrics {
		if hd.totalRequests == 0 {
			continue
		}

		// 解析 service key 获取 domain/path
		domain, pathPrefix := p.resolveServiceKey(ctx, clusterID, host)

		raw := &database.SLOMetricsRaw{
			ClusterID:     clusterID,
			Host:          host,
			Domain:        domain,
			PathPrefix:    pathPrefix,
			Timestamp:     now,
			TotalRequests: hd.totalRequests,
			ErrorRequests: hd.errorRequests,
		}

		if err := p.repo.InsertRawMetrics(ctx, raw); err != nil {
			log.Printf("[SLO] 写入 Raw Metrics 失败: %v", err)
		}
	}

	return nil
}

// hostCounterData host 级别的 counter 聚合数据
type hostCounterData struct {
	ingressName   string
	ingressClass  string
	namespace     string
	service       string
	tls           bool
	totalRequests int64
	errorRequests int64
}

// processHistogramMetrics 处理 Histogram 类型指标
func (p *Processor) processHistogramMetrics(ctx context.Context, clusterID string, histograms []model_v2.IngressHistogramMetric, now time.Time) error {
	// 按 host 分组
	hostBuckets := make(map[string]*hostHistogramData)

	for _, h := range histograms {
		if h.Host == "" {
			continue
		}

		// 获取或创建 host 数据
		hd, ok := hostBuckets[h.Host]
		if !ok {
			hd = &hostHistogramData{
				ingressName: h.IngressName,
				namespace:   h.Namespace,
				buckets:     make(map[float64]int64),
			}
			hostBuckets[h.Host] = hd
		}

		// 获取之前的 histogram snapshot
		prevSnapshots, err := p.repo.GetHistogramSnapshot(ctx, clusterID, h.Host)
		if err != nil {
			log.Printf("[SLO] 获取 Histogram Snapshot 失败: %v", err)
		}

		// 构建之前值的 map
		prevValues := make(map[float64]int64)
		for _, ps := range prevSnapshots {
			prevValues[ps.LE] = ps.BucketValue
		}

		// 标记是否是第一次采集（没有 prevSnapshot）
		isFirstCollection := len(prevSnapshots) == 0

		// 处理每个 bucket (h.Buckets 的键是字符串，需要转换为 float64)
		for leStr, value := range h.Buckets {
			// 将字符串键解析为 float64
			le, err := strconv.ParseFloat(leStr, 64)
			if err != nil {
				log.Printf("[SLO] 解析 LE 失败: %s -> %v", leStr, err)
				continue
			}

			// 只有非首次采集时才计算增量
			// 第一次采集时只记录 snapshot，不写入 bucket 数据
			if !isFirstCollection {
				delta := CalculateDelta(value, prevValues[le])
				hd.buckets[le] += delta
			}

			// 更新 snapshot（总是更新，为下次采集准备 prevValue）
			snapshot := &database.IngressHistogramSnapshot{
				ClusterID:   clusterID,
				Host:        h.Host,
				IngressName: h.IngressName,
				Namespace:   h.Namespace,
				LE:          le,
				BucketValue: value,
				UpdatedAt:   now,
			}
			if err := p.repo.UpsertHistogramSnapshot(ctx, snapshot); err != nil {
				log.Printf("[SLO] 更新 Histogram Snapshot 失败: %v", err)
			}
		}

		// 保存 sum 和 count
		hd.sum += h.Sum
		hd.count += h.Count
	}

	// 更新 raw metrics 中的 bucket 数据
	for host, hd := range hostBuckets {
		// 获取刚才写入的 raw metrics（同一时间戳）
		rawMetrics, err := p.repo.GetRawMetrics(ctx, clusterID, host, now.Add(-time.Second), now.Add(time.Second))
		if err != nil || len(rawMetrics) == 0 {
			// 如果没有对应的 counter 数据，创建新记录
			// 解析 service key 获取 domain/path
			domain, pathPrefix := p.resolveServiceKey(ctx, clusterID, host)

			raw := &database.SLOMetricsRaw{
				ClusterID:  clusterID,
				Host:       host,
				Domain:     domain,
				PathPrefix: pathPrefix,
				Timestamp:  now,
			}
			setBucketsToRaw(raw, hd.buckets)
			if err := p.repo.InsertRawMetrics(ctx, raw); err != nil {
				log.Printf("[SLO] 写入 Raw Metrics (histogram) 失败: %v", err)
			}
		} else {
			// 更新已有记录的 bucket 数据
			raw := rawMetrics[len(rawMetrics)-1] // 取最新的一条
			setBucketsToRaw(raw, hd.buckets)
			if err := p.repo.UpdateRawMetricsBuckets(ctx, raw); err != nil {
				log.Printf("[SLO] 更新 Raw Metrics bucket 失败: %v", err)
			}
		}
	}

	return nil
}

// hostHistogramData host 级别的 histogram 聚合数据
type hostHistogramData struct {
	ingressName string
	namespace   string
	buckets     map[float64]int64
	sum         float64
	count       int64
}

// isErrorStatus 判断是否为错误状态码
func isErrorStatus(status string) bool {
	// 5xx 错误
	if len(status) > 0 && status[0] == '5' {
		return true
	}
	return false
}

// setBucketsToRaw 设置 bucket 值到 raw metrics
// 支持 nginx-ingress 和 Traefik 的不同 bucket 边界
func setBucketsToRaw(raw *database.SLOMetricsRaw, buckets map[float64]int64) {
	// 遍历所有 bucket，映射到最接近的标准列
	for le, v := range buckets {
		switch {
		case le <= 0.005:
			raw.Bucket5ms += v
		case le <= 0.01:
			raw.Bucket10ms += v
		case le <= 0.025:
			raw.Bucket25ms += v
		case le <= 0.05:
			raw.Bucket50ms += v
		case le <= 0.1:
			raw.Bucket100ms += v
		case le <= 0.25:
			raw.Bucket250ms += v
		case le <= 0.3: // Traefik: 0.3s -> 300ms
			raw.Bucket250ms += v
		case le <= 0.5:
			raw.Bucket500ms += v
		case le <= 1.0:
			raw.Bucket1s += v
		case le <= 1.2: // Traefik: 1.2s -> 1200ms
			raw.Bucket1s += v
		case le <= 2.5:
			raw.Bucket2500ms += v
		case le <= 5.0:
			raw.Bucket5s += v
		case le <= 10.0:
			raw.Bucket10s += v
		default:
			raw.BucketInf += v
		}
	}
}

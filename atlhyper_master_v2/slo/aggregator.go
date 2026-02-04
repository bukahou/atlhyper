// atlhyper_master_v2/slo/aggregator.go
// SLO 小时聚合器
// 定时将 raw 数据聚合为 hourly 数据
package slo

import (
	"context"
	"log"
	"math"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// Aggregator 小时聚合器
type Aggregator struct {
	repo     database.SLORepository
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewAggregator 创建聚合器
func NewAggregator(repo database.SLORepository, interval time.Duration) *Aggregator {
	return &Aggregator{
		repo:     repo,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start 启动聚合器
func (a *Aggregator) Start() {
	go a.run()
	log.Printf("[SLO Aggregator] 已启动，聚合间隔: %v", a.interval)
}

// Stop 停止聚合器
func (a *Aggregator) Stop() {
	close(a.stopCh)
	<-a.doneCh
	log.Println("[SLO Aggregator] 已停止")
}

func (a *Aggregator) run() {
	defer close(a.doneCh)

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	// 启动时立即执行一次
	a.aggregateAll()

	for {
		select {
		case <-ticker.C:
			a.aggregateAll()
		case <-a.stopCh:
			return
		}
	}
}

// aggregateAll 聚合所有集群的所有域名
func (a *Aggregator) aggregateAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 获取所有有数据的 host
	// 这里简化处理：聚合最近 2 小时的数据
	now := time.Now()
	hourStart := now.Truncate(time.Hour)
	prevHour := hourStart.Add(-time.Hour)

	// 聚合上一个完整小时
	if err := a.aggregateHour(ctx, prevHour); err != nil {
		log.Printf("[SLO Aggregator] 聚合 %v 失败: %v", prevHour, err)
	}

	// 聚合当前小时（实时）
	if err := a.aggregateHour(ctx, hourStart); err != nil {
		log.Printf("[SLO Aggregator] 聚合 %v 失败: %v", hourStart, err)
	}
}

// aggregateHour 聚合指定小时的数据
func (a *Aggregator) aggregateHour(ctx context.Context, hour time.Time) error {
	start := hour
	end := hour.Add(time.Hour)

	// 获取所有有数据的集群
	clusterIDs, err := a.repo.GetAllClusterIDs(ctx)
	if err != nil {
		log.Printf("[SLO Aggregator] 获取集群列表失败: %v", err)
		return err
	}

	for _, clusterID := range clusterIDs {
		// 获取该集群的所有 host
		hosts, err := a.repo.GetAllHosts(ctx, clusterID)
		if err != nil {
			log.Printf("[SLO Aggregator] 获取 hosts 失败: %v", err)
			continue
		}

		for _, host := range hosts {
			if err := a.aggregateHostHour(ctx, clusterID, host, start, end); err != nil {
				log.Printf("[SLO Aggregator] 聚合 %s/%s 失败: %v", clusterID, host, err)
			}
		}
	}

	return nil
}

// aggregateHostHour 聚合单个 host 的一个小时数据
func (a *Aggregator) aggregateHostHour(ctx context.Context, clusterID, host string, start, end time.Time) error {
	// 获取该小时的所有 raw 数据
	rawMetrics, err := a.repo.GetRawMetrics(ctx, clusterID, host, start, end)
	if err != nil {
		return err
	}

	if len(rawMetrics) == 0 {
		return nil
	}

	// 聚合数据
	hourly := a.aggregateRawToHourly(clusterID, host, start, rawMetrics)

	// 写入 hourly 表
	return a.repo.UpsertHourlyMetrics(ctx, hourly)
}

// aggregateRawToHourly 从 raw 数据聚合到 hourly
func (a *Aggregator) aggregateRawToHourly(clusterID, host string, hourStart time.Time, raws []*database.SLOMetricsRaw) *database.SLOMetricsHourly {
	var totalRequests, errorRequests int64
	var sumLatency int64
	buckets := make(map[float64]int64)

	// 从第一条 raw 中获取 domain/path（同一 host 的所有 raw 应该相同）
	var domain, pathPrefix string
	if len(raws) > 0 {
		domain = raws[0].Domain
		pathPrefix = raws[0].PathPrefix
	}
	if pathPrefix == "" {
		pathPrefix = "/"
	}

	for _, raw := range raws {
		totalRequests += raw.TotalRequests
		errorRequests += raw.ErrorRequests
		sumLatency += raw.SumLatencyMs

		// 累加 buckets
		buckets[0.005] += raw.Bucket5ms
		buckets[0.01] += raw.Bucket10ms
		buckets[0.025] += raw.Bucket25ms
		buckets[0.05] += raw.Bucket50ms
		buckets[0.1] += raw.Bucket100ms
		buckets[0.25] += raw.Bucket250ms
		buckets[0.5] += raw.Bucket500ms
		buckets[1.0] += raw.Bucket1s
		buckets[2.5] += raw.Bucket2500ms
		buckets[5.0] += raw.Bucket5s
		buckets[10.0] += raw.Bucket10s
		buckets[math.Inf(1)] += raw.BucketInf
	}

	// 计算指标
	availability := CalculateAvailability(totalRequests, errorRequests)
	p50 := CalculateQuantileMs(buckets, 0.50)
	p95 := CalculateQuantileMs(buckets, 0.95)
	p99 := CalculateQuantileMs(buckets, 0.99)

	var avgLatency int
	if totalRequests > 0 {
		avgLatency = int(sumLatency / totalRequests)
	}

	// 计算 RPS（假设每个 raw 是 10 秒间隔）
	durationSeconds := float64(len(raws)) * 10.0
	avgRPS := CalculateRPS(totalRequests, durationSeconds)

	rawBuckets := BucketsToRaw(buckets)

	return &database.SLOMetricsHourly{
		ClusterID:     clusterID,
		Host:          host,
		Domain:        domain,
		PathPrefix:    pathPrefix,
		HourStart:     hourStart,
		TotalRequests: totalRequests,
		ErrorRequests: errorRequests,
		Availability:  availability,
		P50LatencyMs:  p50,
		P95LatencyMs:  p95,
		P99LatencyMs:  p99,
		AvgLatencyMs:  avgLatency,
		AvgRPS:        avgRPS,
		Bucket5ms:     rawBuckets.Bucket5ms,
		Bucket10ms:    rawBuckets.Bucket10ms,
		Bucket25ms:    rawBuckets.Bucket25ms,
		Bucket50ms:    rawBuckets.Bucket50ms,
		Bucket100ms:   rawBuckets.Bucket100ms,
		Bucket250ms:   rawBuckets.Bucket250ms,
		Bucket500ms:   rawBuckets.Bucket500ms,
		Bucket1s:      rawBuckets.Bucket1s,
		Bucket2500ms:  rawBuckets.Bucket2500ms,
		Bucket5s:      rawBuckets.Bucket5s,
		Bucket10s:     rawBuckets.Bucket10s,
		BucketInf:     rawBuckets.BucketInf,
		SampleCount:   len(raws),
		CreatedAt:     time.Now(),
	}
}

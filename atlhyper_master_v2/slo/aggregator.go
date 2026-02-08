// atlhyper_master_v2/slo/aggregator.go
// SLO 小时聚合器
//
// 定时将 raw 表数据聚合为 hourly 表，支持三层数据：
// - service (服务网格 inbound)
// - edge (拓扑)
// - ingress (入口)
//
// 每次聚合覆盖：上一个完整小时 + 当前不完整小时
package slo

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"strconv"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// Aggregator 小时聚合器
type Aggregator struct {
	repo        database.SLORepository
	serviceRepo database.SLOServiceRepository
	edgeRepo    database.SLOEdgeRepository
	interval    time.Duration
	stopCh      chan struct{}
	doneCh      chan struct{}
}

// NewAggregator 创建聚合器
func NewAggregator(repo database.SLORepository, serviceRepo database.SLOServiceRepository, edgeRepo database.SLOEdgeRepository, interval time.Duration) *Aggregator {
	return &Aggregator{
		repo:        repo,
		serviceRepo: serviceRepo,
		edgeRepo:    edgeRepo,
		interval:    interval,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
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

// aggregateAll 聚合所有层的数据
// 覆盖上一个完整小时 + 当前不完整小时
func (a *Aggregator) aggregateAll() {
	now := time.Now().UTC()
	currentHour := now.Truncate(time.Hour)
	prevHour := currentHour.Add(-time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 聚合上一个完整小时
	a.aggregateHour(ctx, prevHour)

	// 聚合当前不完整小时（确保数据即时可见）
	a.aggregateHour(ctx, currentHour)
}

// aggregateHour 聚合指定小时的所有三层数据
func (a *Aggregator) aggregateHour(ctx context.Context, hourStart time.Time) {
	hourEnd := hourStart.Add(time.Hour)

	a.aggregateServiceHour(ctx, hourStart, hourEnd)
	a.aggregateEdgeHour(ctx, hourStart, hourEnd)
	a.aggregateIngressHour(ctx, hourStart, hourEnd)
}

// aggregateServiceHour 聚合服务网格 raw → hourly
func (a *Aggregator) aggregateServiceHour(ctx context.Context, hourStart, hourEnd time.Time) {
	// 获取所有集群
	clusterIDs, err := a.repo.GetAllClusterIDs(ctx)
	if err != nil {
		log.Printf("[SLO Aggregator] 获取集群列表失败: %v", err)
		return
	}

	for _, clusterID := range clusterIDs {
		// 查询该集群、该小时的所有 service raw 数据
		// 使用空 namespace/name 表示查所有服务
		raws, err := a.serviceRepo.GetServiceRaw(ctx, clusterID, "", "", hourStart, hourEnd)
		if err != nil {
			log.Printf("[SLO Aggregator] 查询 service raw 失败: cluster=%s, err=%v", clusterID, err)
			continue
		}
		if len(raws) == 0 {
			continue
		}

		// 按 (namespace, name) 分组
		type serviceKey struct {
			Namespace string
			Name      string
		}
		groups := map[serviceKey][]*database.SLOServiceRaw{}
		for _, r := range raws {
			k := serviceKey{r.Namespace, r.Name}
			groups[k] = append(groups[k], r)
		}

		for k, rows := range groups {
			hourly := a.aggregateServiceRows(clusterID, k.Namespace, k.Name, hourStart, rows)
			if err := a.serviceRepo.UpsertServiceHourly(ctx, hourly); err != nil {
				log.Printf("[SLO Aggregator] UPSERT service hourly 失败: %s/%s/%s err=%v", clusterID, k.Namespace, k.Name, err)
			}
		}
	}
}

// aggregateServiceRows 聚合一组 service raw 行为一条 hourly
func (a *Aggregator) aggregateServiceRows(clusterID, namespace, name string, hourStart time.Time, rows []*database.SLOServiceRaw) *database.SLOServiceHourly {
	var totalReqs, errorReqs int64
	var s2xx, s3xx, s4xx, s5xx int64
	var latencySum float64
	var latencyCount int64
	var tlsTotal, reqTotal int64

	var allBuckets []map[float64]int64

	for _, r := range rows {
		totalReqs += r.TotalRequests
		errorReqs += r.ErrorRequests
		s2xx += r.Status2xx
		s3xx += r.Status3xx
		s4xx += r.Status4xx
		s5xx += r.Status5xx
		latencySum += r.LatencySum
		latencyCount += r.LatencyCount
		tlsTotal += r.TLSRequestDelta
		reqTotal += r.TotalRequestDelta

		if b := ParseJSONBuckets(r.LatencyBuckets); b != nil {
			allBuckets = append(allBuckets, b)
		}
	}

	// 合并 bucket 并计算分位数
	merged := MergeBuckets(allBuckets...)
	p50 := CalculateQuantileMs(merged, 0.50)
	p95 := CalculateQuantileMs(merged, 0.95)
	p99 := CalculateQuantileMs(merged, 0.99)

	var avgLatency int
	if latencyCount > 0 {
		avgLatency = int(latencySum / float64(latencyCount))
	}

	var avgRPS float64
	if totalReqs > 0 {
		avgRPS = CalculateRPS(totalReqs, 3600)
	}

	avail := CalculateAvailability(totalReqs, errorReqs)

	var mtlsPercent float64
	if reqTotal > 0 {
		mtlsPercent = float64(tlsTotal) / float64(reqTotal) * 100
	}

	// 序列化合并后的 bucket
	var bucketJSON string
	if len(merged) > 0 {
		bucketJSON = marshalFloatBuckets(merged)
	}

	return &database.SLOServiceHourly{
		ClusterID:      clusterID,
		Namespace:      namespace,
		Name:           name,
		HourStart:      hourStart,
		TotalRequests:  totalReqs,
		ErrorRequests:  errorReqs,
		Availability:   avail,
		P50LatencyMs:   p50,
		P95LatencyMs:   p95,
		P99LatencyMs:   p99,
		AvgLatencyMs:   avgLatency,
		AvgRPS:         avgRPS,
		Status2xx:      s2xx,
		Status3xx:      s3xx,
		Status4xx:      s4xx,
		Status5xx:      s5xx,
		LatencyBuckets: bucketJSON,
		MtlsPercent:    mtlsPercent,
		SampleCount:    len(rows),
		CreatedAt:      time.Now(),
	}
}

// aggregateEdgeHour 聚合拓扑边 raw → hourly
func (a *Aggregator) aggregateEdgeHour(ctx context.Context, hourStart, hourEnd time.Time) {
	clusterIDs, err := a.repo.GetAllClusterIDs(ctx)
	if err != nil {
		return
	}

	for _, clusterID := range clusterIDs {
		raws, err := a.edgeRepo.GetEdgeRaw(ctx, clusterID, hourStart, hourEnd)
		if err != nil {
			log.Printf("[SLO Aggregator] 查询 edge raw 失败: cluster=%s, err=%v", clusterID, err)
			continue
		}
		if len(raws) == 0 {
			continue
		}

		// 按 (src_ns, src_name, dst_ns, dst_name) 分组
		type edgeKey struct {
			SrcNS, SrcName, DstNS, DstName string
		}
		groups := map[edgeKey][]*database.SLOEdgeRaw{}
		for _, r := range raws {
			k := edgeKey{r.SrcNamespace, r.SrcName, r.DstNamespace, r.DstName}
			groups[k] = append(groups[k], r)
		}

		for k, rows := range groups {
			hourly := a.aggregateEdgeRows(clusterID, k, hourStart, rows)
			if err := a.edgeRepo.UpsertEdgeHourly(ctx, hourly); err != nil {
				log.Printf("[SLO Aggregator] UPSERT edge hourly 失败: %s %s→%s err=%v", clusterID, k.SrcName, k.DstName, err)
			}
		}
	}
}

// aggregateEdgeRows 聚合一组 edge raw 行
func (a *Aggregator) aggregateEdgeRows(clusterID string, key struct{ SrcNS, SrcName, DstNS, DstName string }, hourStart time.Time, rows []*database.SLOEdgeRaw) *database.SLOEdgeHourly {
	var totalReqs, errorReqs int64
	var latencySum float64
	var latencyCount int64

	for _, r := range rows {
		totalReqs += r.RequestDelta
		errorReqs += r.FailureDelta
		latencySum += r.LatencySum
		latencyCount += r.LatencyCount
	}

	var avgLatency int
	if latencyCount > 0 {
		avgLatency = int(latencySum / float64(latencyCount))
	}

	avgRPS := CalculateRPS(totalReqs, 3600)
	errorRate := CalculateErrorRate(totalReqs, errorReqs)

	return &database.SLOEdgeHourly{
		ClusterID:     clusterID,
		SrcNamespace:  key.SrcNS,
		SrcName:       key.SrcName,
		DstNamespace:  key.DstNS,
		DstName:       key.DstName,
		HourStart:     hourStart,
		TotalRequests: totalReqs,
		ErrorRequests: errorReqs,
		AvgLatencyMs:  avgLatency,
		AvgRPS:        avgRPS,
		ErrorRate:     errorRate,
		SampleCount:   len(rows),
		CreatedAt:     time.Now(),
	}
}

// aggregateIngressHour 聚合入口 raw → hourly
func (a *Aggregator) aggregateIngressHour(ctx context.Context, hourStart, hourEnd time.Time) {
	clusterIDs, err := a.repo.GetAllClusterIDs(ctx)
	if err != nil {
		return
	}

	for _, clusterID := range clusterIDs {
		hosts, err := a.repo.GetAllHosts(ctx, clusterID)
		if err != nil {
			log.Printf("[SLO Aggregator] 获取 host 列表失败: cluster=%s, err=%v", clusterID, err)
			continue
		}

		for _, host := range hosts {
			raws, err := a.repo.GetRawMetrics(ctx, clusterID, host, hourStart, hourEnd)
			if err != nil {
				log.Printf("[SLO Aggregator] 查询 ingress raw 失败: cluster=%s host=%s err=%v", clusterID, host, err)
				continue
			}
			if len(raws) == 0 {
				continue
			}

			hourly := a.aggregateIngressRows(clusterID, host, hourStart, raws)
			if err := a.repo.UpsertHourlyMetrics(ctx, hourly); err != nil {
				log.Printf("[SLO Aggregator] UPSERT ingress hourly 失败: %s/%s err=%v", clusterID, host, err)
			}
		}
	}
}

// aggregateIngressRows 聚合一组 ingress raw 行
func (a *Aggregator) aggregateIngressRows(clusterID, host string, hourStart time.Time, rows []*database.SLOMetricsRaw) *database.SLOMetricsHourly {
	var totalReqs, errorReqs int64
	var latencySum float64
	var latencyCount int64
	var mGet, mPost, mPut, mDelete, mOther int64
	var s2xx, s3xx, s4xx, s5xx int64

	var allBuckets []map[float64]int64
	var domain, pathPrefix string

	for _, r := range rows {
		totalReqs += r.TotalRequests
		errorReqs += r.ErrorRequests
		latencySum += r.LatencySum
		latencyCount += r.LatencyCount
		mGet += r.MethodGet
		mPost += r.MethodPost
		mPut += r.MethodPut
		mDelete += r.MethodDelete
		mOther += r.MethodOther
		s2xx += r.Status2xx
		s3xx += r.Status3xx
		s4xx += r.Status4xx
		s5xx += r.Status5xx

		if b := ParseJSONBuckets(r.LatencyBuckets); b != nil {
			allBuckets = append(allBuckets, b)
		}

		// 取第一个非空的 domain/path
		if domain == "" && r.Domain != "" {
			domain = r.Domain
			pathPrefix = r.PathPrefix
		}
	}

	merged := MergeBuckets(allBuckets...)
	p50 := CalculateQuantileMs(merged, 0.50)
	p95 := CalculateQuantileMs(merged, 0.95)
	p99 := CalculateQuantileMs(merged, 0.99)

	var avgLatency int
	if latencyCount > 0 {
		avgLatency = int(latencySum / float64(latencyCount))
	}

	avgRPS := CalculateRPS(totalReqs, 3600)
	avail := CalculateAvailability(totalReqs, errorReqs)

	var bucketJSON string
	if len(merged) > 0 {
		bucketJSON = marshalFloatBuckets(merged)
	}

	return &database.SLOMetricsHourly{
		ClusterID:      clusterID,
		Host:           host,
		Domain:         domain,
		PathPrefix:     pathPrefix,
		HourStart:      hourStart,
		TotalRequests:  totalReqs,
		ErrorRequests:  errorReqs,
		Availability:   avail,
		P50LatencyMs:   p50,
		P95LatencyMs:   p95,
		P99LatencyMs:   p99,
		AvgLatencyMs:   avgLatency,
		AvgRPS:         avgRPS,
		LatencyBuckets: bucketJSON,
		MethodGet:      mGet,
		MethodPost:     mPost,
		MethodPut:      mPut,
		MethodDelete:   mDelete,
		MethodOther:    mOther,
		Status2xx:      s2xx,
		Status3xx:      s3xx,
		Status4xx:      s4xx,
		Status5xx:      s5xx,
		SampleCount:    len(rows),
		CreatedAt:      time.Now(),
	}
}

// marshalFloatBuckets 将 float64 key 的 bucket map 序列化为 JSON 字符串
// key 从秒转回毫秒字符串，保持存储格式一致
func marshalFloatBuckets(buckets map[float64]int64) string {
	if len(buckets) == 0 {
		return ""
	}
	result := make(map[string]int64, len(buckets))
	for le, count := range buckets {
		if math.IsInf(le, 1) {
			result["+Inf"] = count
			continue
		}
		ms := le * 1000
		if ms == float64(int64(ms)) {
			result[strconv.FormatInt(int64(ms), 10)] = count
		} else {
			result[strconv.FormatFloat(ms, 'f', -1, 64)] = count
		}
	}
	data, err := json.Marshal(result)
	if err != nil {
		return ""
	}
	return string(data)
}

package query

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/slo"
)

// sloRepository SLO 查询仓库
type sloRepository struct {
	client sdk.ClickHouseClient
}

// NewSLOQueryRepository 创建 SLO 查询仓库
func NewSLOQueryRepository(client sdk.ClickHouseClient) repository.SLOQueryRepository {
	return &sloRepository{client: client}
}

// ListIngressSLO 查询 Traefik 入口 SLO
func (r *sloRepository) ListIngressSLO(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	sec := sinceSeconds(since)

	// 请求计数（按 service + code 分组）
	countQuery := fmt.Sprintf(`
		SELECT Attributes['service'] AS svc,
		       Attributes['code'] AS code,
		       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) AS delta
		FROM otel_metrics_sum
		WHERE MetricName = 'traefik_service_requests_total'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY svc, code
		HAVING count() >= 2
	`, sec)

	rows, err := r.client.Query(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("query ingress counts: %w", err)
	}
	defer rows.Close()

	type svcData struct {
		totalReqs   int64
		totalErrors int64
		codes       map[string]int64
	}
	svcMap := make(map[string]*svcData)

	for rows.Next() {
		var svcKey, code string
		var delta float64
		if err := rows.Scan(&svcKey, &code, &delta); err != nil {
			continue
		}
		d, ok := svcMap[svcKey]
		if !ok {
			d = &svcData{codes: make(map[string]int64)}
			svcMap[svcKey] = d
		}
		cnt := int64(delta)
		d.totalReqs += cnt
		d.codes[code] += cnt
		if len(code) > 0 && (code[0] == '4' || code[0] == '5') {
			d.totalErrors += cnt
		}
	}

	// 延迟分位数 (histogram)
	latencyQuery := fmt.Sprintf(`
		SELECT Attributes['service'] AS svc,
		       ExplicitBounds,
		       BucketCounts
		FROM otel_metrics_histogram
		WHERE MetricName = 'traefik_service_request_duration_seconds'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		ORDER BY svc, TimeUnix DESC
	`, sec)

	// 由于 histogram 聚合复杂，每个 service 取最新一条的 bounds/counts
	latencyRows, err := r.client.Query(ctx, latencyQuery)
	if err != nil {
		// 延迟查询失败不影响主数据
		latencyRows = nil
	}

	type latencyData struct {
		p50, p90, p95, p99, avg float64
		buckets                 []slo.LatencyBucket // 原始桶数据
	}
	latencyMap := make(map[string]*latencyData)

	if latencyRows != nil {
		defer latencyRows.Close()
		seen := make(map[string]bool)
		for latencyRows.Next() {
			var svcKey string
			var bounds []float64
			var counts []uint64
			if err := latencyRows.Scan(&svcKey, &bounds, &counts); err != nil {
				continue
			}
			if seen[svcKey] {
				continue
			}
			seen[svcKey] = true

			// 构建延迟分布桶（bounds 从秒→毫秒）
			var buckets []slo.LatencyBucket
			for i, b := range bounds {
				var cnt int64
				if i < len(counts) {
					cnt = int64(counts[i])
				}
				buckets = append(buckets, slo.LatencyBucket{
					LE:    roundTo(b*1000, 2),
					Count: cnt,
				})
			}
			// +Inf 桶
			if len(counts) > len(bounds) {
				buckets = append(buckets, slo.LatencyBucket{
					LE:    0, // 0 表示 +Inf
					Count: int64(counts[len(bounds)]),
				})
			}

			latencyMap[svcKey] = &latencyData{
				p50:     roundTo(histogramPercentile(bounds, counts, 0.50)*1000, 2),
				p90:     roundTo(histogramPercentile(bounds, counts, 0.90)*1000, 2),
				p95:     roundTo(histogramPercentile(bounds, counts, 0.95)*1000, 2),
				p99:     roundTo(histogramPercentile(bounds, counts, 0.99)*1000, 2),
				buckets: buckets,
			}
		}
	}

	// HTTP 方法分布查询
	methodQuery := fmt.Sprintf(`
		SELECT Attributes['service'] AS svc,
		       Attributes['method'] AS method,
		       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) AS delta
		FROM otel_metrics_sum
		WHERE MetricName = 'traefik_service_requests_total'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY svc, method
		HAVING count() >= 2
	`, sec)

	methodMap := make(map[string][]slo.MethodCount)
	methodRows, methodErr := r.client.Query(ctx, methodQuery)
	if methodErr == nil && methodRows != nil {
		defer methodRows.Close()
		for methodRows.Next() {
			var svcKey, method string
			var delta float64
			if err := methodRows.Scan(&svcKey, &method, &delta); err != nil {
				continue
			}
			if method == "" {
				continue
			}
			cnt := int64(delta)
			if cnt > 0 {
				methodMap[svcKey] = append(methodMap[svcKey], slo.MethodCount{Method: method, Count: cnt})
			}
		}
	}

	// 组装结果
	duration := float64(sec)
	var result []slo.IngressSLO
	for key, d := range svcMap {
		item := slo.IngressSLO{
			ServiceKey:    key,
			DisplayName:   key,
			TotalRequests: d.totalReqs,
			TotalErrors:   d.totalErrors,
			RPS:           roundTo(float64(d.totalReqs)/duration, 2),
		}
		if d.totalReqs > 0 {
			item.SuccessRate = roundTo(float64(d.totalReqs-d.totalErrors)/float64(d.totalReqs)*100, 2)
			item.ErrorRate = roundTo(float64(d.totalErrors)/float64(d.totalReqs)*100, 2)
		}
		for code, cnt := range d.codes {
			item.StatusCodes = append(item.StatusCodes, slo.StatusCodeCount{Code: code, Count: cnt})
		}
		if lat, ok := latencyMap[key]; ok {
			item.P50Ms = lat.p50
			item.P90Ms = lat.p90
			item.P95Ms = lat.p95
			item.P99Ms = lat.p99
			item.LatencyBuckets = lat.buckets
		}
		if methods, ok := methodMap[key]; ok {
			item.Methods = methods
		}
		result = append(result, item)
	}
	if result == nil {
		result = []slo.IngressSLO{}
	}
	return result, nil
}

// ListServiceSLO 查询 Linkerd 服务网格 SLO
func (r *sloRepository) ListServiceSLO(ctx context.Context, since time.Duration) ([]slo.ServiceSLO, error) {
	sec := sinceSeconds(since)

	// response_total (gauge) — 按 deployment/namespace/status_code/tls 分组
	query := fmt.Sprintf(`
		SELECT Attributes['deployment'] AS deploy,
		       Attributes['namespace'] AS ns,
		       Attributes['status_code'] AS code,
		       Attributes['tls'] AS tls,
		       sum(Value) AS total
		FROM otel_metrics_gauge
		WHERE MetricName = 'response_total'
		  AND Attributes['direction'] = 'inbound'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY deploy, ns, code, tls
	`, sec)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query service SLO: %w", err)
	}
	defer rows.Close()

	type svcKey struct {
		name, ns string
	}
	type svcData struct {
		totalReqs   float64
		successReqs float64
		tlsReqs     float64
		codes       map[string]int64
	}
	svcMap := make(map[svcKey]*svcData)

	for rows.Next() {
		var deploy, ns, code, tls string
		var total float64
		if err := rows.Scan(&deploy, &ns, &code, &tls, &total); err != nil {
			continue
		}
		key := svcKey{deploy, ns}
		d, ok := svcMap[key]
		if !ok {
			d = &svcData{codes: make(map[string]int64)}
			svcMap[key] = d
		}
		d.totalReqs += total
		if code == "200" || code == "201" || code == "204" || (len(code) > 0 && code[0] == '2') {
			d.successReqs += total
		}
		if tls == "true" {
			d.tlsReqs += total
		}
		d.codes[code] += int64(total)
	}

	// Linkerd latency buckets
	latQuery := fmt.Sprintf(`
		SELECT Attributes['deployment'] AS deploy,
		       Attributes['namespace'] AS ns,
		       Attributes['le'] AS le,
		       sum(Value) AS total
		FROM otel_metrics_gauge
		WHERE MetricName = 'response_latency_ms_bucket'
		  AND Attributes['direction'] = 'inbound'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY deploy, ns, le
		ORDER BY deploy, ns, toFloat64OrNull(le)
	`, sec)

	latRows, err := r.client.Query(ctx, latQuery)
	if err != nil {
		latRows = nil
	}

	type latKey struct {
		name, ns string
	}
	latBuckets := make(map[latKey]struct {
		bounds []float64
		counts []uint64
	})

	if latRows != nil {
		defer latRows.Close()
		for latRows.Next() {
			var deploy, ns, le string
			var total float64
			if err := latRows.Scan(&deploy, &ns, &le, &total); err != nil {
				continue
			}
			key := latKey{deploy, ns}
			data := latBuckets[key]
			var bound float64
			if le == "+Inf" {
				// 最后一个桶
			} else {
				fmt.Sscanf(le, "%f", &bound)
				data.bounds = append(data.bounds, bound)
			}
			data.counts = append(data.counts, uint64(total))
			latBuckets[key] = data
		}
	}

	duration := float64(sec)
	var result []slo.ServiceSLO
	for key, d := range svcMap {
		item := slo.ServiceSLO{
			Namespace: key.ns,
			Name:      key.name,
			RPS:       roundTo(d.totalReqs/duration, 2),
		}
		if d.totalReqs > 0 {
			item.SuccessRate = roundTo(d.successReqs/d.totalReqs*100, 2)
			item.MTLSRate = roundTo(d.tlsReqs/d.totalReqs*100, 2)
		}
		for code, cnt := range d.codes {
			item.StatusCodes = append(item.StatusCodes, slo.StatusCodeCount{Code: code, Count: cnt})
		}
		lk := latKey{key.name, key.ns}
		if lb, ok := latBuckets[lk]; ok && len(lb.counts) > 0 {
			item.P50Ms = roundTo(histogramPercentile(lb.bounds, lb.counts, 0.50), 2)
			item.P90Ms = roundTo(histogramPercentile(lb.bounds, lb.counts, 0.90), 2)
			item.P99Ms = roundTo(histogramPercentile(lb.bounds, lb.counts, 0.99), 2)
		}
		result = append(result, item)
	}
	if result == nil {
		result = []slo.ServiceSLO{}
	}
	return result, nil
}

// ListServiceEdges 查询 Linkerd 服务间调用拓扑
func (r *sloRepository) ListServiceEdges(ctx context.Context, since time.Duration) ([]slo.ServiceEdge, error) {
	sec := sinceSeconds(since)

	query := fmt.Sprintf(`
		SELECT Attributes['deployment'] AS src,
		       Attributes['namespace'] AS src_ns,
		       Attributes['dst_deployment'] AS dst,
		       Attributes['dst_namespace'] AS dst_ns,
		       Attributes['status_code'] AS code,
		       sum(Value) AS total
		FROM otel_metrics_gauge
		WHERE MetricName = 'response_total'
		  AND Attributes['direction'] = 'outbound'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY src, src_ns, dst, dst_ns, code
	`, sec)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query service edges: %w", err)
	}
	defer rows.Close()

	type edgeKey struct {
		srcNs, src, dstNs, dst string
	}
	type edgeData struct {
		total, success float64
	}
	edgeMap := make(map[edgeKey]*edgeData)

	for rows.Next() {
		var src, srcNs, dst, dstNs, code string
		var total float64
		if err := rows.Scan(&src, &srcNs, &dst, &dstNs, &code, &total); err != nil {
			continue
		}
		key := edgeKey{srcNs, src, dstNs, dst}
		d, ok := edgeMap[key]
		if !ok {
			d = &edgeData{}
			edgeMap[key] = d
		}
		d.total += total
		if len(code) > 0 && code[0] == '2' {
			d.success += total
		}
	}

	duration := float64(sec)
	var result []slo.ServiceEdge
	for key, d := range edgeMap {
		edge := slo.ServiceEdge{
			SrcNamespace: key.srcNs,
			SrcName:      key.src,
			DstNamespace: key.dstNs,
			DstName:      key.dst,
			RPS:          roundTo(d.total/duration, 2),
		}
		if d.total > 0 {
			edge.SuccessRate = roundTo(d.success/d.total*100, 2)
		}
		result = append(result, edge)
	}
	if result == nil {
		result = []slo.ServiceEdge{}
	}
	return result, nil
}

// GetSLOTimeSeries 查询 SLO 时序数据
func (r *sloRepository) GetSLOTimeSeries(ctx context.Context, name string, since time.Duration) (*slo.TimeSeries, error) {
	sec := sinceSeconds(since)

	// 按 5 分钟窗口聚合
	query := fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL 300 SECOND) AS ts,
		       Attributes['status_code'] AS code,
		       sum(Value) AS total
		FROM otel_metrics_gauge
		WHERE MetricName = 'response_total'
		  AND Attributes['direction'] = 'inbound'
		  AND (Attributes['deployment'] = ? OR Attributes['service'] = ?)
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ts, code
		ORDER BY ts
	`, sec)

	rows, err := r.client.Query(ctx, query, name, name)
	if err != nil {
		return nil, fmt.Errorf("query SLO time series: %w", err)
	}
	defer rows.Close()

	type tsData struct {
		total, success float64
	}
	tsMap := make(map[time.Time]*tsData)

	for rows.Next() {
		var ts time.Time
		var code string
		var total float64
		if err := rows.Scan(&ts, &code, &total); err != nil {
			continue
		}
		d, ok := tsMap[ts]
		if !ok {
			d = &tsData{}
			tsMap[ts] = d
		}
		d.total += total
		if len(code) > 0 && code[0] == '2' {
			d.success += total
		}
	}

	ts := &slo.TimeSeries{Name: name}
	for t, d := range tsMap {
		dp := slo.DataPoint{
			Timestamp: t,
			RPS:       roundTo(d.total/300, 2), // 5 分钟窗口
		}
		if d.total > 0 {
			dp.SuccessRate = roundTo(d.success/d.total*100, 2)
		}
		ts.Points = append(ts.Points, dp)
	}
	if ts.Points == nil {
		ts.Points = []slo.DataPoint{}
	}
	return ts, nil
}

// GetSLOSummary 获取 SLO 仪表盘摘要
func (r *sloRepository) GetSLOSummary(ctx context.Context) (*slo.SLOSummary, error) {
	since := 5 * time.Minute

	// 同时获取 Ingress 和 Service SLO
	type ingressResult struct {
		data []slo.IngressSLO
		err  error
	}
	type serviceResult struct {
		data []slo.ServiceSLO
		err  error
	}

	ingCh := make(chan ingressResult, 1)
	svcCh := make(chan serviceResult, 1)

	go func() {
		data, err := r.ListIngressSLO(ctx, since)
		ingCh <- ingressResult{data, err}
	}()
	go func() {
		data, err := r.ListServiceSLO(ctx, since)
		svcCh <- serviceResult{data, err}
	}()

	ingRes := <-ingCh
	svcRes := <-svcCh

	summary := &slo.SLOSummary{}

	// 合并统计
	var totalSuccRate, totalRPS, totalP99 float64
	var count int

	if ingRes.err == nil {
		for _, s := range ingRes.data {
			count++
			totalSuccRate += s.SuccessRate
			totalRPS += s.RPS
			totalP99 += s.P99Ms

			if s.SuccessRate >= 99.9 {
				summary.HealthyServices++
			} else if s.SuccessRate >= 99.0 {
				summary.WarningServices++
			} else {
				summary.CriticalServices++
			}
		}
	}

	if svcRes.err == nil {
		for _, s := range svcRes.data {
			count++
			totalSuccRate += s.SuccessRate
			totalRPS += s.RPS
			totalP99 += s.P99Ms

			if s.SuccessRate >= 99.9 {
				summary.HealthyServices++
			} else if s.SuccessRate >= 99.0 {
				summary.WarningServices++
			} else {
				summary.CriticalServices++
			}
		}
	}

	summary.TotalServices = count
	if count > 0 {
		summary.AvgSuccessRate = roundTo(totalSuccRate/float64(count), 2)
		summary.TotalRPS = roundTo(totalRPS, 2)
		summary.AvgP99Ms = roundTo(totalP99/float64(count), 2)
	}

	return summary, nil
}

// ListIngressSLOPrevious 查询上一周期的 Traefik 入口 SLO
// since 表示窗口大小，查询 [now-2*since, now-since) 的数据
func (r *sloRepository) ListIngressSLOPrevious(ctx context.Context, since time.Duration) ([]slo.IngressSLO, error) {
	sec := sinceSeconds(since)

	// 请求计数（上一周期）
	countQuery := fmt.Sprintf(`
		SELECT Attributes['service'] AS svc,
		       Attributes['code'] AS code,
		       (argMax(Value, TimeUnix) - argMin(Value, TimeUnix)) AS delta
		FROM otel_metrics_sum
		WHERE MetricName = 'traefik_service_requests_total'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		  AND TimeUnix < now() - INTERVAL %d SECOND
		GROUP BY svc, code
		HAVING count() >= 2
	`, 2*sec, sec)

	rows, err := r.client.Query(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("query ingress counts (previous): %w", err)
	}
	defer rows.Close()

	type svcData struct {
		totalReqs   int64
		totalErrors int64
		codes       map[string]int64
	}
	svcMap := make(map[string]*svcData)

	for rows.Next() {
		var svcKey, code string
		var delta float64
		if err := rows.Scan(&svcKey, &code, &delta); err != nil {
			continue
		}
		d, ok := svcMap[svcKey]
		if !ok {
			d = &svcData{codes: make(map[string]int64)}
			svcMap[svcKey] = d
		}
		cnt := int64(delta)
		d.totalReqs += cnt
		d.codes[code] += cnt
		if len(code) > 0 && (code[0] == '4' || code[0] == '5') {
			d.totalErrors += cnt
		}
	}

	// 延迟分位数（上一周期）
	latencyQuery := fmt.Sprintf(`
		SELECT Attributes['service'] AS svc,
		       ExplicitBounds,
		       BucketCounts
		FROM otel_metrics_histogram
		WHERE MetricName = 'traefik_service_request_duration_seconds'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		  AND TimeUnix < now() - INTERVAL %d SECOND
		ORDER BY svc, TimeUnix DESC
	`, 2*sec, sec)

	latencyRows, lerr := r.client.Query(ctx, latencyQuery)
	if lerr != nil {
		latencyRows = nil
	}

	type latencyData struct {
		p50, p90, p95, p99 float64
	}
	latencyMap := make(map[string]*latencyData)

	if latencyRows != nil {
		defer latencyRows.Close()
		seen := make(map[string]bool)
		for latencyRows.Next() {
			var svcKey string
			var bounds []float64
			var counts []uint64
			if err := latencyRows.Scan(&svcKey, &bounds, &counts); err != nil {
				continue
			}
			if seen[svcKey] {
				continue
			}
			seen[svcKey] = true
			latencyMap[svcKey] = &latencyData{
				p50: roundTo(histogramPercentile(bounds, counts, 0.50)*1000, 2),
				p90: roundTo(histogramPercentile(bounds, counts, 0.90)*1000, 2),
				p95: roundTo(histogramPercentile(bounds, counts, 0.95)*1000, 2),
				p99: roundTo(histogramPercentile(bounds, counts, 0.99)*1000, 2),
			}
		}
	}

	duration := float64(sec)
	var result []slo.IngressSLO
	for key, d := range svcMap {
		item := slo.IngressSLO{
			ServiceKey:    key,
			DisplayName:   key,
			TotalRequests: d.totalReqs,
			TotalErrors:   d.totalErrors,
			RPS:           roundTo(float64(d.totalReqs)/duration, 2),
		}
		if d.totalReqs > 0 {
			item.SuccessRate = roundTo(float64(d.totalReqs-d.totalErrors)/float64(d.totalReqs)*100, 2)
			item.ErrorRate = roundTo(float64(d.totalErrors)/float64(d.totalReqs)*100, 2)
		}
		for code, cnt := range d.codes {
			item.StatusCodes = append(item.StatusCodes, slo.StatusCodeCount{Code: code, Count: cnt})
		}
		if lat, ok := latencyMap[key]; ok {
			item.P50Ms = lat.p50
			item.P90Ms = lat.p90
			item.P95Ms = lat.p95
			item.P99Ms = lat.p99
		}
		result = append(result, item)
	}
	if result == nil {
		result = []slo.IngressSLO{}
	}
	return result, nil
}

// GetIngressSLOHistory 查询 Ingress SLO 时序数据
// since: 总时间范围, bucket: 每个桶的时间跨度
func (r *sloRepository) GetIngressSLOHistory(ctx context.Context, since, bucket time.Duration) ([]slo.SLOHistoryPoint, error) {
	sec := sinceSeconds(since)
	bucketSec := sinceSeconds(bucket)

	// 请求计数时序
	countQuery := fmt.Sprintf(`
		SELECT toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       Attributes['service'] AS svc,
		       Attributes['code'] AS code,
		       (max(Value) - min(Value)) AS delta
		FROM otel_metrics_sum
		WHERE MetricName = 'traefik_service_requests_total'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY ts, svc, code
		HAVING count() >= 2
		ORDER BY ts
	`, bucketSec, sec)

	rows, err := r.client.Query(ctx, countQuery)
	if err != nil {
		return nil, fmt.Errorf("query ingress history counts: %w", err)
	}
	defer rows.Close()

	type bucketKey struct {
		ts  time.Time
		svc string
	}
	type bucketData struct {
		totalReqs   int64
		totalErrors int64
	}
	dataMap := make(map[bucketKey]*bucketData)

	for rows.Next() {
		var ts time.Time
		var svcKey, code string
		var delta float64
		if err := rows.Scan(&ts, &svcKey, &code, &delta); err != nil {
			continue
		}
		key := bucketKey{ts: ts, svc: svcKey}
		d, ok := dataMap[key]
		if !ok {
			d = &bucketData{}
			dataMap[key] = d
		}
		cnt := int64(delta)
		d.totalReqs += cnt
		if len(code) > 0 && (code[0] == '4' || code[0] == '5') {
			d.totalErrors += cnt
		}
	}

	// 延迟时序（取每桶最新 histogram）
	latencyQuery := fmt.Sprintf(`
		SELECT Attributes['service'] AS svc,
		       toStartOfInterval(TimeUnix, INTERVAL %d SECOND) AS ts,
		       argMax(ExplicitBounds, TimeUnix) AS bounds,
		       argMax(BucketCounts, TimeUnix) AS counts
		FROM otel_metrics_histogram
		WHERE MetricName = 'traefik_service_request_duration_seconds'
		  AND TimeUnix >= now() - INTERVAL %d SECOND
		GROUP BY svc, ts
		ORDER BY svc, ts
	`, bucketSec, sec)

	type latencyPoint struct {
		p95, p99 float64
	}
	latencyByBucket := make(map[bucketKey]*latencyPoint)

	latRows, latErr := r.client.Query(ctx, latencyQuery)
	if latErr == nil && latRows != nil {
		defer latRows.Close()
		for latRows.Next() {
			var svcKey string
			var ts time.Time
			var bounds []float64
			var counts []uint64
			if err := latRows.Scan(&svcKey, &ts, &bounds, &counts); err != nil {
				continue
			}
			key := bucketKey{ts: ts, svc: svcKey}
			latencyByBucket[key] = &latencyPoint{
				p95: roundTo(histogramPercentile(bounds, counts, 0.95)*1000, 2),
				p99: roundTo(histogramPercentile(bounds, counts, 0.99)*1000, 2),
			}
		}
	}

	// 组装结果
	bucketDuration := float64(bucketSec)
	var result []slo.SLOHistoryPoint
	for key, d := range dataMap {
		point := slo.SLOHistoryPoint{
			Timestamp:     key.ts,
			ServiceKey:    key.svc,
			TotalRequests: d.totalReqs,
			RPS:           roundTo(float64(d.totalReqs)/bucketDuration, 2),
		}
		if d.totalReqs > 0 {
			point.Availability = roundTo(float64(d.totalReqs-d.totalErrors)/float64(d.totalReqs)*100, 2)
			point.ErrorRate = roundTo(float64(d.totalErrors)/float64(d.totalReqs)*100, 2)
		}
		if lat, ok := latencyByBucket[key]; ok {
			point.P95Ms = lat.p95
			point.P99Ms = lat.p99
		}
		result = append(result, point)
	}
	if result == nil {
		result = []slo.SLOHistoryPoint{}
	}
	return result, nil
}

// atlhyper_master_v2/service/query/slo.go
// SLO 查询实现（服务网格 + 域名增强）
//
// 查询策略: hourly 优先 → raw 回退
// Handler（Gateway 层）通过 service.Query 接口调用，不直接访问 Database。
package query

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/slo"
)

// GetMeshTopology 获取服务网格拓扑
func (q *QueryService) GetMeshTopology(ctx context.Context, clusterID, timeRange string) (*model.ServiceMeshTopologyResponse, error) {
	if q.serviceRepo == nil || q.edgeRepo == nil {
		return &model.ServiceMeshTopologyResponse{}, nil
	}

	now := time.Now()
	start := getTimeStart(now, timeRange)
	end := now

	// 1. 获取服务节点
	nodes, err := q.getServiceNodes(ctx, clusterID, start, end)
	if err != nil {
		return nil, fmt.Errorf("获取服务节点失败: %w", err)
	}

	// 2. 获取拓扑边
	edges, err := q.getServiceEdges(ctx, clusterID, start, end)
	if err != nil {
		return nil, fmt.Errorf("获取拓扑边失败: %w", err)
	}

	return &model.ServiceMeshTopologyResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// GetServiceDetail 获取单个服务详情
func (q *QueryService) GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error) {
	if q.serviceRepo == nil || q.edgeRepo == nil {
		return nil, nil
	}

	now := time.Now()
	start := getTimeStart(now, timeRange)
	end := now

	// 1. 获取服务节点数据
	node, err := q.getServiceNodeDetail(ctx, clusterID, namespace, name, start, end)
	if err != nil {
		return nil, fmt.Errorf("获取服务数据失败: %w", err)
	}
	if node == nil {
		return nil, nil
	}

	resp := &model.ServiceDetailResponse{
		ServiceNodeResponse: *node,
	}

	// 2. 获取历史数据点（从 hourly）
	hourlies, err := q.serviceRepo.GetServiceHourly(ctx, clusterID, namespace, name, start, end)
	if err == nil {
		for _, h := range hourlies {
			resp.History = append(resp.History, model.ServiceHistoryPoint{
				Timestamp:    h.HourStart.Format(time.RFC3339),
				RPS:          h.AvgRPS,
				P95LatencyMs: h.P95LatencyMs,
				ErrorRate:    slo.CalculateErrorRate(h.TotalRequests, h.ErrorRequests),
				Availability: h.Availability,
				MtlsPercent:  h.MtlsPercent,
			})
		}
	}

	// 3. 获取上下游边
	allEdges, err := q.getServiceEdges(ctx, clusterID, start, end)
	if err == nil {
		serviceID := namespace + "/" + name
		for _, e := range allEdges {
			if e.Target == serviceID {
				resp.Upstreams = append(resp.Upstreams, e)
			}
			if e.Source == serviceID {
				resp.Downstreams = append(resp.Downstreams, e)
			}
		}
	}

	return resp, nil
}

// getServiceNodes 获取所有服务节点（hourly 优先，raw 回退）
func (q *QueryService) getServiceNodes(ctx context.Context, clusterID string, start, end time.Time) ([]model.ServiceNodeResponse, error) {
	// 尝试 hourly
	hourlies, err := q.serviceRepo.GetServiceHourly(ctx, clusterID, "", "", start, end)
	if err != nil {
		return nil, err
	}

	if len(hourlies) > 0 {
		// 按 (namespace, name) 聚合多个小时
		type key struct{ NS, Name string }
		groups := map[key][]struct {
			TotalReqs, ErrorReqs int64
			RPS                  float64
			P95, P99, Avg        int
			MtlsPercent          float64
		}{}
		totals := map[key]struct {
			TotalReqs, ErrorReqs int64
			LatencyWeight        float64
			P95Sum, P99Sum       float64
			AvgSum               float64
			RPSSum               float64
			MtlsSum              float64
			Count                int
		}{}

		for _, h := range hourlies {
			k := key{h.Namespace, h.Name}
			t := totals[k]
			t.TotalReqs += h.TotalRequests
			t.ErrorReqs += h.ErrorRequests
			t.P95Sum += float64(h.P95LatencyMs) * float64(h.TotalRequests)
			t.P99Sum += float64(h.P99LatencyMs) * float64(h.TotalRequests)
			t.AvgSum += float64(h.AvgLatencyMs) * float64(h.TotalRequests)
			t.RPSSum += h.AvgRPS
			t.MtlsSum += h.MtlsPercent
			t.Count++
			totals[k] = t
			_ = groups // unused
		}

		var nodes []model.ServiceNodeResponse
		for k, t := range totals {
			var p95, p99, avg int
			if t.TotalReqs > 0 {
				p95 = int(t.P95Sum / float64(t.TotalReqs))
				p99 = int(t.P99Sum / float64(t.TotalReqs))
				avg = int(t.AvgSum / float64(t.TotalReqs))
			}
			avail := slo.CalculateAvailability(t.TotalReqs, t.ErrorReqs)
			errRate := slo.CalculateErrorRate(t.TotalReqs, t.ErrorReqs)
			rps := t.RPSSum / float64(t.Count)
			mtls := t.MtlsSum / float64(t.Count)

			status := "healthy"
			if errRate > 5 {
				status = "critical"
			} else if errRate > 1 || p95 > 500 {
				status = "warning"
			}

			nodes = append(nodes, model.ServiceNodeResponse{
				ID:            k.NS + "/" + k.Name,
				Name:          k.Name,
				Namespace:     k.NS,
				RPS:           rps,
				AvgLatencyMs:  avg,
				P50LatencyMs:  0, // hourly 聚合中可用
				P95LatencyMs:  p95,
				P99LatencyMs:  p99,
				ErrorRate:     errRate,
				Availability:  avail,
				Status:        status,
				MtlsPercent:   mtls,
				TotalRequests: t.TotalReqs,
			})
		}
		return nodes, nil
	}

	// 回退到 raw
	raws, err := q.serviceRepo.GetServiceRaw(ctx, clusterID, "", "", start, end)
	if err != nil {
		return nil, err
	}

	type key struct{ NS, Name string }
	groups := map[key]struct {
		TotalReqs, ErrorReqs int64
		LatencySum           float64
		LatencyCount         int64
		TLS, Total           int64
		Buckets              []map[float64]int64
		SampleCount          int
	}{}

	for _, r := range raws {
		k := key{r.Namespace, r.Name}
		g := groups[k]
		g.TotalReqs += r.TotalRequests
		g.ErrorReqs += r.ErrorRequests
		g.LatencySum += r.LatencySum
		g.LatencyCount += r.LatencyCount
		g.TLS += r.TLSRequestDelta
		g.Total += r.TotalRequestDelta
		if b := slo.ParseJSONBuckets(r.LatencyBuckets); b != nil {
			g.Buckets = append(g.Buckets, b)
		}
		g.SampleCount++
		groups[k] = g
	}

	var nodes []model.ServiceNodeResponse
	for k, g := range groups {
		merged := slo.MergeBuckets(g.Buckets...)
		p95 := slo.CalculateQuantileMs(merged, 0.95)
		p99 := slo.CalculateQuantileMs(merged, 0.99)
		var avg int
		if g.LatencyCount > 0 {
			avg = int(g.LatencySum / float64(g.LatencyCount))
		}
		avail := slo.CalculateAvailability(g.TotalReqs, g.ErrorReqs)
		errRate := slo.CalculateErrorRate(g.TotalReqs, g.ErrorReqs)
		durationSec := float64(g.SampleCount) * 10.0
		rps := slo.CalculateRPS(g.TotalReqs, durationSec)

		var mtls float64
		if g.Total > 0 {
			mtls = float64(g.TLS) / float64(g.Total) * 100
		}

		status := "healthy"
		if errRate > 5 {
			status = "critical"
		} else if errRate > 1 || p95 > 500 {
			status = "warning"
		}

		nodes = append(nodes, model.ServiceNodeResponse{
			ID:            k.NS + "/" + k.Name,
			Name:          k.Name,
			Namespace:     k.NS,
			RPS:           rps,
			AvgLatencyMs:  avg,
			P95LatencyMs:  p95,
			P99LatencyMs:  p99,
			ErrorRate:     errRate,
			Availability:  avail,
			Status:        status,
			MtlsPercent:   mtls,
			TotalRequests: g.TotalReqs,
		})
	}
	return nodes, nil
}

// getServiceNodeDetail 获取单个服务节点详情
func (q *QueryService) getServiceNodeDetail(ctx context.Context, clusterID, namespace, name string, start, end time.Time) (*model.ServiceNodeResponse, error) {
	// 尝试 hourly
	hourlies, err := q.serviceRepo.GetServiceHourly(ctx, clusterID, namespace, name, start, end)
	if err != nil {
		return nil, err
	}

	if len(hourlies) > 0 {
		var totalReqs, errorReqs int64
		var p95Sum, p99Sum, avgSum float64
		var rpsSum, mtlsSum float64

		for _, h := range hourlies {
			totalReqs += h.TotalRequests
			errorReqs += h.ErrorRequests
			p95Sum += float64(h.P95LatencyMs) * float64(h.TotalRequests)
			p99Sum += float64(h.P99LatencyMs) * float64(h.TotalRequests)
			avgSum += float64(h.AvgLatencyMs) * float64(h.TotalRequests)
			rpsSum += h.AvgRPS
			mtlsSum += h.MtlsPercent
		}

		var p95, p99, avg int
		if totalReqs > 0 {
			p95 = int(p95Sum / float64(totalReqs))
			p99 = int(p99Sum / float64(totalReqs))
			avg = int(avgSum / float64(totalReqs))
		}

		avail := slo.CalculateAvailability(totalReqs, errorReqs)
		errRate := slo.CalculateErrorRate(totalReqs, errorReqs)
		rps := rpsSum / float64(len(hourlies))
		mtls := mtlsSum / float64(len(hourlies))

		status := "healthy"
		if errRate > 5 {
			status = "critical"
		} else if errRate > 1 || p95 > 500 {
			status = "warning"
		}

		return &model.ServiceNodeResponse{
			ID:            namespace + "/" + name,
			Name:          name,
			Namespace:     namespace,
			RPS:           rps,
			AvgLatencyMs:  avg,
			P95LatencyMs:  p95,
			P99LatencyMs:  p99,
			ErrorRate:     errRate,
			Availability:  avail,
			Status:        status,
			MtlsPercent:   mtls,
			TotalRequests: totalReqs,
		}, nil
	}

	// 回退 raw
	raws, err := q.serviceRepo.GetServiceRaw(ctx, clusterID, namespace, name, start, end)
	if err != nil || len(raws) == 0 {
		return nil, err
	}

	var totalReqs, errorReqs int64
	var latencySum float64
	var latencyCount int64
	var tls, total int64
	var allBuckets []map[float64]int64

	for _, r := range raws {
		totalReqs += r.TotalRequests
		errorReqs += r.ErrorRequests
		latencySum += r.LatencySum
		latencyCount += r.LatencyCount
		tls += r.TLSRequestDelta
		total += r.TotalRequestDelta
		if b := slo.ParseJSONBuckets(r.LatencyBuckets); b != nil {
			allBuckets = append(allBuckets, b)
		}
	}

	merged := slo.MergeBuckets(allBuckets...)
	p95 := slo.CalculateQuantileMs(merged, 0.95)
	p99 := slo.CalculateQuantileMs(merged, 0.99)
	var avg int
	if latencyCount > 0 {
		avg = int(latencySum / float64(latencyCount))
	}

	avail := slo.CalculateAvailability(totalReqs, errorReqs)
	errRate := slo.CalculateErrorRate(totalReqs, errorReqs)
	durationSec := float64(len(raws)) * 10.0
	rps := slo.CalculateRPS(totalReqs, durationSec)

	var mtls float64
	if total > 0 {
		mtls = float64(tls) / float64(total) * 100
	}

	status := "healthy"
	if errRate > 5 {
		status = "critical"
	} else if errRate > 1 || p95 > 500 {
		status = "warning"
	}

	return &model.ServiceNodeResponse{
		ID:            namespace + "/" + name,
		Name:          name,
		Namespace:     namespace,
		RPS:           rps,
		AvgLatencyMs:  avg,
		P95LatencyMs:  p95,
		P99LatencyMs:  p99,
		ErrorRate:     errRate,
		Availability:  avail,
		Status:        status,
		MtlsPercent:   mtls,
		TotalRequests: totalReqs,
	}, nil
}

// getServiceEdges 获取拓扑边（hourly 优先，raw 回退）
func (q *QueryService) getServiceEdges(ctx context.Context, clusterID string, start, end time.Time) ([]model.ServiceEdgeResponse, error) {
	// 尝试 hourly
	hourlies, err := q.edgeRepo.GetEdgeHourly(ctx, clusterID, start, end)
	if err != nil {
		return nil, err
	}

	if len(hourlies) > 0 {
		type key struct{ Src, Dst string }
		groups := map[key]struct {
			TotalReqs, ErrorReqs int64
			LatencyMs            float64
			RPSSum               float64
			Count                int
		}{}

		for _, h := range hourlies {
			k := key{
				Src: h.SrcNamespace + "/" + h.SrcName,
				Dst: h.DstNamespace + "/" + h.DstName,
			}
			g := groups[k]
			g.TotalReqs += h.TotalRequests
			g.ErrorReqs += h.ErrorRequests
			g.LatencyMs += float64(h.AvgLatencyMs) * float64(h.TotalRequests)
			g.RPSSum += h.AvgRPS
			g.Count++
			groups[k] = g
		}

		var edges []model.ServiceEdgeResponse
		for k, g := range groups {
			var avg int
			if g.TotalReqs > 0 {
				avg = int(g.LatencyMs / float64(g.TotalReqs))
			}
			edges = append(edges, model.ServiceEdgeResponse{
				Source:       k.Src,
				Target:       k.Dst,
				RPS:          g.RPSSum / float64(g.Count),
				AvgLatencyMs: avg,
				ErrorRate:    slo.CalculateErrorRate(g.TotalReqs, g.ErrorReqs),
			})
		}
		return edges, nil
	}

	// 回退 raw
	raws, err := q.edgeRepo.GetEdgeRaw(ctx, clusterID, start, end)
	if err != nil {
		return nil, err
	}

	type key struct{ Src, Dst string }
	groups := map[key]struct {
		TotalReqs, ErrorReqs int64
		LatencySum           float64
		LatencyCount         int64
		SampleCount          int
	}{}

	for _, r := range raws {
		k := key{
			Src: r.SrcNamespace + "/" + r.SrcName,
			Dst: r.DstNamespace + "/" + r.DstName,
		}
		g := groups[k]
		g.TotalReqs += r.RequestDelta
		g.ErrorReqs += r.FailureDelta
		g.LatencySum += r.LatencySum
		g.LatencyCount += r.LatencyCount
		g.SampleCount++
		groups[k] = g
	}

	var edges []model.ServiceEdgeResponse
	for k, g := range groups {
		var avg int
		if g.LatencyCount > 0 {
			avg = int(g.LatencySum / float64(g.LatencyCount))
		}
		durationSec := float64(g.SampleCount) * 10.0
		edges = append(edges, model.ServiceEdgeResponse{
			Source:       k.Src,
			Target:       k.Dst,
			RPS:          slo.CalculateRPS(g.TotalReqs, durationSec),
			AvgLatencyMs: avg,
			ErrorRate:    slo.CalculateErrorRate(g.TotalReqs, g.ErrorReqs),
		})
	}
	return edges, nil
}

// getTimeStart 根据时间范围计算起始时间
func getTimeStart(now time.Time, timeRange string) time.Time {
	switch timeRange {
	case "1h":
		return now.Add(-time.Hour)
	case "6h":
		return now.Add(-6 * time.Hour)
	case "24h", "1d":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.Add(-7 * 24 * time.Hour)
	case "30d":
		return now.Add(-30 * 24 * time.Hour)
	default:
		return now.Add(-24 * time.Hour)
	}
}

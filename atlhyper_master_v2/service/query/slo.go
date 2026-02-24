// atlhyper_master_v2/service/query/slo.go
// SLO 查询实现 — OTelSnapshot 直读模式
//
// 服务网格拓扑和服务详情从 OTelSnapshot 直读，不再依赖 SQLite 时序表。
// Handler（Gateway 层）通过 service.Query 接口调用。
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	slomodel "AtlHyper/model_v3/slo"
)

// GetMeshTopology 获取服务网格拓扑（OTelSnapshot 直读）
func (q *QueryService) GetMeshTopology(ctx context.Context, clusterID, timeRange string) (*model.ServiceMeshTopologyResponse, error) {
	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil || otel == nil {
		return &model.ServiceMeshTopologyResponse{}, nil
	}

	// 转换 SLOServices → 节点
	nodes := make([]model.ServiceNodeResponse, 0, len(otel.SLOServices))
	for _, svc := range otel.SLOServices {
		errRate := (1 - svc.SuccessRate) * 100
		avail := svc.SuccessRate * 100
		status := determineMeshStatus(errRate, svc.P99Ms)

		nodes = append(nodes, model.ServiceNodeResponse{
			ID:            svc.Namespace + "/" + svc.Name,
			Name:          svc.Name,
			Namespace:     svc.Namespace,
			RPS:           svc.RPS,
			AvgLatencyMs:  int(svc.P50Ms),
			P50LatencyMs:  int(svc.P50Ms),
			P95LatencyMs:  int(svc.P90Ms),
			P99LatencyMs:  int(svc.P99Ms),
			ErrorRate:     errRate,
			Availability:  avail,
			Status:        status,
			MtlsPercent:   svc.MTLSRate * 100,
			TotalRequests: totalFromStatusCodes(svc.StatusCodes),
		})
	}

	// 转换 SLOEdges → 边
	edges := convertEdges(otel.SLOEdges)

	return &model.ServiceMeshTopologyResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// GetServiceDetail 获取单个服务详情（OTelSnapshot 直读）
func (q *QueryService) GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error) {
	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil || otel == nil {
		return nil, nil
	}

	serviceID := namespace + "/" + name

	// 1. 查找服务节点
	var node *model.ServiceNodeResponse
	for _, svc := range otel.SLOServices {
		if svc.Namespace == namespace && svc.Name == name {
			errRate := (1 - svc.SuccessRate) * 100
			avail := svc.SuccessRate * 100
			totalReqs := totalFromStatusCodes(svc.StatusCodes)

			node = &model.ServiceNodeResponse{
				ID:            serviceID,
				Name:          svc.Name,
				Namespace:     svc.Namespace,
				RPS:           svc.RPS,
				AvgLatencyMs:  int(svc.P50Ms),
				P50LatencyMs:  int(svc.P50Ms),
				P95LatencyMs:  int(svc.P90Ms),
				P99LatencyMs:  int(svc.P99Ms),
				ErrorRate:     errRate,
				Availability:  avail,
				Status:        determineMeshStatus(errRate, svc.P99Ms),
				MtlsPercent:   svc.MTLSRate * 100,
				TotalRequests: totalReqs,
			}

			break
		}
	}

	if node == nil {
		return nil, nil
	}

	resp := &model.ServiceDetailResponse{
		ServiceNodeResponse: *node,
	}

	// 2. 历史数据（Concentrator 时序，1min 粒度 × 60 点）
	for _, ts := range otel.SLOTimeSeries {
		if ts.ServiceName == name || ts.ServiceName == serviceID {
			for _, p := range ts.Points {
				resp.History = append(resp.History, model.ServiceHistoryPoint{
					Timestamp:    p.Timestamp.Format(time.RFC3339),
					RPS:          p.RPS,
					P95LatencyMs: int(p.P99Ms),
					ErrorRate:    p.ErrorRate * 100,
					Availability: p.SuccessRate * 100,
				})
			}
			break
		}
	}

	// 3. 状态码（从 ServiceSLO.StatusCodes）
	for _, svc := range otel.SLOServices {
		if svc.Namespace == namespace && svc.Name == name {
			for _, sc := range svc.StatusCodes {
				if sc.Count > 0 {
					resp.StatusCodes = append(resp.StatusCodes, model.StatusCodeBreakdown{
						Code:  sc.Code,
						Count: sc.Count,
					})
				}
			}
			break
		}
	}

	// 4. 上下游边
	for _, e := range otel.SLOEdges {
		edge := model.ServiceEdgeResponse{
			Source:       e.SrcNamespace + "/" + e.SrcName,
			Target:       e.DstNamespace + "/" + e.DstName,
			RPS:          e.RPS,
			AvgLatencyMs: int(e.AvgMs),
			ErrorRate:    (1 - e.SuccessRate) * 100,
		}
		if edge.Target == serviceID {
			resp.Upstreams = append(resp.Upstreams, edge)
		}
		if edge.Source == serviceID {
			resp.Downstreams = append(resp.Downstreams, edge)
		}
	}

	return resp, nil
}

// ==================== 辅助函数 ====================

// determineMeshStatus 根据错误率和延迟判断服务状态
func determineMeshStatus(errRatePct, p99Ms float64) string {
	if errRatePct > 5 {
		return "critical"
	}
	if errRatePct > 1 || p99Ms > 500 {
		return "warning"
	}
	return "healthy"
}

// convertEdges 转换 SLOEdges → ServiceEdgeResponse
func convertEdges(edges []slomodel.ServiceEdge) []model.ServiceEdgeResponse {
	result := make([]model.ServiceEdgeResponse, 0, len(edges))
	for _, e := range edges {
		result = append(result, model.ServiceEdgeResponse{
			Source:       e.SrcNamespace + "/" + e.SrcName,
			Target:       e.DstNamespace + "/" + e.DstName,
			RPS:          e.RPS,
			AvgLatencyMs: int(e.AvgMs),
			ErrorRate:    (1 - e.SuccessRate) * 100,
		})
	}
	return result
}

// totalFromStatusCodes 从状态码计算总请求数
func totalFromStatusCodes(codes []slomodel.StatusCodeCount) int64 {
	var total int64
	for _, sc := range codes {
		total += sc.Count
	}
	return total
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

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

	// 优先从 SLOWindows[timeRange] 获取 Mesh 数据
	var services []slomodel.ServiceSLO
	var edges []slomodel.ServiceEdge
	if otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok {
			services = w.MeshServices
			edges = w.MeshEdges
		}
	}
	// 回退: 5min 快照
	if len(services) == 0 {
		services = otel.SLOServices
		edges = otel.SLOEdges
	}

	// 转换 ServiceSLO → 节点
	// 注意: Agent 返回的 SuccessRate/MTLSRate 已经是 0-100 百分比
	nodes := make([]model.ServiceNodeResponse, 0, len(services))
	for _, svc := range services {
		nodes = append(nodes, serviceToNode(svc))
	}

	return &model.ServiceMeshTopologyResponse{
		Nodes: nodes,
		Edges: convertEdges(edges),
	}, nil
}

// GetServiceDetail 获取单个服务详情（OTelSnapshot 直读）
func (q *QueryService) GetServiceDetail(ctx context.Context, clusterID, namespace, name, timeRange string) (*model.ServiceDetailResponse, error) {
	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil || otel == nil {
		return nil, nil
	}

	serviceID := namespace + "/" + name

	// 优先从 SLOWindows[timeRange] 获取 Mesh 数据
	var services []slomodel.ServiceSLO
	var allEdges []slomodel.ServiceEdge
	if otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok {
			services = w.MeshServices
			allEdges = w.MeshEdges
		}
	}
	// 回退: 5min 快照
	if len(services) == 0 {
		services = otel.SLOServices
		allEdges = otel.SLOEdges
	}

	// 1. 查找服务节点
	var foundSvc *slomodel.ServiceSLO
	for i, svc := range services {
		if svc.Namespace == namespace && svc.Name == name {
			foundSvc = &services[i]
			break
		}
	}
	if foundSvc == nil {
		return nil, nil
	}

	node := serviceToNode(*foundSvc)
	resp := &model.ServiceDetailResponse{
		ServiceNodeResponse: node,
	}

	// 2. 历史数据（Concentrator 时序，1min 粒度 × 60 点）
	// Concentrator key 为 "namespace/name"（mesh 服务）或 ServiceKey（ingress）
	for _, ts := range otel.SLOTimeSeries {
		if ts.ServiceName == name || ts.ServiceName == serviceID {
			for _, p := range ts.Points {
				resp.History = append(resp.History, model.ServiceHistoryPoint{
					Timestamp:    p.Timestamp.Format(time.RFC3339),
					RPS:          p.RPS,
					P95LatencyMs: p.P99Ms,
					ErrorRate:    p.ErrorRate,     // 已经是 0-100
					Availability: p.SuccessRate,   // 已经是 0-100
				})
			}
			break
		}
	}

	// 3. 状态码（从 ServiceSLO.StatusCodes）
	for _, sc := range foundSvc.StatusCodes {
		if sc.Count > 0 {
			resp.StatusCodes = append(resp.StatusCodes, model.StatusCodeBreakdown{
				Code:  sc.Code,
				Count: sc.Count,
			})
		}
	}

	// 4. 延迟分布桶（Linkerd histogram）
	for _, lb := range foundSvc.LatencyBuckets {
		resp.LatencyBuckets = append(resp.LatencyBuckets, model.LatencyBucket{
			LE:    lb.LE,
			Count: lb.Count,
		})
	}

	// 5. 上下游边
	for _, e := range allEdges {
		edge := convertEdge(e)
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

// serviceToNode 将 ServiceSLO 转换为 ServiceNodeResponse
// Agent 返回的 SuccessRate/MTLSRate 已经是 0-100 百分比
func serviceToNode(svc slomodel.ServiceSLO) model.ServiceNodeResponse {
	errRate := 100 - svc.SuccessRate
	return model.ServiceNodeResponse{
		ID:            svc.Namespace + "/" + svc.Name,
		Name:          svc.Name,
		Namespace:     svc.Namespace,
		RPS:           svc.RPS,
		AvgLatencyMs:  svc.P50Ms,
		P50LatencyMs:  svc.P50Ms,
		P95LatencyMs:  svc.P90Ms,
		P99LatencyMs:  svc.P99Ms,
		ErrorRate:     errRate,
		Availability:  svc.SuccessRate,
		Status:        determineMeshStatus(errRate, svc.P99Ms),
		MtlsEnabled:   svc.MTLSEnabled,
		TotalRequests: totalFromStatusCodes(svc.StatusCodes),
	}
}

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

// convertEdge 转换单个 ServiceEdge → ServiceEdgeResponse
func convertEdge(e slomodel.ServiceEdge) model.ServiceEdgeResponse {
	return model.ServiceEdgeResponse{
		Source:       e.SrcNamespace + "/" + e.SrcName,
		Target:       e.DstNamespace + "/" + e.DstName,
		RPS:          e.RPS,
		AvgLatencyMs: e.AvgMs,
		ErrorRate:    100 - e.SuccessRate, // SuccessRate 已经是 0-100
	}
}

// convertEdges 转换 SLOEdges → ServiceEdgeResponse
func convertEdges(edges []slomodel.ServiceEdge) []model.ServiceEdgeResponse {
	result := make([]model.ServiceEdgeResponse, 0, len(edges))
	for _, e := range edges {
		result = append(result, convertEdge(e))
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

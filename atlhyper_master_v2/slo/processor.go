// Package slo SLO 数据处理器
//
// processor.go - SLO 数据处理器
//
// 接收 Agent 上报的 SLOSnapshot（已计算增量），直接写入 raw 表。
// Agent 已完成 per-pod delta 计算和 service 聚合，Master 无需 delta。
package slo

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var procLog = logger.Module("SLO")

// Processor SLO 数据处理器
type Processor struct {
	sloRepo     database.SLORepository
	serviceRepo database.SLOServiceRepository
	edgeRepo    database.SLOEdgeRepository
}

// NewProcessor 创建 SLO Processor
func NewProcessor(sloRepo database.SLORepository, serviceRepo database.SLOServiceRepository, edgeRepo database.SLOEdgeRepository) *Processor {
	return &Processor{
		sloRepo:     sloRepo,
		serviceRepo: serviceRepo,
		edgeRepo:    edgeRepo,
	}
}

// ProcessSLOSnapshot 处理完整 SLO 快照
// Agent 发送的是增量值，Master 直接存储，无需 delta 计算
func (p *Processor) ProcessSLOSnapshot(ctx context.Context, clusterID string, snapshot *model_v2.SLOSnapshot) error {
	if snapshot == nil {
		return nil
	}

	ts := time.Unix(snapshot.Timestamp, 0)

	// 1. 服务网格指标 → slo_service_raw
	for i := range snapshot.Services {
		if err := p.processServiceMetrics(ctx, clusterID, ts, &snapshot.Services[i]); err != nil {
			procLog.Error("处理 ServiceMetrics 失败", "cluster", clusterID, "service", snapshot.Services[i].Name, "err", err)
		}
	}

	// 2. 拓扑边 → slo_edge_raw
	for i := range snapshot.Edges {
		if err := p.processEdge(ctx, clusterID, ts, &snapshot.Edges[i]); err != nil {
			procLog.Error("处理 Edge 失败", "cluster", clusterID, "err", err)
		}
	}

	// 3. 入口指标 → slo_metrics_raw
	for i := range snapshot.Ingress {
		if err := p.processIngressMetrics(ctx, clusterID, ts, &snapshot.Ingress[i]); err != nil {
			procLog.Error("处理 IngressMetrics 失败", "cluster", clusterID, "key", snapshot.Ingress[i].ServiceKey, "err", err)
		}
	}

	// 4. 路由映射 → slo_route_mapping
	p.processRoutes(ctx, clusterID, snapshot.Routes)

	return nil
}

// ProcessIngressRoutes 处理 Agent 上报的 IngressRoute 映射信息
func (p *Processor) ProcessIngressRoutes(ctx context.Context, clusterID string, routes []model_v2.IngressRouteInfo) error {
	p.processRoutes(ctx, clusterID, routes)
	return nil
}

// processServiceMetrics 存储服务网格指标到 slo_service_raw
func (p *Processor) processServiceMetrics(ctx context.Context, clusterID string, ts time.Time, svc *model_v2.ServiceMetrics) error {
	// 从 Requests[] 聚合状态码分组
	var totalReqs, errorReqs int64
	var s2xx, s3xx, s4xx, s5xx int64
	for _, r := range svc.Requests {
		totalReqs += r.Delta
		if r.Classification == "failure" {
			errorReqs += r.Delta
		}
		switch {
		case strings.HasPrefix(r.StatusCode, "2"):
			s2xx += r.Delta
		case strings.HasPrefix(r.StatusCode, "3"):
			s3xx += r.Delta
		case strings.HasPrefix(r.StatusCode, "4"):
			s4xx += r.Delta
		case strings.HasPrefix(r.StatusCode, "5"):
			s5xx += r.Delta
		}
	}

	raw := &database.SLOServiceRaw{
		ClusterID:         clusterID,
		Namespace:         svc.Namespace,
		Name:              svc.Name,
		Timestamp:         ts,
		TotalRequests:     totalReqs,
		ErrorRequests:     errorReqs,
		Status2xx:         s2xx,
		Status3xx:         s3xx,
		Status4xx:         s4xx,
		Status5xx:         s5xx,
		LatencySum:        svc.LatencySum,
		LatencyCount:      svc.LatencyCount,
		LatencyBuckets:    marshalBuckets(svc.LatencyBuckets),
		TLSRequestDelta:   svc.TLSRequestDelta,
		TotalRequestDelta: svc.TotalRequestDelta,
	}
	return p.serviceRepo.InsertServiceRaw(ctx, raw)
}

// processEdge 存储拓扑边到 slo_edge_raw
func (p *Processor) processEdge(ctx context.Context, clusterID string, ts time.Time, edge *model_v2.ServiceEdge) error {
	raw := &database.SLOEdgeRaw{
		ClusterID:    clusterID,
		SrcNamespace: edge.SrcNamespace,
		SrcName:      edge.SrcName,
		DstNamespace: edge.DstNamespace,
		DstName:      edge.DstName,
		Timestamp:    ts,
		RequestDelta: edge.RequestDelta,
		FailureDelta: edge.FailureDelta,
		LatencySum:   edge.LatencySum,
		LatencyCount: edge.LatencyCount,
	}
	return p.edgeRepo.InsertEdgeRaw(ctx, raw)
}

// processIngressMetrics 存储入口指标到 slo_metrics_raw
func (p *Processor) processIngressMetrics(ctx context.Context, clusterID string, ts time.Time, ing *model_v2.IngressMetrics) error {
	// 从 Requests[] 聚合
	var totalReqs, errorReqs int64
	methodCounts := map[string]int64{}
	for _, r := range ing.Requests {
		totalReqs += r.Delta
		code, _ := strconv.Atoi(r.Code)
		if code >= 500 {
			errorReqs += r.Delta
		}
		methodCounts[strings.ToUpper(r.Method)] += r.Delta
	}

	raw := &database.SLOMetricsRaw{
		ClusterID:      clusterID,
		Host:           ing.ServiceKey,
		Timestamp:      ts,
		TotalRequests:  totalReqs,
		ErrorRequests:  errorReqs,
		LatencySum:     ing.LatencySum,
		LatencyCount:   ing.LatencyCount,
		LatencyBuckets: marshalBuckets(ing.LatencyBuckets),
		MethodGet:      methodCounts["GET"],
		MethodPost:     methodCounts["POST"],
		MethodPut:      methodCounts["PUT"],
		MethodDelete:   methodCounts["DELETE"],
		MethodOther:    methodCounts["OTHER"] + methodCounts["PATCH"] + methodCounts["HEAD"],
	}

	// 关联域名（从 route_mapping 查）
	mapping, _ := p.sloRepo.GetRouteMappingByServiceKey(ctx, clusterID, ing.ServiceKey)
	if mapping != nil {
		raw.Domain = mapping.Domain
		raw.PathPrefix = mapping.PathPrefix
	}

	return p.sloRepo.InsertRawMetrics(ctx, raw)
}

// processRoutes 更新路由映射
func (p *Processor) processRoutes(ctx context.Context, clusterID string, routes []model_v2.IngressRouteInfo) {
	now := time.Now()
	for _, route := range routes {
		if err := p.sloRepo.UpsertRouteMapping(ctx, &database.SLORouteMapping{
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
		}); err != nil {
			procLog.Error("更新路由映射失败", "cluster", clusterID, "domain", route.Domain, "err", err)
		}
	}
}

// marshalBuckets 将 bucket map 序列化为 JSON 字符串
func marshalBuckets(buckets map[string]int64) string {
	if len(buckets) == 0 {
		return ""
	}
	data, err := json.Marshal(buckets)
	if err != nil {
		return ""
	}
	return string(data)
}

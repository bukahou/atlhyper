// Package slo SLO 数据仓库
//
// slo.go - SLORepository 主入口
//
// 编排 SLO 数据采集的完整管线:
//
//	OTelClient.ScrapeMetrics()
//	    ↓
//	Stage 1: filter      排除 probe/admin/系统ns     ← filter.go
//	    ↓
//	Stage 2: per-pod delta   snapshotManager 算增量    ← snapshot.go
//	    ↓
//	Stage 3a: aggregateServices  inbound → ServiceMetrics[]  ┐
//	Stage 3b: extractEdges       outbound → ServiceEdge[]    ├─ aggregate.go
//	Stage 3c: aggregateIngress   ingress → IngressMetrics[]  ┘
//	    ↓
//	Stage 4: collectRoutes  IngressRoute CRD → RouteInfo[]
//	    ↓
//	SLOSnapshot
package slo

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var sloLog = logger.Module("SLORepository")

// sloRepository SLO 数据仓库实现
type sloRepository struct {
	otelClient        sdk.OTelClient
	ingressClient     sdk.IngressClient
	snapshot          *snapshotManager
	excludeNamespaces []string
}

// NewSLORepository 创建 SLO 数据仓库
//
// 参数:
//   - otelClient: OTel Collector 采集客户端
//   - ingressClient: Ingress 路由采集客户端（用于 CollectRoutes）
//   - excludeNamespaces: 排除的 namespace 列表
func NewSLORepository(
	otelClient sdk.OTelClient,
	ingressClient sdk.IngressClient,
	excludeNamespaces []string,
) repository.SLORepository {
	return &sloRepository{
		otelClient:        otelClient,
		ingressClient:     ingressClient,
		snapshot:          newSnapshotManager(),
		excludeNamespaces: excludeNamespaces,
	}
}

// Collect 采集并处理 SLO 数据
//
// 完成: scrape → filter → per-pod delta → aggregate → routes → SLOSnapshot
func (r *sloRepository) Collect(ctx context.Context) (*model_v2.SLOSnapshot, error) {
	if r.otelClient == nil {
		return nil, nil
	}

	// Stage 0: 从 OTel Collector 采集原始指标
	raw, err := r.otelClient.ScrapeMetrics(ctx)
	if err != nil {
		sloLog.Warn("采集 OTel 指标失败", "err", err)
		return nil, err
	}

	// Stage 1: 过滤
	filtered := r.filter(raw)

	// Stage 2: Per-pod delta
	deltas := r.snapshot.calcDeltas(filtered)

	// Stage 3a: 聚合 inbound → ServiceMetrics
	services := aggregateServices(
		deltas.inboundResponses,
		deltas.inboundBuckets,
		deltas.inboundSums,
		deltas.inboundCounts,
	)

	// Stage 3b: 提取 outbound → ServiceEdge
	edges := extractEdges(
		deltas.outboundResponses,
		deltas.outboundSums,
		deltas.outboundCounts,
	)

	// Stage 3c: 聚合 ingress → IngressMetrics (秒→毫秒)
	ingress := aggregateIngress(
		deltas.ingressRequests,
		deltas.ingressBuckets,
		deltas.ingressSums,
		deltas.ingressCounts,
	)

	// Stage 4: 采集路由映射
	var routes []model_v2.IngressRouteInfo
	if r.ingressClient != nil {
		sdkRoutes, routeErr := r.ingressClient.CollectRoutes(ctx)
		if routeErr != nil {
			sloLog.Debug("采集路由映射失败", "err", routeErr)
		} else {
			routes = make([]model_v2.IngressRouteInfo, 0, len(sdkRoutes))
			for _, rt := range sdkRoutes {
				routes = append(routes, model_v2.IngressRouteInfo{
					Name:        rt.Name,
					Namespace:   rt.Namespace,
					Domain:      rt.Domain,
					PathPrefix:  rt.PathPrefix,
					ServiceKey:  rt.ServiceKey,
					ServiceName: rt.ServiceName,
					ServicePort: rt.ServicePort,
					TLS:         rt.TLS,
				})
			}
		}
	}

	// 组装 SLOSnapshot
	result := &model_v2.SLOSnapshot{
		Timestamp: time.Now().Unix(),
		Services:  services,
		Edges:     edges,
		Ingress:   ingress,
		Routes:    routes,
	}

	sloLog.Debug("SLO 采集完成",
		"services", len(services),
		"edges", len(edges),
		"ingress", len(ingress),
		"routes", len(routes),
	)

	return result, nil
}


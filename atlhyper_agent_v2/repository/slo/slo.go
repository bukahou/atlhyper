// Package slo SLO 数据仓库
//
// slo.go - SLORepository 主入口
//
// 当前为过渡版本（临时桩），Agent P3 阶段将完全重写为:
// OTelClient.ScrapeMetrics → filter → per-pod delta → aggregate → SLOSnapshot
package slo

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
	"AtlHyper/model_v2"
)

var sloLog = logger.Module("SLORepository")

// sloRepository SLO 数据仓库实现（过渡版本）
type sloRepository struct {
	otelClient    sdk.OTelClient
	ingressClient sdk.IngressClient
}

// NewSLORepository 创建 SLO 数据仓库
//
// P3 阶段将扩展为完整实现，包含 filter/snapshot/aggregate 管线。
func NewSLORepository(otelClient sdk.OTelClient, ingressClient sdk.IngressClient) repository.SLORepository {
	return &sloRepository{
		otelClient:    otelClient,
		ingressClient: ingressClient,
	}
}

// Collect 采集 SLO 指标数据（过渡版本: 仅采集路由，不采集指标）
func (r *sloRepository) Collect(ctx context.Context) (*model_v2.SLOSnapshot, error) {
	// TODO(P3): 完整实现 OTel 采集管线
	// scrape → filter → per-pod delta → aggregate → SLOSnapshot
	sloLog.Debug("SLO Collect: 过渡版本，等待 P3 完整实现")
	return nil, nil
}

// CollectRoutes 采集 IngressRoute 配置
func (r *sloRepository) CollectRoutes(ctx context.Context) ([]model_v2.IngressRouteInfo, error) {
	if r.ingressClient == nil {
		return nil, nil
	}

	sdkRoutes, err := r.ingressClient.CollectRoutes(ctx)
	if err != nil {
		return nil, err
	}

	routes := make([]model_v2.IngressRouteInfo, 0, len(sdkRoutes))
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
	return routes, nil
}

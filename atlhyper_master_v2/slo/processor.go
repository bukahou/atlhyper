// Package slo SLO 数据处理器
//
// processor.go - SLO 数据处理器（过渡版本）
//
// 接收 Agent 上报的 SLO 指标，写入数据库。
// Master P2 阶段将完全重写为支持 ServiceMetrics + ServiceEdge + IngressMetrics 三层数据。
package slo

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v2"
)

// Processor SLO 数据处理器
type Processor struct {
	repo database.SLORepository
}

// NewProcessor 创建 SLO Processor
func NewProcessor(repo database.SLORepository) *Processor {
	return &Processor{
		repo: repo,
	}
}

// ProcessSLOSnapshot 处理 Agent 上报的 SLO 快照数据（过渡版本）
//
// TODO(Master P2): 实现完整的处理管线:
//   - ServiceMetrics → slo_service_raw 表
//   - ServiceEdge → slo_edge_raw 表
//   - IngressMetrics → slo_ingress_raw 表
func (p *Processor) ProcessSLOSnapshot(ctx context.Context, clusterID string, snapshot *model_v2.SLOSnapshot) error {
	if snapshot == nil {
		return nil
	}
	// TODO(Master P2): 写入数据库
	return nil
}

// ProcessIngressRoutes 处理 Agent 上报的 IngressRoute 映射信息
func (p *Processor) ProcessIngressRoutes(ctx context.Context, clusterID string, routes []model_v2.IngressRouteInfo) error {
	if len(routes) == 0 {
		return nil
	}

	// TODO(Master P2): 更新路由映射
	return nil
}


// Package slo SLO 领域处理器
//
// interfaces.go - 对外接口定义
//
// slo 包是领域处理器（domain processor），不是独立的架构层。
// 组件按调用方式分为两类:
//   - processor: 被 Service 层调用（service/sync/slo_persist.go）
//   - aggregator/cleaner/status_checker: 独立后台任务（master.go 启动）
//   - calculator: 纯函数，任何层可调用
package slo

import (
	"context"

	"AtlHyper/model_v2"
)

// SLOProcessor SLO 数据处理器接口
//
// 接收 Agent 上报的 SLO 快照数据，直接写入 raw 表。
// Agent 已完成 per-pod delta 计算和 service 聚合，Master 无需 delta。
type SLOProcessor interface {
	ProcessSLOSnapshot(ctx context.Context, clusterID string, snapshot *model_v2.SLOSnapshot) error
}

// SLOAggregator SLO 数据预聚合器接口
//
// 定时将 raw 数据聚合为 hourly，用于历史查询和性能优化。
// 每次聚合上一个完整小时 + 当前不完整小时。
type SLOAggregator interface {
	Start()
	Stop()
}

// SLOCleaner SLO 数据清理器接口
//
// 定时清理过期数据:
//   - raw: 48h
//   - hourly: 90d
//   - status_history: 180d
type SLOCleaner interface {
	Start()
	Stop()
}

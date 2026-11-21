// Package service 业务逻辑层
//
// Service 层封装业务逻辑，协调多个 Repository 调用。
// 上层 (Scheduler) 只依赖 Service 接口，不直接操作 Repository。
//
// 主要服务:
//   - SnapshotService: 集群快照采集
//   - CommandService: 指令执行
package service

import (
	"context"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// SnapshotService 快照采集服务接口
type SnapshotService interface {
	// Collect 采集 K8s 资源快照；OTel 部分附上最近一次刷新好的结果，不在这里等 ClickHouse
	Collect(ctx context.Context) (*cluster.ClusterSnapshot, error)
	// RefreshOTel 刷新 OTel 概览（ClickHouse 查询）；由独立循环调用，ctx 取消即返回
	RefreshOTel(ctx context.Context)
}

// CommandService 指令执行服务接口
type CommandService interface {
	Execute(ctx context.Context, cmd *command.Command) *command.Result
}

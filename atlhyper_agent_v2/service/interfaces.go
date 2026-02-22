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
	Collect(ctx context.Context) (*cluster.ClusterSnapshot, error)
}

// CommandService 指令执行服务接口
type CommandService interface {
	Execute(ctx context.Context, cmd *command.Command) *command.Result
}

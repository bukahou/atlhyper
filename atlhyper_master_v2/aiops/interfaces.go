// atlhyper_master_v2/aiops/interfaces.go
// AIOps 引擎对外接口
package aiops

import "context"

// Engine AIOps 引擎接口
type Engine interface {
	// OnSnapshot 快照更新时触发（图更新 + 基线检测）
	OnSnapshot(clusterID string)

	// GetGraph 获取指定集群的依赖图
	GetGraph(clusterID string) *DependencyGraph

	// GetGraphTrace 追踪指定实体的上下游链路
	GetGraphTrace(clusterID, fromKey, direction string, maxDepth int) *TraceResult

	// GetBaseline 获取指定实体的基线状态
	GetBaseline(entityKey string) *EntityBaseline

	// Start 启动引擎（加载 DB 状态 + 定时 flush）
	Start(ctx context.Context) error

	// Stop 停止引擎（最终 flush + 清理）
	Stop() error
}

// atlhyper_master_v2/aiops/interfaces.go
// AIOps 引擎对外接口
package aiops

import (
	"context"
	"time"
)

// Engine AIOps 引擎接口
type Engine interface {
	// OnSnapshot 快照更新时触发（图更新 + 基线检测 + 风险评分 + 状态机评估）
	OnSnapshot(clusterID string)

	// GetGraph 获取指定集群的依赖图
	GetGraph(clusterID string) *DependencyGraph

	// GetGraphTrace 追踪指定实体的上下游链路
	GetGraphTrace(clusterID, fromKey, direction string, maxDepth int) *TraceResult

	// GetBaseline 获取指定实体的基线状态
	GetBaseline(clusterID, entityKey string) *EntityBaseline

	// GetClusterRisk 获取集群风险评分
	GetClusterRisk(clusterID string) *ClusterRisk

	// GetEntityRisks 获取实体风险列表（支持排序和分页）
	GetEntityRisks(clusterID, sortBy string, limit int) []*EntityRisk

	// GetEntityRisk 获取单个实体的风险详情
	GetEntityRisk(clusterID, entityKey string) *EntityRiskDetail

	// GetIncidents 查询事件列表
	GetIncidents(ctx context.Context, opts IncidentQueryOpts) ([]*Incident, int, error)

	// GetIncidentDetail 获取事件详情
	GetIncidentDetail(ctx context.Context, incidentID string) *IncidentDetail

	// GetIncidentStats 获取事件统计
	GetIncidentStats(ctx context.Context, clusterID string, since time.Time) *IncidentStats

	// GetIncidentPatterns 获取历史事件模式
	GetIncidentPatterns(ctx context.Context, clusterID, entityKey string, since time.Time) []*IncidentPattern

	// SubscribeTransitions 订阅事件变化（须在 Start 之前调用）。
	// 订阅者被同步调用：多数在快照处理主路径上，关闭类变化在引擎的定期检查协程里；
	// 各次调用先后有序、不会并发。只允许做「放进队列」这类不阻塞的事
	SubscribeTransitions(fn func(IncidentTransition))

	// SubscribeLinks 订阅拓扑归并的关联变化（须在 Start 之前调用）：受影响事件归到源头、换源头、解除（RootIncidentID 为空）。
	// Start 时会把库里恢复的有效关联逐条发一次，订阅者据此重建状态。约束同 SubscribeTransitions
	SubscribeLinks(fn func(IncidentLink))

	// Start 启动引擎（加载 DB 状态 + 定时 flush + Recovery 检查）
	Start(ctx context.Context) error

	// Stop 停止引擎（最终 flush + 清理）
	Stop() error
}

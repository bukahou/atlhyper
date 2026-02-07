// Package service 业务逻辑层
//
// Service 层封装业务逻辑，协调多个 Repository 调用。
// 上层 (Scheduler) 只依赖 Service 接口，不直接操作 Repository。
//
// 架构位置:
//
//	Scheduler
//	    ↓ 调用
//	Service (本包) ← 业务逻辑
//	    ↓ 调用
//	Repository     ← 数据访问
//	    ↓ 调用
//	SDK            ← K8s 客户端
//
// 主要服务:
//   - SnapshotService: 集群快照采集
//   - CommandService: 指令执行
//   - (SLO 数据已合入 SnapshotService，随快照一起采集)
package service

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v2"
)

// 注：指令使用 model_v2.Command（统一定义），结果使用 model.Result（Agent 内部）

// =============================================================================
// 快照服务
// =============================================================================

// SnapshotService 快照采集服务接口
//
// 负责采集集群所有资源，生成完整的 ClusterSnapshot。
// 内部并发采集多种资源类型，提高效率。
type SnapshotService interface {
	// Collect 采集集群快照
	//
	// 并发采集 16 种资源类型 (Pod, Node, Deployment 等)，
	// 汇总为完整快照，包含资源列表和统计摘要。
	//
	// 返回:
	//   - *ClusterSnapshot: 完整的集群快照
	//   - error: 第一个发生的错误 (部分失败时仍返回已采集的数据)
	Collect(ctx context.Context) (*model_v2.ClusterSnapshot, error)
}

// =============================================================================
// 指令服务
// =============================================================================

// CommandService 指令执行服务接口
//
// 负责解析和执行来自 Master 的指令。
// 根据 Command.Action 路由到对应的处理逻辑。
type CommandService interface {
	// Execute 执行单个指令
	//
	// 支持的 Action:
	//   - scale: 扩缩容 Deployment
	//   - restart: 重启 Deployment
	//   - delete: 删除资源
	//   - get_logs: 获取 Pod 日志
	//   - dynamic: 动态 API 调用
	//
	// 参数:
	//   - cmd: 待执行的指令 (使用 model_v2.Command 统一定义)
	//
	// 返回:
	//   - *Result: 执行结果 (始终返回，不返回 error)
	Execute(ctx context.Context, cmd *model_v2.Command) *model.Result
}



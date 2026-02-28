// atlhyper_master_v2/service/query/impl.go
// QueryService 结构体与构造函数
//
// 各功能域实现分布在:
//   k8s.go        — K8s 资源快照查询 (19 个方法)
//   otel.go       — OTel 快照/时间线查询
//   overview.go   — 集群概览、Agent 状态、事件、单资源查询
//   slo.go        — SLO 服务网格查询
//   aiops.go      — AIOps 查询与 AI 增强
package query

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	aiopsai "AtlHyper/atlhyper_master_v2/aiops/ai"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/mq"
)

// QueryService Query 层实现
type QueryService struct {
	store       datahub.Store
	bus         mq.Producer
	eventRepo   database.ClusterEventRepository
	aiopsEngine aiops.Engine
	aiopsAI     *aiopsai.Enhancer
}

// New 创建 QueryService 实例
func New(store datahub.Store, bus mq.Producer) *QueryService {
	return &QueryService{
		store: store,
		bus:   bus,
	}
}

// NewWithEventRepo 创建带事件仓库的 QueryService 实例（用于 Alert Trends）
func NewWithEventRepo(store datahub.Store, bus mq.Producer, eventRepo database.ClusterEventRepository) *QueryService {
	return &QueryService{
		store:     store,
		bus:       bus,
		eventRepo: eventRepo,
	}
}

// SetAIOpsEngine 注入 AIOps 引擎（可选）
func (q *QueryService) SetAIOpsEngine(engine aiops.Engine) {
	q.aiopsEngine = engine
}

// SetAIOpsAI 注入 AIOps AI 增强服务（可选）
func (q *QueryService) SetAIOpsAI(enhancer *aiopsai.Enhancer) {
	q.aiopsAI = enhancer
}

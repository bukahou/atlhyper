// Package tools AI 可调用的全部工具：每个工具在这里定义一次（参数结构体 → schema、权限、处理函数），
// 由 master.go 装配进 ai 服务的注册表。处理函数失败时返回错误（对话里显示为失败，模型也知道失败了），
// ⛔ 不把失败写成一段文字当作正常结果返回。
package tools

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// CommandRunner 经 Agent 执行只读指令并等结果（operations.CommandService 满足）
type CommandRunner interface {
	ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error)
}

// AIOpsReader AIOps 引擎的只读查询
type AIOpsReader interface {
	GetClusterRisk(clusterID string) *aiops.ClusterRisk
	GetEntityRisks(clusterID, sortBy string, limit int) []*aiops.EntityRisk
	GetEntityRisk(clusterID, entityKey string) *aiops.EntityRiskDetail
	GetIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error)
}

// SnapshotReader 集群快照（入口 SLO 从这里读）
type SnapshotReader interface {
	GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error)
}

// ClusterLister 正在上报的集群（service.Query 满足）
type ClusterLister interface {
	ListClusters(ctx context.Context) ([]agent.ClusterInfo, error)
}

// OverviewReader 集群概览；集群没有数据时返回 (nil, nil)（service.Query 满足）
type OverviewReader interface {
	GetOverview(ctx context.Context, clusterID string) (*cluster.ClusterOverview, error)
}

// Deps 工具用到的依赖。可选的依赖为 nil 时对应工具不注册（清单里也就没有它）
type Deps struct {
	Clusters ClusterLister
	Overview OverviewReader

	Commands    CommandRunner
	ToolTimeout time.Duration
	AIOps       AIOpsReader
	// SummarizeIncident 事件 AI 总结（AIOps 增强器）
	SummarizeIncident func(ctx context.Context, incidentID string) (any, error)
	Snapshots         SnapshotReader
	Events            EventLogReader

	GitHub      github.Client // nil = 不注册 GitHub 工具
	GitHubScope *GitHubRepoScope
}

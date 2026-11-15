// atlhyper_master_v2/service/interfaces.go
// Service 接口定义
// Query: 只读查询 (datahub.Store + mq.Producer 读取)
// Ops:   写入操作 (mq.Producer 写入)
package service

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// ================================================================
// 子接口（按功能域划分）
// ================================================================

// QueryK8s K8s 资源快照查询
type QueryK8s interface {
	GetSnapshot(ctx context.Context, clusterID string) (*cluster.ClusterSnapshot, error)
	GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]cluster.Pod, error)
	GetNodes(ctx context.Context, clusterID string) ([]cluster.Node, error)
	GetDeployments(ctx context.Context, clusterID string, namespace string) ([]cluster.Deployment, error)
	GetServices(ctx context.Context, clusterID string, namespace string) ([]cluster.Service, error)
	GetIngresses(ctx context.Context, clusterID string, namespace string) ([]cluster.Ingress, error)
	GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]cluster.ConfigMap, error)
	GetSecrets(ctx context.Context, clusterID string, namespace string) ([]cluster.Secret, error)
	GetNamespaces(ctx context.Context, clusterID string) ([]cluster.Namespace, error)
	GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]cluster.DaemonSet, error)
	GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]cluster.StatefulSet, error)
	GetJobs(ctx context.Context, clusterID string, namespace string) ([]cluster.Job, error)
	GetCronJobs(ctx context.Context, clusterID string, namespace string) ([]cluster.CronJob, error)
	GetPersistentVolumes(ctx context.Context, clusterID string) ([]cluster.PersistentVolume, error)
	GetPersistentVolumeClaims(ctx context.Context, clusterID string, namespace string) ([]cluster.PersistentVolumeClaim, error)
	GetNetworkPolicies(ctx context.Context, clusterID string, namespace string) ([]cluster.NetworkPolicy, error)
	GetResourceQuotas(ctx context.Context, clusterID string, namespace string) ([]cluster.ResourceQuota, error)
	GetLimitRanges(ctx context.Context, clusterID string, namespace string) ([]cluster.LimitRange, error)
	GetServiceAccounts(ctx context.Context, clusterID string, namespace string) ([]cluster.ServiceAccount, error)
}

// QueryOTel OTel 快照/时间线查询
type QueryOTel interface {
	GetOTelSnapshot(ctx context.Context, clusterID string) (*cluster.OTelSnapshot, error)
	GetOTelTimeline(ctx context.Context, clusterID string, since time.Time) ([]cluster.OTelEntry, error)
	// GetHardwareHealth 硬件健康矩阵（阈值判定已在 Master 完成，前端只渲染）
	GetHardwareHealth(ctx context.Context, clusterID string) (*model.HardwareHealthResponse, error)
	// GetNodeComparison 节点横向对比表（硬件列复用 GetHardwareHealth 的判定）
	GetNodeComparison(ctx context.Context, clusterID string) (*model.NodeComparisonResponse, error)
	// GetSignalFreshness 各信号的数据新鲜度，用于区分「没有流量」和「采集挂了」
	GetSignalFreshness(ctx context.Context, clusterID string) (*model.FreshnessResponse, error)
}

// QuerySLO SLO 服务网格查询 + SLO 目标/路由映射查询
type QuerySLO interface {
	// SLO 目标查询（返回 model 类型，非 database 类型）
	GetSLOTargets(ctx context.Context, clusterID string) ([]model.SLOTargetResponse, error)
	// SLO 路由映射查询（返回 model 类型，非 database 类型）
	GetSLORouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*model.SLORouteMapping, error)
	GetSLORouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*model.SLORouteMapping, error)
	GetSLOAllDomains(ctx context.Context, clusterID string) ([]string, error)
}

// QueryAIOps AIOps 查询与 AI 增强
type QueryAIOps interface {
	GetAIOpsGraph(ctx context.Context, clusterID string) (*aiops.DependencyGraph, error)
	GetAIOpsGraphTrace(ctx context.Context, clusterID, fromKey, direction string, maxDepth int) (*aiops.TraceResult, error)
	GetAIOpsBaseline(ctx context.Context, clusterID, entityKey string) (*aiops.EntityBaseline, error)
	GetAIOpsClusterRisk(ctx context.Context, clusterID string) (*aiops.ClusterRisk, error)
	GetAIOpsEntityRisks(ctx context.Context, clusterID, sortBy string, limit int) ([]*aiops.EntityRisk, error)
	GetAIOpsEntityRisk(ctx context.Context, clusterID, entityKey string) (*aiops.EntityRiskDetail, error)
	GetAIOpsIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error)
	GetAIOpsIncidentDetail(ctx context.Context, incidentID string) (*aiops.IncidentDetail, error)
	GetAIOpsIncidentStats(ctx context.Context, clusterID string, since time.Time) (*aiops.IncidentStats, error)
	GetAIOpsIncidentPatterns(ctx context.Context, clusterID, entityKey string, since time.Time) ([]*aiops.IncidentPattern, error)
	SummarizeIncident(ctx context.Context, incidentID string) (*enricher.SummarizeResponse, error)
	// AI 报告查询
	ListAIReports(ctx context.Context, incidentID string) ([]*database.AIReport, error)
	GetAIReport(ctx context.Context, id int64) (*database.AIReport, error)
}

// QueryOverview 集群概览、Agent 状态、事件、单资源查询
type QueryOverview interface {
	ListClusters(ctx context.Context) ([]agent.ClusterInfo, error)
	GetAgentStatus(ctx context.Context, clusterID string) (*agent.AgentStatus, error)
	GetCommandStatus(ctx context.Context, commandID string) (*command.Status, error)
	GetOverview(ctx context.Context, clusterID string) (*cluster.ClusterOverview, error)
	GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]cluster.Event, error)
	GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]cluster.Event, error)
}

// QueryAdmin 管理查询（审计日志、命令历史、事件历史、设置、AI Provider）
type QueryAdmin interface {
	// Audit
	ListAuditLogs(ctx context.Context, opts database.AuditQueryOpts) ([]*database.AuditLog, error)
	CountAuditLogs(ctx context.Context, opts database.AuditQueryOpts) (int64, error)
	// Command History
	ListCommandHistory(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error)
	CountCommandHistory(ctx context.Context, opts database.CommandQueryOpts) (int64, error)
	// Event History
	ListEventHistory(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error)
	CountEventHistory(ctx context.Context, clusterID string) (int64, error)
	// AI Provider
	ListAIProviders(ctx context.Context) ([]*database.AIProvider, error)
	GetAIProviderByID(ctx context.Context, id int64) (*database.AIProvider, error)
	GetAISettings(ctx context.Context) (*database.AISettings, error)
	ListAIModels(ctx context.Context) ([]*database.AIProviderModel, error)
	// AI Role Budget
	ListAIRoleBudgets(ctx context.Context) ([]*database.AIRoleBudget, error)
	// AI Reports (调用历史)
	ListRecentAIReports(ctx context.Context, role string, limit, offset int) ([]*database.AIReport, int, error)
}

// QueryDeploy 部署与 GitHub 集成的读取能力。
//
// 独立成域（而非塞进 QueryAdmin）与 QueryOTel/QuerySLO 的划分一致：
// 按功能域拆分，让 handler 能只依赖自己用得到的那一域。
type QueryDeploy interface {
	// GetDeployConfig 取某集群的 GitOps 部署配置
	GetDeployConfig(ctx context.Context, clusterID string) (*database.DeployConfig, error)
	// ListDeployHistory 部署历史（带分页/过滤）
	ListDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) ([]*database.DeployHistory, error)
	// CountDeployHistory 部署历史总数（与 List 同条件，用于分页）
	CountDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) (int, error)
	// GetGitHubInstallation 取 GitHub App 安装信息（全局单例）
	GetGitHubInstallation(ctx context.Context) (*database.GitHubInstallation, error)
	// ListRepoConfigs 已配置的仓库列表
	ListRepoConfigs(ctx context.Context) ([]*database.RepoConfig, error)
}

// QueryUser 用户读取能力。
//
// 密码校验、权限判定等业务逻辑保留在 handler —— service 层只负责
// 数据存取，不承担认证语义（改动最小，且不把认证逻辑分散到两处）。
type QueryUser interface {
	GetUserByID(ctx context.Context, id int64) (*database.User, error)
	GetUserByUsername(ctx context.Context, username string) (*database.User, error)
	// GetUserByUsernameForPasswordLogin 密码登录专用：外部身份账号视为不存在
	GetUserByUsernameForPasswordLogin(ctx context.Context, username string) (*database.User, error)
	ListUsers(ctx context.Context) ([]*database.User, error)
	// FindUserIDByIdentity 按外部身份（issuer + subject）查本地用户 ID
	FindUserIDByIdentity(ctx context.Context, issuer, subject string) (int64, bool, error)
	// ListFederatedUserIDs 拥有外部身份的用户 ID 集合
	ListFederatedUserIDs(ctx context.Context) (map[int64]bool, error)
}

// QueryClusterRegistry 集群管理页的读取能力
type QueryClusterRegistry interface {
	// ListClusterRegistry 全部行（已登记的 + 正在上报但未登记的），按集群 ID 排序
	ListClusterRegistry(ctx context.Context) ([]model.ClusterRegistryItem, error)
	// GetClusterRegistryItem 单个集群的那一行；既没登记也没上报过时返回 (nil, nil)
	GetClusterRegistryItem(ctx context.Context, clusterID string) (*model.ClusterRegistryItem, error)
}

// OpsClusterRegistry 集群注册表的写入能力。
//
// 输入不合规 → model.ErrInvalidClusterInput；集群 ID 已登记 → database.ErrClusterExists；
// 集群未登记 → database.ErrClusterNotFound
type OpsClusterRegistry interface {
	RegisterCluster(ctx context.Context, req *model.RegisterClusterRequest) error
	UpdateClusterProfile(ctx context.Context, clusterID string, req *model.UpdateClusterProfileRequest) error
	// IssueClusterToken 签发（或轮换）集群令牌；明文只在返回值里出现这一次
	IssueClusterToken(ctx context.Context, clusterID string) (*model.ClusterTokenIssued, error)
	// UnregisterCluster 删除登记，令牌随之失效；不删除该集群的历史数据
	UnregisterCluster(ctx context.Context, clusterID string) error
}

// QueryNotify 通知设置页的读取能力。时间一律 RFC 3339（UTC），列表恒为非 nil；凭据不在任何返回值里
type QueryNotify interface {
	GetNotifyPolicy(ctx context.Context) (*model.NotifyPolicy, error)
	// ListNotifyAlerts / ListNotifyDeliveries 返回当前页与符合条件的总数
	ListNotifyAlerts(ctx context.Context, opts model.NotifyAlertQuery) ([]model.NotifyAlert, int, error)
	ListNotifyDeliveries(ctx context.Context, opts model.NotifyDeliveryQuery) ([]model.NotifyDelivery, int, error)
	ListNotifyReceivers(ctx context.Context) ([]model.NotifyReceiver, error)
	// GetNotifyReceiver / GetNotifyRule / GetNotifySilence 不存在时返回 (nil, nil)
	GetNotifyReceiver(ctx context.Context, id int64) (*model.NotifyReceiver, error)
	// ListNotifyRules 按 position 从小到大
	ListNotifyRules(ctx context.Context) ([]model.NotifyRule, error)
	GetNotifyRule(ctx context.Context, id int64) (*model.NotifyRule, error)
	// ListNotifySilences state 为 pending / active / expired，空 = 全部
	ListNotifySilences(ctx context.Context, state string) ([]model.NotifySilence, error)
	GetNotifySilence(ctx context.Context, id int64) (*model.NotifySilence, error)
	// MatchNotifyRule 试匹配：这样一条告警会命中哪条规则、发给谁
	MatchNotifyRule(ctx context.Context, req *model.NotifyRuleMatchRequest) (*model.NotifyRuleMatchResult, error)
}

// OpsNotify 通知设置的写入能力。
//
// 输入不合规 → model.ErrInvalidNotifyInput；不存在 → database.ErrNotify*NotFound；
// 接收人重名 → database.ErrNotifyReceiverNameTaken；删除仍被引用的接收人 → *model.NotifyReceiverInUseError
type OpsNotify interface {
	UpdateNotifyPolicy(ctx context.Context, req *model.UpdateNotifyPolicyRequest) error
	CreateNotifyReceiver(ctx context.Context, req *model.NotifyReceiverRequest) (int64, error)
	// UpdateNotifyReceiver 凭据留空 = 保持原值
	UpdateNotifyReceiver(ctx context.Context, id int64, req *model.NotifyReceiverRequest) error
	DeleteNotifyReceiver(ctx context.Context, id int64) error
	CreateNotifyRule(ctx context.Context, req *model.NotifyRuleRequest) (int64, error)
	UpdateNotifyRule(ctx context.Context, id int64, req *model.NotifyRuleRequest) error
	DeleteNotifyRule(ctx context.Context, id int64) error
	// ReorderNotifyRules ids 必须恰好是全部规则的一个排列，否则 database.ErrNotifyRuleOrderMismatch
	ReorderNotifyRules(ctx context.Context, ids []int64) error
	CreateNotifySilence(ctx context.Context, req *model.CreateNotifySilenceRequest, createdBy string) (int64, error)
	// ExpireNotifySilence 提前结束（还没开始的也可以）
	ExpireNotifySilence(ctx context.Context, id int64) error
}

// OpsAdmin 管理写入操作（设置、AI Provider）
type OpsAdmin interface {
	CreateAIProvider(ctx context.Context, p *database.AIProvider) error
	UpdateAIProvider(ctx context.Context, p *database.AIProvider) error
	DeleteAIProvider(ctx context.Context, id int64) error
	UpdateAISettings(ctx context.Context, cfg *database.AISettings) error
	UpdateAIProviderRoles(ctx context.Context, id int64, roles []string) error
	UpdateAIRoleBudget(ctx context.Context, budget *database.AIRoleBudget) error
}

// OpsDeploy 部署与 GitHub 集成的写入能力。
type OpsDeploy interface {
	// UpsertDeployConfig 写入/更新集群的 GitOps 部署配置
	UpsertDeployConfig(ctx context.Context, cfg *database.DeployConfig) error
	// UpsertGitHubInstallation 写入/更新 GitHub App 安装信息
	UpsertGitHubInstallation(ctx context.Context, inst *database.GitHubInstallation) error
	// DeleteGitHubInstallation 解除 GitHub App 安装
	DeleteGitHubInstallation(ctx context.Context) error
}

// OpsUser 用户写入能力。
type OpsUser interface {
	CreateUser(ctx context.Context, user *database.User) error
	UpdateUser(ctx context.Context, user *database.User) error
	DeleteUser(ctx context.Context, id int64) error
	UpdateUserLastLogin(ctx context.Context, id int64, ip string) error
	// CreateFederatedUser 同一事务内建用户与外部身份行
	CreateFederatedUser(ctx context.Context, user *database.User, issuer, subject string) error
}

// ================================================================
// 组合接口（向后兼容，现有代码无需修改）
// ================================================================

// Query 只读查询接口
type Query interface {
	QueryK8s
	QueryOTel
	QuerySLO
	QueryAIOps
	QueryOverview
	QueryAdmin
	QueryDeploy
	QueryUser
	QueryClusterRegistry
	QueryNotify
}

// OpsSLO SLO 写入操作
type OpsSLO interface {
	UpsertSLOTarget(ctx context.Context, req *model.UpdateSLOTargetRequest) error
}

// Ops 写入操作接口
type Ops interface {
	OpsDeploy
	OpsUser
	CreateCommand(req *model.CreateCommandRequest) (*model.CreateCommandResponse, error)
	// ExecuteCommandSync 同步执行指令（创建 + 等待 Agent 结果）
	ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error)
	OpsAdmin
	OpsSLO
	OpsClusterRegistry
	OpsNotify
}

// Service 组合接口 (master.go 持有)
type Service interface {
	Query
	Ops
}

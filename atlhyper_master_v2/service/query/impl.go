// atlhyper_master_v2/service/query/impl.go
// Query 层的对外唯一入口与内部组合。
//
// 对外：NewQueryService(deps) 返回 *QueryService，满足 service.Query（Facade，调用方只认它）。
// 对内：按功能域拆成 10 个小服务，各自只持有用到的依赖；*QueryService 嵌入它们，方法经嵌入提升。
// 新增功能域 = 新增一个小服务 + 在这里嵌入一行，调用方不用改。
//
// 各功能域的实现文件:
//
//	K8sQuery              k8s.go
//	OTelQuery             otel.go、metrics_compare.go、metrics_hardware.go、observe_freshness.go
//	SLOQuery              slo.go
//	AIOpsQuery            aiops.go、aiops_incident_view.go
//	OverviewQuery         overview.go
//	AdminQuery            admin.go
//	GitHubQuery           github.go
//	UserQuery             user.go
//	ClusterRegistryQuery  cluster_registry.go
//	NotifyQuery           notify.go
package query

import (
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/mq"
)

// QueryService Query 层的对外入口：10 个功能域小服务的组合
type QueryService struct {
	*K8sQuery
	*OTelQuery
	*SLOQuery
	*AIOpsQuery
	*OverviewQuery
	*AdminQuery
	*GitHubQuery
	*UserQuery
	*ClusterRegistryQuery
	*NotifyQuery
}

// ==================== 功能域小服务 ====================

// K8sQuery K8s 资源快照查询（满足 service.QueryK8s）
type K8sQuery struct {
	store datahub.Store
}

// NewK8sQuery 创建 K8sQuery
func NewK8sQuery(store datahub.Store) *K8sQuery {
	return &K8sQuery{store: store}
}

// OTelQuery OTel 快照 / 时间线与据此算出的指标视图：硬件健康、节点对比、信号新鲜度（满足 service.QueryOTel）
type OTelQuery struct {
	store datahub.Store
}

// NewOTelQuery 创建 OTelQuery
func NewOTelQuery(store datahub.Store) *OTelQuery {
	return &OTelQuery{store: store}
}

// SLOQuery SLO 服务网格与路由映射查询（满足 service.QuerySLO）
type SLOQuery struct {
	sloRepo database.SLORepository
}

// NewSLOQuery 创建 SLOQuery
func NewSLOQuery(sloRepo database.SLORepository) *SLOQuery {
	return &SLOQuery{sloRepo: sloRepo}
}

// AIOpsQuery AIOps 风险、依赖图、基线、事件与 AI 报告查询（满足 service.QueryAIOps）；事件详情按快照补成员状态
type AIOpsQuery struct {
	store        datahub.Store
	aiopsEngine  aiops.Engine
	aiopsAI      *enricher.Enricher
	aiReportRepo database.AIReportRepository
}

// NewAIOpsQuery 创建 AIOpsQuery
func NewAIOpsQuery(store datahub.Store, aiopsEngine aiops.Engine, aiopsAI *enricher.Enricher, aiReportRepo database.AIReportRepository) *AIOpsQuery {
	return &AIOpsQuery{store: store, aiopsEngine: aiopsEngine, aiopsAI: aiopsAI, aiReportRepo: aiReportRepo}
}

// OverviewQuery 集群概览、Agent 状态、指令状态与 K8s 事件查询（满足 service.QueryOverview）
type OverviewQuery struct {
	store     datahub.Store
	bus       mq.Producer
	eventRepo database.ClusterEventRepository
}

// NewOverviewQuery 创建 OverviewQuery
func NewOverviewQuery(store datahub.Store, bus mq.Producer, eventRepo database.ClusterEventRepository) *OverviewQuery {
	return &OverviewQuery{store: store, bus: bus, eventRepo: eventRepo}
}

// AdminQuery 审计、指令历史、事件历史与 AI 设置查询（满足 service.QueryAdmin）
type AdminQuery struct {
	eventRepo      database.ClusterEventRepository
	auditRepo      database.AuditRepository
	commandRepo    database.CommandHistoryRepository
	aiProviderRepo database.AIProviderRepository
	aiSettingsRepo database.AISettingsRepository
	aiModelRepo    database.AIProviderModelRepository
	aiBudgetRepo   database.AIRoleBudgetRepository
	aiReportRepo   database.AIReportRepository
}

// NewAdminQuery 创建 AdminQuery
func NewAdminQuery(eventRepo database.ClusterEventRepository, auditRepo database.AuditRepository, commandRepo database.CommandHistoryRepository, aiProviderRepo database.AIProviderRepository, aiSettingsRepo database.AISettingsRepository, aiModelRepo database.AIProviderModelRepository, aiBudgetRepo database.AIRoleBudgetRepository, aiReportRepo database.AIReportRepository) *AdminQuery {
	return &AdminQuery{eventRepo: eventRepo, auditRepo: auditRepo, commandRepo: commandRepo, aiProviderRepo: aiProviderRepo, aiSettingsRepo: aiSettingsRepo, aiModelRepo: aiModelRepo, aiBudgetRepo: aiBudgetRepo, aiReportRepo: aiReportRepo}
}

// GitHubQuery GitHub 集成查询：App 安装信息与仓库配置（满足 service.QueryGitHub）
type GitHubQuery struct {
	githubInstallRepo database.GitHubInstallationRepository
	repoConfigRepo    database.RepoConfigRepository
}

// NewGitHubQuery 创建 GitHubQuery
func NewGitHubQuery(githubInstallRepo database.GitHubInstallationRepository, repoConfigRepo database.RepoConfigRepository) *GitHubQuery {
	return &GitHubQuery{githubInstallRepo: githubInstallRepo, repoConfigRepo: repoConfigRepo}
}

// UserQuery 用户与外部身份查询（满足 service.QueryUser）
type UserQuery struct {
	userRepo     database.UserRepository
	identityRepo database.UserIdentityRepository
}

// NewUserQuery 创建 UserQuery
func NewUserQuery(userRepo database.UserRepository, identityRepo database.UserIdentityRepository) *UserQuery {
	return &UserQuery{userRepo: userRepo, identityRepo: identityRepo}
}

// ClusterRegistryQuery 集群管理页：注册表 + 在线状态 + 认证状态（满足 service.QueryClusterRegistry）
type ClusterRegistryQuery struct {
	store             datahub.Store
	clusterRepo       ClusterRegistryReader
	clusterAuthStatus ClusterAuthStatus
}

// NewClusterRegistryQuery 创建 ClusterRegistryQuery
func NewClusterRegistryQuery(store datahub.Store, clusterRepo ClusterRegistryReader, clusterAuthStatus ClusterAuthStatus) *ClusterRegistryQuery {
	return &ClusterRegistryQuery{store: store, clusterRepo: clusterRepo, clusterAuthStatus: clusterAuthStatus}
}

// NotifyQuery 通知设置页查询：策略、发件设置、告警、送达记录、接收人、路由规则、静默（满足 service.QueryNotify）
type NotifyQuery struct {
	notify NotifyRepos
	now    func() time.Time
}

// NewNotifyQuery 创建 NotifyQuery
func NewNotifyQuery(notify NotifyRepos, now func() time.Time) *NotifyQuery {
	return &NotifyQuery{notify: notify, now: now}
}

// ==================== 依赖与构造 ====================

// AdminRepos 管理查询所需的 Repository 集合
// 对应 QueryAdmin 接口的所有方法所需依赖
type AdminRepos struct {
	Audit      database.AuditRepository
	Command    database.CommandHistoryRepository
	AIProvider database.AIProviderRepository
	AISettings database.AISettingsRepository
	AIModel    database.AIProviderModelRepository
	AIBudget   database.AIRoleBudgetRepository
	AIReport   database.AIReportRepository
}

// GitHubRepos GitHub 集成所需的 Repository 集合
// 对应 QueryGitHub 接口的所有方法所需依赖
type GitHubRepos struct {
	GitHubInstall database.GitHubInstallationRepository
	RepoConfig    database.RepoConfigRepository
}

// QueryServiceDeps QueryService 全部依赖
// 严格限定：每个字段都被某个功能域小服务用到，禁止新增未使用的依赖
type QueryServiceDeps struct {
	Store        datahub.Store                   // 必需
	Bus          mq.Producer                     // 必需
	EventRepo    database.ClusterEventRepository // 必需（Alert Trends）
	SLORepo      database.SLORepository          // 必需（Phase 2 新增）
	AIOpsEngine  aiops.Engine                    // 可选，nil = AIOps 查询返回空
	AIOpsAI      *enricher.Enricher              // 可选，nil = AI 增强禁用
	AdminRepos   AdminRepos                      // 必需（管理查询）
	GitHubRepos  GitHubRepos                     // 必需（GitHub 集成）
	UserRepo     database.UserRepository         // 必需（用户管理）
	IdentityRepo database.UserIdentityRepository // 必需（外部身份）

	ClusterRepo       ClusterRegistryReader // 必需（集群管理页）
	ClusterAuthStatus ClusterAuthStatus     // 可选，nil = 认证状态一律显示为未认证

	NotifyRepos NotifyRepos // 必需（通知设置页）
}

// NewQueryService 创建 QueryService：按依赖构造各功能域小服务并组合
func NewQueryService(deps QueryServiceDeps) *QueryService {
	return &QueryService{
		K8sQuery:      NewK8sQuery(deps.Store),
		OTelQuery:     NewOTelQuery(deps.Store),
		SLOQuery:      NewSLOQuery(deps.SLORepo),
		AIOpsQuery:    NewAIOpsQuery(deps.Store, deps.AIOpsEngine, deps.AIOpsAI, deps.AdminRepos.AIReport),
		OverviewQuery: NewOverviewQuery(deps.Store, deps.Bus, deps.EventRepo),
		AdminQuery: NewAdminQuery(deps.EventRepo, deps.AdminRepos.Audit, deps.AdminRepos.Command,
			deps.AdminRepos.AIProvider, deps.AdminRepos.AISettings, deps.AdminRepos.AIModel,
			deps.AdminRepos.AIBudget, deps.AdminRepos.AIReport),
		GitHubQuery:          NewGitHubQuery(deps.GitHubRepos.GitHubInstall, deps.GitHubRepos.RepoConfig),
		UserQuery:            NewUserQuery(deps.UserRepo, deps.IdentityRepo),
		ClusterRegistryQuery: NewClusterRegistryQuery(deps.Store, deps.ClusterRepo, deps.ClusterAuthStatus),
		NotifyQuery:          NewNotifyQuery(deps.NotifyRepos, nil),
	}
}

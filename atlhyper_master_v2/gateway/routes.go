// atlhyper_master_v2/gateway/routes.go
// API 路由统一注册
// 所有 API 在此集中管理，便于查看和维护
//
// 权限策略（设计文档 docs/design/active/permission-model-design.md）：
//   - 每条路由按（方法，路径）声明所需权限，权限表见 middleware/permission.go
//   - 同一路径的读与写分别声明：写方法不继承读方法的权限；没声明的方法由 ServeMux 直接 405
//   - 访客（未登录）与新注册成员同为 0 级，只能看到集群状态与形状
//   - 不需要任何主体的路由（健康检查、登录、第三方回跳）用 open 注册
package gateway

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/auth/oidc"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/deployer"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	adminHandler "AtlHyper/atlhyper_master_v2/gateway/handler/admin"
	aiopsHandler "AtlHyper/atlhyper_master_v2/gateway/handler/aiops"
	k8sHandler "AtlHyper/atlhyper_master_v2/gateway/handler/k8s"
	observeHandler "AtlHyper/atlhyper_master_v2/gateway/handler/observe"
	settingsHandler "AtlHyper/atlhyper_master_v2/gateway/handler/settings"
	sloHandler "AtlHyper/atlhyper_master_v2/gateway/handler/slo"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/service"
)

// 权限常量别名，让路由表读起来像一张表
type perm = middleware.Permission

const (
	pClusterRead    = middleware.PermClusterRead
	pEventsRead     = middleware.PermEventsRead
	pMetricsRead    = middleware.PermMetricsRead
	pSLORead        = middleware.PermSLORead
	pAPMSummaryRead = middleware.PermAPMSummaryRead
	pLogsStatsRead  = middleware.PermLogsStatsRead
	pAIOpsRead      = middleware.PermAIOpsRead

	pLogsRead          = middleware.PermLogsRead
	pTracesRead        = middleware.PermTracesRead
	pCommandsRead      = middleware.PermCommandsRead
	pDeployHistoryRead = middleware.PermDeployHistoryRead
	pGitHubStatusRead  = middleware.PermGitHubStatusRead
	pAIChat            = middleware.PermAIChat

	pSecretRead     = middleware.PermSecretRead
	pPodLogsRead    = middleware.PermPodLogsRead
	pClusterOperate = middleware.PermClusterOperate
	pAuditRead      = middleware.PermAuditRead
	pNotifyManage   = middleware.PermNotifyManage
	pSLOManage      = middleware.PermSLOManage
	pAIAnalyze      = middleware.PermAIAnalyze
	pAISettingsRead = middleware.PermAISettingsRead

	pUserManage      = middleware.PermUserManage
	pAISettingsWrite = middleware.PermAISettingsWrite
	pGitHubManage    = middleware.PermGitHubManage
	pDeployManage    = middleware.PermDeployManage
)

// 方法别名，同上
const (
	mGet    = http.MethodGet // ServeMux 的 GET 模式同时匹配 HEAD
	mPost   = http.MethodPost
	mPut    = http.MethodPut
	mPatch  = http.MethodPatch
	mDelete = http.MethodDelete
)

// routeSpec 一条路由声明：同一路径的不同方法各自声明权限与是否审计。
// routes_test.go 遍历全部声明，守住「写方法不挂 0 级权限」「2 级以上的写一律审计」等规则。
type routeSpec struct {
	method  string
	path    string
	perm    perm // 空 = 无需主体
	audited bool
}

// Router 路由管理器
type Router struct {
	specs          []routeSpec
	mux            *http.ServeMux
	service        service.Service
	database       *database.DB
	aiService      ai.AIService
	analyzeTrigger aiopsHandler.AnalyzeTrigger
	ghClient       github.Client
	deployer       deployer.Deployer
	oidc           *oidc.Module // 可选，nil = 外部身份登录未启用
}

// SetOIDC 注入外部身份登录模块（可选）
func (r *Router) SetOIDC(m *oidc.Module) { r.oidc = m }

// NewRouter 创建路由管理器
func NewRouter(svc service.Service, db *database.DB, aiSvc ai.AIService, trigger aiopsHandler.AnalyzeTrigger, ghClient github.Client, dep deployer.Deployer) *Router {
	return &Router{
		mux:            http.NewServeMux(),
		service:        svc,
		database:       db,
		aiService:      aiSvc,
		analyzeTrigger: trigger,
		ghClient:       ghClient,
		deployer:       dep,
	}
}

// Handler 返回包装了中间件的 Handler
// 顺序: Logging -> CORS -> Authenticate（解析主体）-> mux（各路由自带权限检查）
func (r *Router) Handler() http.Handler {
	r.registerRoutes()
	return middleware.Logging(middleware.CORS(middleware.Authenticate(r.service)(r.mux)))
}

// registerRoutes 注册所有路由
func (r *Router) registerRoutes() {
	// 创建 Handlers — 顶层 (package handler)
	clusterH := handler.NewClusterHandler(r.service)
	overviewH := handler.NewOverviewHandler(r.service)
	eventH := handler.NewEventHandler(r.service)
	opsH := handler.NewOpsHandler(r.service)

	// 创建 Handlers — K8s 资源 (package k8s)
	podH := k8sHandler.NewPodHandler(r.service)
	nodeH := k8sHandler.NewNodeHandler(r.service)
	deploymentH := k8sHandler.NewDeploymentHandler(r.service)
	daemonsetH := k8sHandler.NewDaemonSetHandler(r.service)
	statefulsetH := k8sHandler.NewStatefulSetHandler(r.service)
	serviceH := k8sHandler.NewServiceHandler(r.service)
	ingressH := k8sHandler.NewIngressHandler(r.service)
	configmapH := k8sHandler.NewConfigMapHandler(r.service)
	secretH := k8sHandler.NewSecretHandler(r.service)
	namespaceH := k8sHandler.NewNamespaceHandler(r.service)
	jobH := k8sHandler.NewJobHandler(r.service)
	cronjobH := k8sHandler.NewCronJobHandler(r.service)
	pvH := k8sHandler.NewPVHandler(r.service)
	pvcH := k8sHandler.NewPVCHandler(r.service)
	networkPolicyH := k8sHandler.NewNetworkPolicyHandler(r.service)
	resourceQuotaH := k8sHandler.NewResourceQuotaHandler(r.service)
	limitRangeH := k8sHandler.NewLimitRangeHandler(r.service)
	serviceAccountH := k8sHandler.NewServiceAccountHandler(r.service)

	// 创建 Handlers — 可观测性 (package observe)
	nodeMetricsH := observeHandler.NewNodeMetricsHandler(r.service, r.service)
	observeH := observeHandler.NewObserveHandler(r.service, r.service)

	// 创建 Handlers — SLO (package slo)
	sloH := sloHandler.NewSLOHandler(r.service, r.service)

	// 创建 Handlers — AIOps (package aiops)
	aiopsGraphH := aiopsHandler.NewAIOpsGraphHandler(r.service)
	aiopsBaselineH := aiopsHandler.NewAIOpsBaselineHandler(r.service)
	aiopsRiskH := aiopsHandler.NewAIOpsRiskHandler(r.service)
	aiopsIncidentH := aiopsHandler.NewAIOpsIncidentHandler(r.service)
	aiopsAIH := aiopsHandler.NewAIOpsAIHandler(r.service)
	if r.analyzeTrigger != nil {
		aiopsAIH.SetAnalyzeTrigger(r.analyzeTrigger)
	}

	// 创建 Handlers — 管理 (package admin)
	userH := adminHandler.NewUserHandler(r.service, r.service)
	commandH := adminHandler.NewCommandHandler(r.service)
	notifyH := adminHandler.NewNotifyHandler(r.service)
	aiProviderH := adminHandler.NewAIProviderHandler(r.service)
	auditH := adminHandler.NewAuditHandler(r.service)

	authMeH := handler.NewAuthMeHandler(r.service)

	// ================================================================
	// 无需主体
	// ================================================================
	r.open(mGet, "/health", healthCheck)
	r.open(mGet, "/api/v2/auth/me", authMeH.Get)
	r.open(mGet, "/api/v2/auth/permissions", handler.GetPermissionTable)
	r.openAudited(mPost, "/api/v2/user/login", "login", "user", userH.Login) // 记录成功 / 失败的登录尝试

	// 外部身份登录（OIDC）：providers 恒注册（未启用时返回空列表，登录框据此隐藏按钮）
	r.open(mGet, "/api/v2/auth/providers", r.oidc.Providers)
	if r.oidc != nil {
		r.open(mGet, "/api/v2/auth/oidc/start", r.oidc.Start)
		r.openAudited(mGet, "/api/v2/auth/oidc/callback", "login", "user", r.oidc.Callback)
	}

	// ================================================================
	// 0 级：集群状态与形状（访客 = 成员）
	// ================================================================
	for pattern, h := range map[string]http.HandlerFunc{
		"/api/v2/overview":  overviewH.Get,
		"/api/v2/clusters":  clusterH.List,
		"/api/v2/clusters/": clusterH.Get,
		// 资源列表 / 详情
		"/api/v2/pods": podH.List, "/api/v2/pods/": podH.Get,
		"/api/v2/nodes": nodeH.List, "/api/v2/nodes/": nodeH.Get,
		"/api/v2/deployments": deploymentH.List, "/api/v2/deployments/": deploymentH.Get,
		"/api/v2/daemonsets": daemonsetH.List, "/api/v2/daemonsets/": daemonsetH.Get,
		"/api/v2/statefulsets": statefulsetH.List, "/api/v2/statefulsets/": statefulsetH.Get,
		"/api/v2/services": serviceH.List, "/api/v2/services/": serviceH.Get,
		"/api/v2/ingresses": ingressH.List, "/api/v2/ingresses/": ingressH.Get,
		"/api/v2/jobs": jobH.List, "/api/v2/jobs/": jobH.Get,
		"/api/v2/cronjobs": cronjobH.List, "/api/v2/cronjobs/": cronjobH.Get,
		"/api/v2/pvs": pvH.List, "/api/v2/pvs/": pvH.Get,
		"/api/v2/pvcs": pvcH.List, "/api/v2/pvcs/": pvcH.Get,
		"/api/v2/network-policies": networkPolicyH.List, "/api/v2/network-policies/": networkPolicyH.Get,
		"/api/v2/resource-quotas": resourceQuotaH.List, "/api/v2/resource-quotas/": resourceQuotaH.Get,
		"/api/v2/limit-ranges": limitRangeH.List, "/api/v2/limit-ranges/": limitRangeH.Get,
		"/api/v2/service-accounts": serviceAccountH.List, "/api/v2/service-accounts/": serviceAccountH.Get,
		"/api/v2/configmaps": configmapH.List, // 仅列表（名字与键数），详情见 2 级
		"/api/v2/namespaces": namespaceH.List, "/api/v2/namespaces/": namespaceH.Get,
	} {
		// 统一套 env 字面值裁剪：无 workload.env.read 时剥掉任意层级 envs[].value
		r.route(mGet, pClusterRead, pattern, middleware.RedactEnvValues(middleware.PermWorkloadEnvRead, h))
	}

	r.route(mGet, pEventsRead, "/api/v2/events", eventH.List)
	r.route(mGet, pEventsRead, "/api/v2/events/by-resource", eventH.ListByResource)

	r.route(mGet, pMetricsRead, "/api/v2/node-metrics", nodeMetricsH.Route)
	r.route(mGet, pMetricsRead, "/api/v2/node-metrics/", nodeMetricsH.Route)
	r.route(mGet, pMetricsRead, "/api/v2/observe/freshness", observeH.Freshness)
	r.route(mGet, pMetricsRead, "/api/v2/observe/metrics/summary", observeH.MetricsSummary)
	r.route(mGet, pMetricsRead, "/api/v2/observe/metrics/nodes", observeH.MetricsNodes)
	r.route(mGet, pMetricsRead, "/api/v2/observe/metrics/nodes/", observeH.MetricsNodeRoute)
	r.route(mGet, pMetricsRead, "/api/v2/observe/metrics/hardware", observeH.MetricsHardware)
	r.route(mGet, pMetricsRead, "/api/v2/observe/metrics/compare", observeH.MetricsCompare)

	r.route(mGet, pSLORead, "/api/v2/slo/domains", sloH.Domains)      // V1: 按 service key
	r.route(mGet, pSLORead, "/api/v2/slo/domains/v2", sloH.DomainsV2) // V2: 按真实域名
	r.route(mGet, pSLORead, "/api/v2/slo/domains/detail", sloH.DomainDetail)
	r.route(mGet, pSLORead, "/api/v2/slo/domains/history", sloH.DomainHistory)
	r.route(mGet, pSLORead, "/api/v2/slo/domains/latency", sloH.LatencyDistribution)
	r.route(mGet, pSLORead, "/api/v2/slo/targets", sloH.Targets)
	r.route(mGet, pSLORead, "/api/v2/slo/status-history", sloH.StatusHistory)
	r.route(mGet, pSLORead, "/api/v2/observe/slo/summary", observeH.SLOSummary)
	r.route(mGet, pSLORead, "/api/v2/observe/slo/ingress", observeH.SLOIngress)
	r.route(mGet, pSLORead, "/api/v2/observe/slo/timeseries", observeH.SLOTimeSeries)

	r.route(mGet, pAPMSummaryRead, "/api/v2/observe/traces/services", observeH.TracesServices)
	r.route(mGet, pAPMSummaryRead, "/api/v2/observe/traces/services/", observeH.APMServiceSeries)
	r.route(mGet, pAPMSummaryRead, "/api/v2/observe/traces/stats", observeH.TracesStats)
	r.route(mGet, pAPMSummaryRead, "/api/v2/observe/traces/topology", observeH.TracesTopology)
	r.route(mGet, pAPMSummaryRead, "/api/v2/observe/traces/operations", observeH.TracesOperations)

	r.route(mGet, pLogsStatsRead, "/api/v2/observe/logs/summary", observeH.LogsSummary)
	r.route(mGet, pLogsStatsRead, "/api/v2/observe/logs/histogram", observeH.LogsHistogram)

	r.route(mGet, pAIOpsRead, "/api/v2/aiops/graph", aiopsGraphH.Graph)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/graph/trace", aiopsGraphH.Trace)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/baseline", aiopsBaselineH.Baseline)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/risk/cluster", aiopsRiskH.ClusterRisk)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/risk/entities", aiopsRiskH.EntityRisks)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/risk/entity", aiopsRiskH.EntityRisk)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/incidents", aiopsIncidentH.List)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/incidents/stats", aiopsIncidentH.Stats)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/incidents/patterns", aiopsIncidentH.Patterns)
	r.route(mGet, pAIOpsRead, "/api/v2/aiops/incidents/", aiopsIncidentH.Detail)

	// ================================================================
	// 1 级：内容级数据
	// ================================================================
	r.route(mPost, pLogsRead, "/api/v2/observe/logs/query", observeH.LogsQuery) // 查询条件放请求体

	r.route(mGet, pTracesRead, "/api/v2/observe/traces", observeH.TracesList)
	r.route(mGet, pTracesRead, "/api/v2/observe/traces/", observeH.TracesDetail)
	r.route(mGet, pTracesRead, "/api/v2/observe/traces/correlations", observeH.TracesCorrelations) // 返回属性值

	r.route(mGet, pCommandsRead, "/api/v2/commands/history", commandH.ListHistory)
	r.route(mGet, pCommandsRead, "/api/v2/commands/", commandH.GetStatus)

	// AI 对话（SSE 流式响应的 WriteTimeout 在 handler 内处理；删除会话在 handler 内另需 Operator）
	if r.aiService != nil {
		aiH := aiopsHandler.NewAIHandler(r.aiService)
		r.route(mGet, pAIChat, "/api/v2/ai/conversations", aiH.Conversations)
		r.route(mPost, pAIChat, "/api/v2/ai/conversations", aiH.Conversations)
		r.route(mGet, pAIChat, "/api/v2/ai/conversations/", aiH.ConversationByID) // {id}/messages
		r.routeAudited(mDelete, pAIChat, "/api/v2/ai/conversations/", "delete", "ai_conversation", aiH.ConversationByID)
		r.route(mPost, pAIChat, "/api/v2/ai/chat", aiH.Chat)
	}

	// ================================================================
	// 2 级：凭据与写操作（写与敏感读一律审计，含权限不足的失败尝试）
	// ================================================================
	r.route(mGet, pSecretRead, "/api/v2/configmaps/", configmapH.Get)
	r.route(mGet, pSecretRead, "/api/v2/secrets", secretH.List)
	r.routeAudited(mPost, pSecretRead, "/api/v2/ops/configmaps/data", "read", "configmap", opsH.ConfigMapData)
	r.routeAudited(mPost, pSecretRead, "/api/v2/ops/secrets/data", "read", "secret", opsH.SecretData)

	r.routeAudited(mPost, pPodLogsRead, "/api/v2/ops/pods/logs", "read", "pod", opsH.PodLogs)

	r.routeAudited(mPost, pClusterOperate, "/api/v2/commands", "execute", "command", commandH.Create)
	r.routeAudited(mPost, pClusterOperate, "/api/v2/ops/pods/restart", "execute", "pod", opsH.PodRestart)
	r.routeAudited(mPost, pClusterOperate, "/api/v2/ops/deployments/scale", "execute", "deployment", opsH.DeploymentScale)
	r.routeAudited(mPost, pClusterOperate, "/api/v2/ops/deployments/restart", "execute", "deployment", opsH.DeploymentRestart)
	r.routeAudited(mPost, pClusterOperate, "/api/v2/ops/deployments/image", "execute", "deployment", opsH.DeploymentImage)
	r.routeAudited(mPost, pClusterOperate, "/api/v2/ops/nodes/cordon", "execute", "node", opsH.NodeCordon)
	r.routeAudited(mPost, pClusterOperate, "/api/v2/ops/nodes/uncordon", "execute", "node", opsH.NodeUncordon)

	r.route(mGet, pAuditRead, "/api/v2/audit/logs", auditH.List)

	r.route(mGet, pNotifyManage, "/api/v2/notify/channels", notifyH.ListChannels)
	r.route(mGet, pNotifyManage, "/api/v2/notify/channels/", notifyH.ChannelHandler)
	r.routeAudited(mPut, pNotifyManage, "/api/v2/notify/channels/", "update", "notify", notifyH.ChannelHandler)
	r.routeAudited(mPatch, pNotifyManage, "/api/v2/notify/channels/", "update", "notify", notifyH.ChannelHandler)
	r.routeAudited(mPost, pNotifyManage, "/api/v2/notify/channels/", "execute", "notify_test", notifyH.ChannelHandler) // {type}/test

	r.routeAudited(mPut, pSLOManage, "/api/v2/slo/targets", "update", "slo_target", sloH.Targets)

	// AIOps AI 增强（有 LLM 调用成本）
	r.routeAudited(mPost, pAIAnalyze, "/api/v2/aiops/ai/summarize", "execute", "ai_summary", aiopsAIH.Summarize)
	r.routeAudited(mPost, pAIAnalyze, "/api/v2/aiops/ai/recommend", "execute", "ai_recommendation", aiopsAIH.Recommend)
	r.route(mGet, pAIAnalyze, "/api/v2/aiops/ai/reports", aiopsAIH.ReportsHandler)
	r.routeAudited(mGet, pAIAnalyze, "/api/v2/aiops/ai/reports/", "read", "ai_report", aiopsAIH.ReportDetailHandler)
	r.routeAudited(mPost, pAIAnalyze, "/api/v2/aiops/ai/analyze", "execute", "ai_analysis", aiopsAIH.AnalyzeHandler)
	r.route(mGet, pAIAnalyze, "/api/v2/aiops/ai/analyze/status", aiopsAIH.AnalyzeStatus)

	r.route(mGet, pAISettingsRead, "/api/v2/ai/providers", aiProviderH.ProvidersHandler)
	r.route(mGet, pAISettingsRead, "/api/v2/ai/settings", aiProviderH.SettingsHandler)
	r.route(mGet, pAISettingsRead, "/api/v2/ai/roles", aiProviderH.RolesOverviewHandler)
	r.route(mGet, pAISettingsRead, "/api/v2/ai/budgets", aiProviderH.BudgetsHandler)
	r.route(mGet, pAISettingsRead, "/api/v2/ai/reports", aiProviderH.AIReportsHandler)

	// ================================================================
	// 3 级：系统配置（写一律审计）
	// ================================================================
	r.route(mGet, pUserManage, "/api/v2/user/list", userH.List)
	r.routeAudited(mPost, pUserManage, "/api/v2/user/register", "create", "user", userH.Register)
	r.routeAudited(mPost, pUserManage, "/api/v2/user/update-role", "update", "user", userH.UpdateRole)
	r.routeAudited(mPost, pUserManage, "/api/v2/user/update-status", "update", "user", userH.UpdateStatus)
	r.routeAudited(mPost, pUserManage, "/api/v2/user/delete", "delete", "user", userH.Delete)

	r.routeAudited(mPost, pAISettingsWrite, "/api/v2/ai/providers", "create", "ai_provider", aiProviderH.ProvidersHandler)
	r.route(mGet, pAISettingsWrite, "/api/v2/ai/providers/", aiProviderH.ProviderHandler) // 单个 Provider 详情
	r.routeAudited(mPut, pAISettingsWrite, "/api/v2/ai/providers/", "update", "ai_provider", aiProviderH.ProviderHandler)
	r.routeAudited(mPatch, pAISettingsWrite, "/api/v2/ai/providers/", "update", "ai_provider", aiProviderH.ProviderHandler)
	r.routeAudited(mDelete, pAISettingsWrite, "/api/v2/ai/providers/", "delete", "ai_provider", aiProviderH.ProviderHandler)
	r.routeAudited(mPut, pAISettingsWrite, "/api/v2/ai/settings", "update", "ai_settings", aiProviderH.SettingsHandler)
	r.routeAudited(mPatch, pAISettingsWrite, "/api/v2/ai/settings", "update", "ai_settings", aiProviderH.SettingsHandler)
	r.routeAudited(mPut, pAISettingsWrite, "/api/v2/ai/budgets/", "update", "ai_budget", aiProviderH.BudgetHandler)

	// GitHub 集成与部署
	if r.ghClient != nil {
		// 走 service 层，不再直连 database（CLAUDE.md 层级约束）
		githubH := settingsHandler.NewGitHubHandler(r.ghClient, r.service, r.service)
		deployH := adminHandler.NewDeployHandler(r.ghClient, r.service, r.service)
		if r.deployer != nil {
			deployH.SetDeployer(r.deployer)
		}

		// GitHub OAuth 回调：由 GitHub 回跳，无会话；code 由 GitHub 签发、一次性使用
		r.openAudited(mPost, "/api/github/callback", "create", "github", githubH.Callback)

		r.route(mGet, pGitHubStatusRead, "/api/github/connection", githubH.Connection)
		r.route(mGet, pDeployHistoryRead, "/api/deploy/history", deployH.History)

		r.routeAudited(mPost, pGitHubManage, "/api/github/connect", "create", "github", githubH.Connect)
		r.routeAudited(mDelete, pGitHubManage, "/api/github/disconnect", "delete", "github", githubH.Disconnect)
		r.route(mGet, pGitHubManage, "/api/github/repos", githubH.Repos)
		r.route(mGet, pGitHubManage, "/api/github/repos/", githubH.RepoSubRoute)

		r.route(mGet, pDeployManage, "/api/deploy/config", deployH.Config)
		r.routeAudited(mPut, pDeployManage, "/api/deploy/config", "update", "deploy_config", deployH.Config)
		r.route(mGet, pDeployManage, "/api/deploy/kustomize-paths", deployH.KustomizePaths)
		r.route(mGet, pDeployManage, "/api/deploy/status", deployH.Status)
		r.routeAudited(mPost, pDeployManage, "/api/deploy/test-connection", "execute", "deploy", deployH.TestConnection)
		r.routeAudited(mPost, pDeployManage, "/api/deploy/sync", "execute", "deploy", deployH.SyncNow)
		r.routeAudited(mPost, pDeployManage, "/api/deploy/rollback", "execute", "deploy", deployH.Rollback)
	}
}

// ================================================================
// 路由注册辅助函数
// 审计中间件包装在权限检查之外：无论成功或权限不足，都会记录
// ================================================================

// open 注册无需主体的路由
func (r *Router) open(method, path string, h http.HandlerFunc) {
	r.handle(routeSpec{method: method, path: path}, h)
}

// openAudited 注册无需主体、需要审计的路由（登录、第三方回跳）
func (r *Router) openAudited(method, path, action, resource string, h http.HandlerFunc) {
	r.handle(routeSpec{method: method, path: path, audited: true}, r.audit(action, resource)(h))
}

// route 注册需要权限的路由
func (r *Router) route(method string, p perm, path string, h http.HandlerFunc) {
	r.handle(routeSpec{method: method, path: path, perm: p}, middleware.RequirePermission(p, h))
}

// routeAudited 注册需要权限且审计的路由
// 顺序: Audit -> RequirePermission -> Handler
func (r *Router) routeAudited(method string, p perm, path, action, resource string, h http.HandlerFunc) {
	r.handle(routeSpec{method: method, path: path, perm: p, audited: true}, r.audit(action, resource)(middleware.RequirePermission(p, h)))
}

// handle 以「方法 路径」模式注册（Go 1.22 起的 ServeMux 语法），并登记路由声明
func (r *Router) handle(spec routeSpec, h http.HandlerFunc) {
	r.specs = append(r.specs, spec)
	r.mux.HandleFunc(spec.method+" "+spec.path, h)
}

// audit 创建审计中间件
func (r *Router) audit(action, resource string) func(http.HandlerFunc) http.HandlerFunc {
	return middleware.Audit(r.database.Audit, middleware.AuditConfig{
		Action:   action,
		Resource: resource,
	})
}

// healthCheck 健康检查
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

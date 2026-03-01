// atlhyper_master_v2/gateway/routes.go
// API 路由统一注册
// 所有 API 在此集中管理，便于查看和维护
//
// 认证策略（开源项目，展示优先）：
//   - Public: 所有只读查询，无需登录
//   - Operator (2): 敏感信息查看、指令下发
//   - Admin (3): 用户管理、系统配置
//   - Viewer (1): 等同于游客，无额外权限
package gateway

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service"
)

// Router 路由管理器
type Router struct {
	mux       *http.ServeMux
	publicMux *http.ServeMux // 公开路由（不需要认证）
	service   service.Service
	database  *database.DB
	bus       mq.Producer
	aiService ai.AIService
}

// NewRouter 创建路由管理器
func NewRouter(svc service.Service, db *database.DB, bus mq.Producer, aiSvc ai.AIService) *Router {
	return &Router{
		mux:       http.NewServeMux(),
		publicMux: http.NewServeMux(),
		service:   svc,
		database:  db,
		bus:       bus,
		aiService: aiSvc,
	}
}

// Handler 返回包装了中间件的 Handler
func (r *Router) Handler() http.Handler {
	r.registerRoutes()

	// 组合路由器：先检查公开路由，再检查需要认证的路由
	return middleware.Logging(middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// 尝试公开路由
		if h, pattern := r.publicMux.Handler(req); pattern != "" {
			h.ServeHTTP(w, req)
			return
		}
		// 需要认证的路由
		middleware.AuthRequired(r.mux).ServeHTTP(w, req)
	})))
}

// registerRoutes 注册所有路由
func (r *Router) registerRoutes() {
	// 创建 Handlers
	userHandler := handler.NewUserHandler(r.database.User)
	clusterHandler := handler.NewClusterHandler(r.service)
	overviewHandler := handler.NewOverviewHandler(r.service)
	sloHandler := handler.NewSLOHandler(r.service, r.database.SLO)
	sloMeshHandler := handler.NewSLOMeshHandler(r.service)
	podHandler := handler.NewPodHandler(r.service)
	nodeHandler := handler.NewNodeHandler(r.service)
	deploymentHandler := handler.NewDeploymentHandler(r.service)
	daemonsetHandler := handler.NewDaemonSetHandler(r.service)
	statefulsetHandler := handler.NewStatefulSetHandler(r.service)
	serviceHandler := handler.NewServiceHandler(r.service)
	ingressHandler := handler.NewIngressHandler(r.service)
	configmapHandler := handler.NewConfigMapHandler(r.service)
	secretHandler := handler.NewSecretHandler(r.service)
	namespaceHandler := handler.NewNamespaceHandler(r.service)
	eventHandler := handler.NewEventHandler(r.service, r.database)
	commandHandler := handler.NewCommandHandler(r.service, r.database)
	notifyHandler := handler.NewNotifyHandler(r.database)
	settingsHandler := handler.NewSettingsHandler(r.database)
	aiProviderHandler := handler.NewAIProviderHandler(r.database)
	opsHandler := handler.NewOpsHandler(r.service, r.bus)
	auditHandler := handler.NewAuditHandler(r.database)
	jobHandler := handler.NewJobHandler(r.service)
	cronjobHandler := handler.NewCronJobHandler(r.service)
	pvHandler := handler.NewPVHandler(r.service)
	pvcHandler := handler.NewPVCHandler(r.service)
	networkPolicyHandler := handler.NewNetworkPolicyHandler(r.service)
	resourceQuotaHandler := handler.NewResourceQuotaHandler(r.service)
	limitRangeHandler := handler.NewLimitRangeHandler(r.service)
	serviceAccountHandler := handler.NewServiceAccountHandler(r.service)
	nodeMetricsHandler := handler.NewNodeMetricsHandler(r.service, r.service, r.bus)
	observeHandler := handler.NewObserveHandler(r.service, r.service, r.bus)
	aiopsGraphHandler := handler.NewAIOpsGraphHandler(r.service)
	aiopsBaselineHandler := handler.NewAIOpsBaselineHandler(r.service)
	aiopsRiskHandler := handler.NewAIOpsRiskHandler(r.service)
	aiopsIncidentHandler := handler.NewAIOpsIncidentHandler(r.service)
	aiopsAIHandler := handler.NewAIOpsAIHandler(r.service)

	// ================================================================
	// 公开路由（无需认证）
	// 开源项目：所有只读查询对外开放
	// ================================================================

	// 登录需要审计（记录成功/失败的登录尝试）
	r.publicAudited("/api/v2/user/login", "login", "user", userHandler.Login)

	r.public(func(register func(pattern string, h http.HandlerFunc)) {
		// 健康检查
		register("/health", healthCheck)

		// ---------- 集群概览 ----------
		register("/api/v2/overview", overviewHandler.Get)

		// ---------- 集群查询 ----------
		register("/api/v2/clusters", clusterHandler.List)
		register("/api/v2/clusters/", clusterHandler.Get)

		// ---------- 工作负载查询 ----------
		// Pod
		register("/api/v2/pods", podHandler.List)
		register("/api/v2/pods/", podHandler.Get)

		// Node
		register("/api/v2/nodes", nodeHandler.List)
		register("/api/v2/nodes/", nodeHandler.Get)

		// Deployment
		register("/api/v2/deployments", deploymentHandler.List)
		register("/api/v2/deployments/", deploymentHandler.Get)

		// DaemonSet
		register("/api/v2/daemonsets", daemonsetHandler.List)
		register("/api/v2/daemonsets/", daemonsetHandler.Get)

		// StatefulSet
		register("/api/v2/statefulsets", statefulsetHandler.List)
		register("/api/v2/statefulsets/", statefulsetHandler.Get)

		// ---------- 网络查询 ----------
		// Service
		register("/api/v2/services", serviceHandler.List)
		register("/api/v2/services/", serviceHandler.Get)

		// Ingress
		register("/api/v2/ingresses", ingressHandler.List)
		register("/api/v2/ingresses/", ingressHandler.Get)

		// ---------- 批处理工作负载查询 ----------
		// Job
		register("/api/v2/jobs", jobHandler.List)
		register("/api/v2/jobs/", jobHandler.Get)

		// CronJob
		register("/api/v2/cronjobs", cronjobHandler.List)
		register("/api/v2/cronjobs/", cronjobHandler.Get)

		// ---------- 存储查询 ----------
		// PersistentVolume
		register("/api/v2/pvs", pvHandler.List)
		register("/api/v2/pvs/", pvHandler.Get)

		// PersistentVolumeClaim
		register("/api/v2/pvcs", pvcHandler.List)
		register("/api/v2/pvcs/", pvcHandler.Get)

		// ---------- 策略与配额查询 ----------
		// NetworkPolicy
		register("/api/v2/network-policies", networkPolicyHandler.List)
		register("/api/v2/network-policies/", networkPolicyHandler.Get)

		// ResourceQuota
		register("/api/v2/resource-quotas", resourceQuotaHandler.List)
		register("/api/v2/resource-quotas/", resourceQuotaHandler.Get)

		// LimitRange
		register("/api/v2/limit-ranges", limitRangeHandler.List)
		register("/api/v2/limit-ranges/", limitRangeHandler.Get)

		// ServiceAccount
		register("/api/v2/service-accounts", serviceAccountHandler.List)
		register("/api/v2/service-accounts/", serviceAccountHandler.Get)

		// ---------- 配置查询（仅列表，详情需要权限） ----------
		register("/api/v2/configmaps", configmapHandler.List)

		// ---------- 命名空间查询 ----------
		register("/api/v2/namespaces", namespaceHandler.List)
		register("/api/v2/namespaces/", namespaceHandler.Get)

		// ---------- 事件查询 ----------
		register("/api/v2/events", eventHandler.List)
		register("/api/v2/events/by-resource", eventHandler.ListByResource)

		// ---------- 指令查询 ----------
		register("/api/v2/commands/history", commandHandler.ListHistory)
		register("/api/v2/commands/", commandHandler.GetStatus)

		// ---------- SLO 监控查询（只读） ----------
		register("/api/v2/slo/domains", sloHandler.Domains)       // V1: 按 service key
		register("/api/v2/slo/domains/v2", sloHandler.DomainsV2)  // V2: 按真实域名
		register("/api/v2/slo/domains/detail", sloHandler.DomainDetail)
		register("/api/v2/slo/domains/history", sloHandler.DomainHistory)
		register("/api/v2/slo/domains/latency", sloHandler.LatencyDistribution)
		register("/api/v2/slo/targets", sloHandler.Targets)
		register("/api/v2/slo/status-history", sloHandler.StatusHistory)

		// ---------- SLO 服务网格查询（只读） ----------
		register("/api/v2/slo/mesh/topology", sloMeshHandler.MeshTopology)
		register("/api/v2/slo/mesh/service/detail", sloMeshHandler.ServiceDetail)

		// ---------- 节点指标查询（只读） ----------
		register("/api/v2/node-metrics", nodeMetricsHandler.Route)
		register("/api/v2/node-metrics/", nodeMetricsHandler.Route)

		// ---------- 可观测性查询（ClickHouse 按需） ----------
		register("/api/v2/observe/metrics/summary", observeHandler.MetricsSummary)
		register("/api/v2/observe/metrics/nodes", observeHandler.MetricsNodes)
		register("/api/v2/observe/metrics/nodes/", observeHandler.MetricsNodeRoute)
		register("/api/v2/observe/logs/summary", observeHandler.LogsSummary)
		register("/api/v2/observe/logs/query", observeHandler.LogsQuery)
		register("/api/v2/observe/logs/histogram", observeHandler.LogsHistogram)
		register("/api/v2/observe/traces/services", observeHandler.TracesServices)
		register("/api/v2/observe/traces/services/", observeHandler.APMServiceSeries)
		register("/api/v2/observe/traces/stats", observeHandler.TracesStats)
		register("/api/v2/observe/traces/topology", observeHandler.TracesTopology)
		register("/api/v2/observe/traces/operations", observeHandler.TracesOperations)
		register("/api/v2/observe/traces", observeHandler.TracesList)
		register("/api/v2/observe/traces/", observeHandler.TracesDetail)
		register("/api/v2/observe/slo/summary", observeHandler.SLOSummary)
		register("/api/v2/observe/slo/ingress", observeHandler.SLOIngress)
		register("/api/v2/observe/slo/services", observeHandler.SLOServices)
		register("/api/v2/observe/slo/edges", observeHandler.SLOEdges)
		register("/api/v2/observe/slo/timeseries", observeHandler.SLOTimeSeries)

		// ---------- AIOps 查询（只读） ----------
		register("/api/v2/aiops/graph", aiopsGraphHandler.Graph)
		register("/api/v2/aiops/graph/trace", aiopsGraphHandler.Trace)
		register("/api/v2/aiops/baseline", aiopsBaselineHandler.Baseline)
		register("/api/v2/aiops/risk/cluster", aiopsRiskHandler.ClusterRisk)
		register("/api/v2/aiops/risk/entities", aiopsRiskHandler.EntityRisks)
		register("/api/v2/aiops/risk/entity", aiopsRiskHandler.EntityRisk)
		register("/api/v2/aiops/incidents", aiopsIncidentHandler.List)
		register("/api/v2/aiops/incidents/stats", aiopsIncidentHandler.Stats)
		register("/api/v2/aiops/incidents/patterns", aiopsIncidentHandler.Patterns)
		register("/api/v2/aiops/incidents/", aiopsIncidentHandler.Detail)
	})

	// ================================================================
	// Operator 权限（Role >= 2）
	// 敏感信息查看、操作执行
	// 所有敏感操作都需要审计（包括权限不足的失败尝试）
	// ================================================================

	// AIOps AI 增强 (Operator 权限，有 LLM API 调用成本)
	r.operator(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/aiops/ai/summarize", aiopsAIHandler.Summarize)
		register("/api/v2/aiops/ai/recommend", aiopsAIHandler.Recommend)
	})

	// ConfigMap 详情、通知渠道、审计日志、AI 配置查询（不审计，只是查看）
	r.operator(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/configmaps/", configmapHandler.Get)
		register("/api/v2/secrets", secretHandler.List)
		register("/api/v2/notify/channels", notifyHandler.ListChannels)
		register("/api/v2/audit/logs", auditHandler.List)
		register("/api/v2/settings/ai", settingsHandler.AIConfigHandler)
		register("/api/v2/ai/providers", aiProviderHandler.ProvidersHandler)
		register("/api/v2/ai/active", aiProviderHandler.ActiveConfigHandler)
	})

	// ---------- 需要审计的敏感操作 ----------
	// 指令下发
	r.operatorAudited("/api/v2/commands", "execute", "command", commandHandler.Create)

	// Pod 操作
	r.operatorAudited("/api/v2/ops/pods/logs", "read", "pod", opsHandler.PodLogs)
	r.operatorAudited("/api/v2/ops/pods/restart", "execute", "pod", opsHandler.PodRestart)

	// Deployment 操作
	r.operatorAudited("/api/v2/ops/deployments/scale", "execute", "deployment", opsHandler.DeploymentScale)
	r.operatorAudited("/api/v2/ops/deployments/restart", "execute", "deployment", opsHandler.DeploymentRestart)
	r.operatorAudited("/api/v2/ops/deployments/image", "execute", "deployment", opsHandler.DeploymentImage)

	// Node 操作
	r.operatorAudited("/api/v2/ops/nodes/cordon", "execute", "node", opsHandler.NodeCordon)
	r.operatorAudited("/api/v2/ops/nodes/uncordon", "execute", "node", opsHandler.NodeUncordon)

	// ConfigMap/Secret 数据获取（敏感数据读取需要审计）
	r.operatorAudited("/api/v2/ops/configmaps/data", "read", "configmap", opsHandler.ConfigMapData)
	r.operatorAudited("/api/v2/ops/secrets/data", "read", "secret", opsHandler.SecretData)

	// ================================================================
	// AI 对话（需要认证，Viewer+ 即可使用）
	// SSE 流式响应需要较长的 WriteTimeout（在 handler 内处理）
	// ================================================================

	if r.aiService != nil {
		aiHandler := handler.NewAIHandler(r.aiService)
		r.mux.HandleFunc("/api/v2/ai/conversations", aiHandler.Conversations)
		r.mux.HandleFunc("/api/v2/ai/conversations/", aiHandler.ConversationByID)
		r.mux.HandleFunc("/api/v2/ai/chat", aiHandler.Chat)
	}

	// ================================================================
	// Admin 权限（Role >= 3）
	// 用户管理、系统配置
	// 所有管理操作都需要审计
	// ================================================================

	// 用户列表查询（不审计，只是查看）
	r.admin(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/user/list", userHandler.List)
	})

	// ---------- 需要审计的管理操作 ----------
	// 用户管理
	r.adminAudited("/api/v2/user/register", "create", "user", userHandler.Register)
	r.adminAudited("/api/v2/user/update-role", "update", "user", userHandler.UpdateRole)
	r.adminAudited("/api/v2/user/update-status", "update", "user", userHandler.UpdateStatus)
	r.adminAudited("/api/v2/user/delete", "delete", "user", userHandler.Delete)

	// 通知渠道管理（Operator 可管理）
	r.operatorAudited("/api/v2/notify/channels/", "update", "notify", notifyHandler.ChannelHandler)

	// AI 配置管理（需要 Admin 权限）
	r.adminAudited("/api/v2/settings/ai/", "update", "ai_config", settingsHandler.AIConfigHandler)

	// AI Provider 管理（需要 Admin 权限）
	r.adminAudited("/api/v2/ai/providers/", "update", "ai_provider", aiProviderHandler.ProviderHandler)
	r.adminAudited("/api/v2/ai/active/", "update", "ai_provider", aiProviderHandler.ActiveConfigHandler)
}

// ================================================================
// 路由注册辅助函数
// ================================================================

// public 注册公开路由（无需认证）
func (r *Router) public(fn func(register func(pattern string, h http.HandlerFunc))) {
	fn(func(pattern string, h http.HandlerFunc) {
		r.publicMux.HandleFunc(pattern, h)
	})
}

// operator 注册 Operator 权限路由
func (r *Router) operator(fn func(register func(pattern string, h http.HandlerFunc))) {
	fn(func(pattern string, h http.HandlerFunc) {
		r.mux.HandleFunc(pattern, middleware.RequireMinRole(middleware.RoleOperator, h))
	})
}

// admin 注册 Admin 权限路由
func (r *Router) admin(fn func(register func(pattern string, h http.HandlerFunc))) {
	fn(func(pattern string, h http.HandlerFunc) {
		r.mux.HandleFunc(pattern, middleware.RequireMinRole(middleware.RoleAdmin, h))
	})
}

// ================================================================
// 审计路由辅助函数
// 审计中间件包装在权限检查之外，无论成功失败都会记录
// ================================================================

// audit 创建审计中间件
func (r *Router) audit(action, resource string) func(http.HandlerFunc) http.HandlerFunc {
	return middleware.Audit(r.database.Audit, middleware.AuditConfig{
		Action:   action,
		Resource: resource,
	})
}

// operatorAudited 注册带审计的 Operator 权限路由
// 顺序: Audit -> RequireMinRole(Operator) -> Handler
// 无论认证成功/失败，都会记录审计日志
func (r *Router) operatorAudited(pattern, action, resource string, h http.HandlerFunc) {
	handler := r.audit(action, resource)(middleware.RequireMinRole(middleware.RoleOperator, h))
	r.mux.HandleFunc(pattern, handler)
}

// adminAudited 注册带审计的 Admin 权限路由
// 顺序: Audit -> RequireMinRole(Admin) -> Handler
func (r *Router) adminAudited(pattern, action, resource string, h http.HandlerFunc) {
	handler := r.audit(action, resource)(middleware.RequireMinRole(middleware.RoleAdmin, h))
	r.mux.HandleFunc(pattern, handler)
}

// publicAudited 注册带审计的公开路由（如登录）
func (r *Router) publicAudited(pattern, action, resource string, h http.HandlerFunc) {
	handler := r.audit(action, resource)(h)
	r.publicMux.HandleFunc(pattern, handler)
}

// healthCheck 健康检查
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

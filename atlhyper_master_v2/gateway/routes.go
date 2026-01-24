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
}

// NewRouter 创建路由管理器
func NewRouter(svc service.Service, db *database.DB, bus mq.Producer) *Router {
	return &Router{
		mux:       http.NewServeMux(),
		publicMux: http.NewServeMux(),
		service:   svc,
		database:  db,
		bus:       bus,
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
	commandHandler := handler.NewCommandHandler(r.service)
	notifyHandler := handler.NewNotifyHandler(r.database)
	opsHandler := handler.NewOpsHandler(r.service, r.bus)
	auditHandler := handler.NewAuditHandler(r.database)

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

		// ---------- 配置查询（仅列表，详情需要权限） ----------
		register("/api/v2/configmaps", configmapHandler.List)

		// ---------- 命名空间查询 ----------
		register("/api/v2/namespaces", namespaceHandler.List)
		register("/api/v2/namespaces/", namespaceHandler.Get)

		// ---------- 事件查询 ----------
		register("/api/v2/events", eventHandler.List)
		register("/api/v2/events/by-resource", eventHandler.ListByResource)

		// ---------- 指令状态查询 ----------
		register("/api/v2/commands/", commandHandler.GetStatus)

		// ---------- 通知渠道查询 ----------
		register("/api/v2/notify/channels", notifyHandler.ListChannels)

		// ---------- 审计日志查询 ----------
		register("/api/v2/audit/logs", auditHandler.List)
	})

	// ================================================================
	// Operator 权限（Role >= 2）
	// 敏感信息查看、操作执行
	// 所有敏感操作都需要审计（包括权限不足的失败尝试）
	// ================================================================

	// ConfigMap 详情（不审计，只是查看）
	r.operator(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/configmaps/", configmapHandler.Get)
		register("/api/v2/secrets", secretHandler.List)
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

	// 通知渠道管理
	r.adminAudited("/api/v2/notify/channels/", "update", "notify", notifyHandler.ChannelHandler)
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

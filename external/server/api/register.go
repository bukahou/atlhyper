package uiapi

import (
	"NeuroController/external/server/api/auth"
	"NeuroController/external/server/api/cluster"
	"NeuroController/external/server/api/configmap"
	"NeuroController/external/server/api/deployment"
	"NeuroController/external/server/api/event"
	"NeuroController/external/server/api/ingress"
	"NeuroController/external/server/api/metrics"
	"NeuroController/external/server/api/namespace"
	"NeuroController/external/server/api/node"
	"NeuroController/external/server/api/pod"
	"NeuroController/external/server/api/podlist"
	"NeuroController/external/server/api/service"

	"github.com/gin-gonic/gin"
)

func RegisterUIAPIRoutes(router *gin.RouterGroup) {
	// ✅ 注册登录接口（不需要任何认证）
	router.POST("/auth/login", auth.HandleLogin)

	// =============================
	// 📖 基础只读接口（角色 ≥ 1）
	// =============================
	// read := router.Group("")
	// read.Use(auth.RequireMinRole(auth.RoleViewer))
	read := router.Group("")
	read.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleViewer))

	//新接口/获取agent推送上来的集群信息
	read.GET("/podlist/latest", podlist.HandleGetLatestPodList)

	//主页使用，overview
	read.GET("/cluster/overview", cluster.ClusterOverviewHandler)
	read.GET("/namespace/list", namespace.ListAllNamespacesHandler)
	read.GET("/event/list/recent", event.GetRecentLogEventsHandler)
	read.GET("/ingress/list/all", ingress.GetAllIngressesHandler)
	read.GET("/service/list/all", service.GetAllServicesHandler)
	read.GET("/metrics/latest", metrics.GetInMemoryLatestHandler)
	read.GET("/auth/user/list", auth.HandleListAllUsers)
	read.GET("/auth/userauditlogs/list", auth.HandleGetUserAuditLogs)
	deployment.RegisterDeploymentRoutes(read.Group("/deployment"))
	pod.RegisterPodRoutes(read.Group("/pod"))
	node.RegisterNodeRoutes(read.Group("/node"))
	configmap.RegisterConfigMapRoutes(read.Group("/configmap"))

	// =============================
	// 🔒 操作类接口（角色 ≥ 2）
	// =============================
	// ops := router.Group("")
	// ops.Use(auth.RequireMinRole(auth.RoleOperator))

	ops := router.Group("")
	ops.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleOperator))

	// pod.RegisterPodOpsRoutes(ops.Group("/pod-ops"))
	// deployment.RegisterDeploymentOpsRoutes(ops.Group("/deployment-ops"))
	ops.POST("/pod-ops/restart/:ns/:name", pod.RestartPodHandler)
	ops.POST("/deployment-ops/scale", deployment.ScaleDeploymentHandler)
	ops.POST("/node-ops/schedule", node.ToggleNodeSchedulableHandler)
	// pod.RegisterPodOpsRoutes(ops.Group("/pod-ops"))
	// deployment.RegisterDeploymentOpsRoutes(ops.Group("/deployment-ops"))
	// ops.GET("/auth/user/list", auth.HandleListAllUsers)
	// ops.GET("/auth/userauditlogs/list", auth.HandleGetUserAuditLogs)


	// =============================
	// 🔐 管理员权限接口（角色 == 3）
	// =============================
	admin := router.Group("")
	admin.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleAdmin))

	admin.POST("/auth/user/register", auth.HandleRegisterUser)
	admin.POST("/auth/user/update-role", auth.HandleUpdateUserRole)
	//获取全部用户信息接口
	// admin.GET("/auth/user/list", auth.HandleListAllUsers)
	//针对node的操作。因此需要在在组组最高权限
	// admin.POST("/node-ops/schedule", node.ToggleNodeSchedulableHandler)
	// 获取用户审计日志
	// admin.GET("/auth/userauditlogs/list", auth.HandleGetUserAuditLogs)

}

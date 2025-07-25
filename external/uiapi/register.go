package uiapi

import (
	"NeuroController/external/uiapi/auth"
	"NeuroController/external/uiapi/cluster"
	"NeuroController/external/uiapi/configmap"
	"NeuroController/external/uiapi/deployment"
	"NeuroController/external/uiapi/event"
	"NeuroController/external/uiapi/ingress"
	"NeuroController/external/uiapi/namespace"
	"NeuroController/external/uiapi/node"
	"NeuroController/external/uiapi/pod"
	"NeuroController/external/uiapi/service"

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

	cluster.RegisterClusterRoutes(read.Group("/cluster"))
	deployment.RegisterDeploymentRoutes(read.Group("/deployment"))
	pod.RegisterPodRoutes(read.Group("/pod"))
	node.RegisterNodeRoutes(read.Group("/node"))
	namespace.RegisterNamespaceRoutes(read.Group("/namespace"))
	event.RegisterEventRoutes(read.Group("/event"))
	ingress.RegisterIngressRoutes(read.Group("/ingress"))
	service.RegisterServiceRoutes(read.Group("/service"))
	configmap.RegisterConfigMapRoutes(read.Group("/configmap"))

	// =============================
	// 🔒 操作类接口（角色 ≥ 2）
	// =============================
	// ops := router.Group("")
	// ops.Use(auth.RequireMinRole(auth.RoleOperator))

	ops := router.Group("")
	ops.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleOperator))

	pod.RegisterPodOpsRoutes(ops.Group("/pod-ops"))
	deployment.RegisterDeploymentOpsRoutes(ops.Group("/deployment-ops"))

	// =============================
	// 🔐 管理员权限接口（角色 == 3）
	// =============================
	admin := router.Group("")
	admin.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleAdmin))

	// 用户注册接口
	admin.POST("/auth/user/register", auth.HandleRegisterUser)
	// 用户权限更新接口
	admin.POST("/auth/user/update-role", auth.HandleUpdateUserRole)
	//获取全部用户信息接口
	admin.GET("/auth/user/list", auth.HandleListAllUsers)
	//针对node的操作。因此需要在在组组最高权限
	admin.POST("/node-ops/schedule", node.ToggleNodeSchedulableHandler)
	// 获取用户审计日志
	admin.GET("/auth/userauditlogs/list", auth.HandleGetUserAuditLogs)

}

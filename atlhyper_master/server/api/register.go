package uiapi

import (
	"AtlHyper/atlhyper_master/client/alert"
	"AtlHyper/atlhyper_master/control"
	"AtlHyper/atlhyper_master/server/api/auth"
	"AtlHyper/atlhyper_master/server/api/web_api"

	"github.com/gin-gonic/gin"
)

func RegisterUIAPIRoutes(router *gin.RouterGroup) {
	// =============================
	// 🌐 公开接口（无需认证）
	// =============================
	// 说明：这些接口供网站整体使用，任何人都可以访问
	// 包括：登录、数据查看（overview/list/detail）等

	// 登录接口
	router.POST("/auth/login", auth.HandleLogin)
	router.GET("/alert/slack/preview", alert.HandleAlertSlackPreview)

	// 用户相关接口（查看）
	router.GET("/auth/user/list", auth.HandleListAllUsers)
	router.GET("/auth/userauditlogs/list", auth.HandleGetUserAuditLogs)
	router.POST("/user/todos/by-username", web_api.GetUserTodosHandler)
	router.GET("/user/todos/all", web_api.GetAllTodosHandler)

	// 集群数据查看接口（只读）
	router.POST("/cluster/overview", web_api.GetOverviewHandler)
	router.POST("/pod/overview", web_api.GetPodOverviewHandler)
	router.POST("/pod/detail", web_api.GetPodDetailHandler)
	router.POST("/node/overview", web_api.GetNodeOverviewHandler)
	router.POST("/node/detail", web_api.GetNodeDetailHandler)
	router.POST("/service/overview", web_api.GetServiceOverviewHandler)
	router.POST("/service/detail", web_api.GetServiceDetailHandler)
	router.POST("/namespace/overview", web_api.GetNamespaceOverviewHandler)
	router.POST("/namespace/detail", web_api.GetNamespaceDetailHandler)
	router.POST("/ingress/overview", web_api.GetIngressOverviewHandler)
	router.POST("/ingress/detail", web_api.GetIngressDetailHandler)
	router.POST("/deployment/overview", web_api.GetDeploymentOverviewHandler)
	router.POST("/deployment/detail", web_api.GetDeploymentDetailHandler)
	router.POST("/configmap/detail", web_api.GetConfigMapDetailHandler)
	router.POST("/event/logs", web_api.GetEventLogsSinceHandler)
	router.POST("/metrics/overview", web_api.GetMetricsOverviewHandler)
	router.POST("/metrics/node/detail", web_api.GetMetricsNodeDetailHandler)
	router.POST("/config/slack/get", web_api.GetSlackConfig)

	// =============================
	// 🔒 需要登录的接口（资源操作）
	// =============================
	// 说明：这些接口需要登录后才能使用
	// 包括：Pod/Node/Deployment 操作、配置修改等

	ops := router.Group("")
	ops.Use(auth.AuthMiddleware())
	{
		// Todo 操作（需要登录）
		ops.POST("/user/todo/create", web_api.CreateTodoHandler)
		ops.POST("/user/todo/update", web_api.UpdateTodoHandler)
		ops.POST("/user/todo/delete", web_api.SoftDeleteTodoHandler)

		// Pod 操作
		ops.POST("/ops/pod/logs", control.HandleWebGetPodLogs)
		ops.POST("/ops/pod/restart", control.HandleWebRestartPod)

		// Node 操作
		ops.POST("/ops/node/cordon", control.HandleWebCordonNode)
		ops.POST("/ops/node/uncordon", control.HandleWebUncordonNode)

		// Workload 操作
		ops.POST("/ops/workload/updateImage", control.HandleWebUpdateImage)
		ops.POST("/ops/workload/scale", control.HandleWebScaleWorkload)
	}

	// =============================
	// 🔐 管理员接口（需要 Admin 权限）
	// =============================
	admin := router.Group("")
	admin.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleAdmin))
	{
		admin.POST("/auth/user/register", auth.HandleRegisterUser)
		admin.POST("/auth/user/update-role", auth.HandleUpdateUserRole)
		admin.POST("/auth/user/delete", auth.HandleDeleteUser)
		admin.POST("/config/slack/update", web_api.UpdateSlackConfig)
	}
}

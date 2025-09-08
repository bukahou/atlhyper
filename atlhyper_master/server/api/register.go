package uiapi

import (
	"AtlHyper/atlhyper_master/client/alert"
	"AtlHyper/atlhyper_master/control"
	"AtlHyper/atlhyper_master/server/api/auth"
	"AtlHyper/atlhyper_master/server/api/testapi"
	"AtlHyper/atlhyper_master/server/api/web_api"

	"github.com/gin-gonic/gin"
)

func RegisterUIAPIRoutes(router *gin.RouterGroup) {
	// ✅ 注册登录接口（不需要任何认证）
	router.POST("/auth/login", auth.HandleLogin)
	router.GET("/alert/slack/preview", alert.HandleAlertSlackPreview)

	// =============================
	// 📖 基础只读接口（角色 ≥ 1）
	// =============================
	read := router.Group("")
	read.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleViewer))
	//新接口/获取agent推送上来的集群信息
	read.GET("/auth/user/list", auth.HandleListAllUsers)
	read.GET("/auth/userauditlogs/list", auth.HandleGetUserAuditLogs)
	//用户相关接口
	read.POST("/user/todos", web_api.GetUserTodosHandler)
	read.POST("/user/todos", web_api.GetAllTodosHandler)
	read.POST("/user/todo/create", web_api.CreateTodoHandler)
	read.POST("/user/todo/update", web_api.UpdateTodoHandler)
	//集群相关接口
	read.POST("/cluster/overview", web_api.GetOverviewHandler)
	read.POST("/pod/overview", web_api.GetPodOverviewHandler)
	read.POST("/pod/detail", web_api.GetPodDetailHandler)
	read.POST("/node/overview", web_api.GetNodeOverviewHandler)
	read.POST("/node/detail", web_api.GetNodeDetailHandler)
	read.POST("/service/overview", web_api.GetServiceOverviewHandler)
	read.POST("/service/detail", web_api.GetServiceDetailHandler)
	read.POST("/namespace/overview", web_api.GetNamespaceOverviewHandler)
	read.POST("/namespace/detail", web_api.GetNamespaceDetailHandler)
	read.POST("/ingress/overview", web_api.GetIngressOverviewHandler)
	read.POST("/ingress/detail", web_api.GetIngressDetailHandler)
	read.POST("/deployment/overview", web_api.GetDeploymentOverviewHandler)
	read.POST("/deployment/detail", web_api.GetDeploymentDetailHandler)
	read.POST("/configmap/detail", web_api.GetConfigMapDetailHandler)
	read.POST("/event/logs", web_api.GetEventLogsSinceHandler)
	read.POST("/metrics/overview", web_api.GetMetricsOverviewHandler)
	read.POST("/metrics/node/detail", web_api.GetMetricsNodeDetailHandler)
	read.POST("/config/slack/get", web_api.GetSlackConfig)

	testapi.RegisterRoutes(read)


	// =============================
	// 🔒 操作类接口（角色 ≥ 2）
	// =============================
	ops := router.Group("")
	ops.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleOperator))
	control.RegisterWebOpsRoutes(ops)

	
	// =============================
	// 🔐 管理员权限接口（角色 == 3）
	// =============================
	admin := router.Group("")
	admin.Use(auth.AuthMiddleware(), auth.RequireMinRole(auth.RoleAdmin))

	admin.POST("/auth/user/register", auth.HandleRegisterUser)
	admin.POST("/auth/user/update-role", auth.HandleUpdateUserRole)
	admin.POST("/config/slack/update", web_api.UpdateSlackConfig)

}

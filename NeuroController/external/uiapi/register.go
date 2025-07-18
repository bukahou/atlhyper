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

	admin.POST("/auth/user/register", auth.HandleRegisterUser)
	admin.POST("/auth/user/update-role", auth.HandleUpdateUserRole)
	admin.GET("/auth/user/list", auth.HandleListAllUsers)


}

// +---------------------------------------------+----------+-------------------------------+
// | Path                                        | Method   | Description                   |
// +=============================================+==========+===============================+
// | /uiapi/cluster/overview                     | GET      | 获取集群概要信息              |
// | /uiapi/deployment/list/all                  | GET      | 获取所有 Deployment           |
// | /uiapi/deployment/list/by-namespace/:ns     | GET      | 按命名空间获取 Deployment     |
// | /uiapi/deployment/get/:ns/:name             | GET      | 获取指定 Deployment           |
// | /uiapi/deployment/list/unavailable          | GET      | 获取不可用 Deployment         |
// | /uiapi/deployment/list/progressing          | GET      | 获取正在滚动更新的 Deployment |
// | /uiapi/event/list/all                       | GET      | 获取所有事件                  |
// | /uiapi/event/list/by-namespace/:ns          | GET      | 获取指定命名空间事件          |
// | /uiapi/event/list/by-object/:ns/:kind/:name | GET      | 获取指定对象关联事件          |
// | /uiapi/event/summary/type                   | GET      | 事件类型统计                  |
// | /uiapi/ingress/list/all                     | GET      | 获取所有 Ingress              |
// | /uiapi/ingress/list/by-namespace/:ns        | GET      | 按命名空间获取 Ingress        |
// | /uiapi/ingress/get/:ns/:name                | GET      | 获取指定 Ingress              |
// | /uiapi/ingress/list/ready                   | GET      | 获取已就绪 Ingress            |
// | /uiapi/namespace/list                       | GET      | 获取所有 Namespace            |
// | /uiapi/namespace/get/:name                  | GET      | 获取指定 Namespace            |
// | /uiapi/namespace/list/active                | GET      | 获取活跃状态 Namespace        |
// | /uiapi/namespace/list/terminating           | GET      | 获取终止中 Namespace          |
// | /uiapi/namespace/summary/status             | GET      | 命名空间状态统计              |
// | /uiapi/node/list                            | GET      | 获取所有 Node                 |
// | /uiapi/node/metrics                         | GET      | 获取 Node 资源使用情况        |
// | /uiapi/pod/list                             | GET      | 获取所有 Pod                  |
// | /uiapi/pod/list/:namespace                  | GET      | 按命名空间获取 Pod            |
// | /uiapi/pod/summary                          | GET      | 获取 Pod 状态摘要             |
// | /uiapi/pod/usage                            | GET      | 获取 Pod 资源使用量           |
// | /uiapi/service/list/all                     | GET      | 获取所有 Service              |
// | /uiapi/service/list/by-namespace/:ns        | GET      | 按命名空间获取 Service        |
// | /uiapi/service/get/:ns/:name                | GET      | 获取指定 Service              |
// | /uiapi/service/list/external                | GET      | 获取对外暴露 Service          |
// | /uiapi/service/list/headless                | GET      | 获取 Headless Service         |
// +---------------------------------------------+----------+-------------------------------+


// import (
// 	"NeuroController/external/auth"
// 	"NeuroController/external/uiapi/cluster"
// 	"NeuroController/external/uiapi/configmap"
// 	"NeuroController/external/uiapi/deployment"
// 	"NeuroController/external/uiapi/event"
// 	"NeuroController/external/uiapi/ingress"
// 	"NeuroController/external/uiapi/namespace"
// 	"NeuroController/external/uiapi/node"
// 	"NeuroController/external/uiapi/pod"
// 	"NeuroController/external/uiapi/service"

// 	"github.com/gin-gonic/gin"
// )

// RegisterUIAPIRoutes 注册所有 UI API 模块的路由入口
// func RegisterUIAPIRoutes(router *gin.RouterGroup) {
// 	// 各子模块直接挂载在 /uiapi/** 下
// 	cluster.RegisterClusterRoutes(router.Group("/cluster"))

// 	deployment.RegisterDeploymentRoutes(router.Group("/deployment"))

// 	pod.RegisterPodRoutes(router.Group("/pod"))

// 	node.RegisterNodeRoutes(router.Group("/node"))

// 	namespace.RegisterNamespaceRoutes(router.Group("/namespace"))

// 	event.RegisterEventRoutes(router.Group("/event"))

// 	ingress.RegisterIngressRoutes(router.Group("/ingress"))

// 	service.RegisterServiceRoutes(router.Group("/service"))

// 	configmap.RegisterConfigMapRoutes(router.Group("/configmap"))

// 	// ✅ 注册 Pod 操作类接口（
// 	pod.RegisterPodOpsRoutes(router.Group("/pod-ops"))
// 	deployment.RegisterDeploymentOpsRoutes(router.Group("/deployment-ops"))

// }
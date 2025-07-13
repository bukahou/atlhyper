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

package uiapi

import (
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

// RegisterUIAPIRoutes 注册所有 UI API 模块的路由入口
func RegisterUIAPIRoutes(router *gin.RouterGroup) {
	// 各子模块直接挂载在 /uiapi/** 下
	cluster.RegisterClusterRoutes(router.Group("/cluster"))

	deployment.RegisterDeploymentRoutes(router.Group("/deployment"))

	pod.RegisterPodRoutes(router.Group("/pod"))

	node.RegisterNodeRoutes(router.Group("/node"))

	namespace.RegisterNamespaceRoutes(router.Group("/namespace"))

	event.RegisterEventRoutes(router.Group("/event"))

	ingress.RegisterIngressRoutes(router.Group("/ingress"))

	service.RegisterServiceRoutes(router.Group("/service"))

	configmap.RegisterConfigMapRoutes(router.Group("/configmap"))

	// ✅ 后续添加模块也在这里统一注册
	// namespace.RegisterNamespaceRoutes(router.Group("/namespace"))
	// pod.RegisterPodRoutes(router.Group("/pod"))
}

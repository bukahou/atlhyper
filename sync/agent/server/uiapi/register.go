package uiapi

import "github.com/gin-gonic/gin"

// RegisterAlertSettingsRoutes 注册告警配置相关接口
func RegisterUIRoutes(rg *gin.RouterGroup) {
	rg.GET("/config/alert", HandleGetAlertConfig)
	rg.POST("/config/slack", HandleUpdateSlackConfig)
	rg.POST("/config/webhook", HandleUpdateWebhookConfig)
	rg.POST("/config/mail", HandleUpdateMailConfig)

	// 集群概览接口
	rg.GET("/cluster/overview", HandleClusterOverview)

	// ConfigMap 接口
	rg.GET("/configmaps/all", HandleAllConfigMaps)
	rg.GET("/configmaps/by-namespace/:ns", HandleConfigMapsByNamespace)
	rg.GET("/configmaps/detail/:ns/:name", HandleConfigMapDetail)

	// Deployment 接口
	rg.GET("/deployments/all", HandleAllDeployments)
	rg.GET("/deployments/by-namespace/:ns", HandleDeploymentsByNamespace)
	rg.GET("/deployments/detail/:ns/:name", HandleDeploymentDetail)
	rg.GET("/deployments/unavailable", HandleUnavailableDeployments)
	rg.GET("/deployments/progressing", HandleProgressingDeployments)
	rg.POST("/deployments/replicas", HandleUpdateDeploymentReplicas)
	rg.POST("/deployments/image", HandleUpdateDeploymentImage)

	// Event 事件接口
	rg.GET("/event/list/all", HandleGetAllEvents)
	rg.GET("/event/list/by-namespace/:ns", HandleGetEventsByNamespace)
	rg.GET("/event/list/by-object/:ns/:kind/:name", HandleGetEventsByObject)
	rg.GET("/event/stats/type-count", HandleGetEventTypeCounts)

	// Ingress 资源接口
	rg.GET("/ingress/list/all", HandleGetAllIngresses)
	rg.GET("/ingress/list/by-namespace/:ns", HandleGetIngressesByNamespace)
	rg.GET("/ingress/detail/:ns/:name", HandleGetIngressByName)
	rg.GET("/ingress/list/ready", HandleGetReadyIngresses)

	// Namespace 资源接口
	rg.GET("/namespace/list", HandleGetAllNamespaces)

	// Node 资源接口
	rg.GET("/node/overview", HandleGetNodeOverview)
	rg.GET("/node/list", HandleListAllNodes)
	rg.GET("/node/get/:name", HandleGetNodeDetail)
	rg.GET("/node/metrics-summary", HandleGetNodeMetricsSummary)
	rg.POST("/node/schedulable", HandleToggleNodeSchedulable)


	// Pod 资源接口
	rg.GET("/pod/list", HandleListAllPods)
	rg.GET("/pod/list/by-namespace/:ns", HandleListPodsByNamespace)
	rg.GET("/pod/summary", HandlePodStatusSummary)
	rg.GET("/pod/usage", HandlePodUsage)
	rg.GET("/pod/infos", HandlePodInfos)
	rg.GET("/pod/describe", HandlePodDescribe)
	rg.POST("/pod/restart", HandleRestartPod)
	rg.GET("/pod/logs", HandleGetPodLogs)


	// Service 资源接口
	rg.GET("/service/list", HandleListAllServices)
	rg.GET("/service/list/by-namespace/:ns", HandleListServicesByNamespace)
	rg.GET("/service/describe", HandleGetServiceByName)
	rg.GET("/service/list/external", HandleListExternalServices)
	rg.GET("/service/list/headless", HandleListHeadlessServices)


}

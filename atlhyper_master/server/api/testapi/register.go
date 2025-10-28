package testapi

import "github.com/gin-gonic/gin"

// RegisterRoutes 把 testapi 下的所有只读接口挂到二级前缀 /testapi 下
func RegisterRoutes(parent *gin.RouterGroup) {
	g := parent.Group("/testapi")

	// ============================
	// 🧩 现有测试接口
	// ============================
	g.GET("/metrics/latest", GetInMemoryLatestHandler)
	g.GET("/event/store/all", HandleGetStoreEvents)
	g.GET("/event/db/since", HandleGetDbEvents)
	g.GET("/podlist/latest", HandleGetLatestPodList)
	g.GET("/nodelist/latest", HandleGetLatestNodeList)
	g.GET("/servicelist/latest", HandleGetLatestServiceList)
	g.GET("/namespace/latest", HandleGetLatestNamespaceList)
	g.GET("/ingresslist/latest", HandleGetLatestIngressList)
	g.GET("/deploymentlist/latest", HandleGetLatestDeploymentList)
	g.GET("/configmaplist/latest", HandleGetLatestConfigMapList)

	// ============================
	// 📦 Snapshot 系列
	// ============================
	g.GET("/Snapshot/events", GetRecentEventsHandler)
	g.GET("/Snapshot/metrics", GetClusterMetricsRangeHandler)
	g.GET("/Snapshot/nodes", GetNodeListLatestHandler)
	g.GET("/Snapshot/pods", GetPodListLatestHandler)
	g.GET("/Snapshot/services", GetServiceListLatestHandler)
	g.GET("/Snapshot/namespaces", GetNamespaceListLatestHandler)
	g.GET("/Snapshot/ingresses", GetIngressListLatestHandler)
	g.GET("/Snapshot/deployments", GetDeploymentListLatestHandler)
	g.GET("/Snapshot/configmaps", GetConfigMapListLatestHandler)

	// （可选）最新一次全量指标快照
	g.GET("/Snapshot/metrics/latest", GetClusterMetricsLatestHandler)

	// ============================
	// 🚨 新增告警增量事件接口
	// ============================
	// 调用 alert.CollectNewEventLogsForAlert() 收集最新未重复的事件
	// 示例：GET /testapi/event/alert/new
	g.GET("/event/alert/new", HandleGetNewAlertEvents)
	g.GET("/ai/diagnose/run", HandleRunAIDiagnosis)
}

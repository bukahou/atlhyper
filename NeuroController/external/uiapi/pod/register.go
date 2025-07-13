// =======================================================================================
// 📄 register.go（external/uiapi/pod）
//
// ✨ 文件说明：
//     注册 Pod 相关的所有路由接口，包括：
//     - Pod 列表查询（全部 / 指定命名空间）
//     - Pod 状态汇总
//     - Pod 资源使用量
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package pod

import (
	"github.com/gin-gonic/gin"
)

// RegisterPodRoutes 将 Pod 模块相关的路由注册到指定分组
func RegisterPodRoutes(router *gin.RouterGroup) {
	router.GET("/list", ListAllPodsHandler)
	router.GET("/list/:ns", ListPodsByNamespaceHandler)
	router.GET("/summary", PodStatusSummaryHandler)
	router.GET("/usage", PodMetricsUsageHandler)
	router.GET("/list/brief", ListBriefPodsHandler)
	router.GET("/describe/:ns/:name", GetPodDescribeHandler)
	router.POST("/restart/:ns/:name", RestartPodHandler)
	// ✅ 获取 Pod 日志（支持 query: container & tail）
	router.GET("/logs/:ns/:name", GetPodLogsHandler)

}

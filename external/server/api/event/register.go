// =======================================================================================
// 📄 register.go（external/uiapi/event）
//
// ✨ 文件说明：
//     将 Event 模块的所有 REST API 路由挂载到 /uiapi/event 子路径下。
//     包含：列表、命名空间过滤、资源过滤、类型统计。
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package event

import "github.com/gin-gonic/gin"

// RegisterEventRoutes 挂载 /uiapi/event 路由组
func RegisterEventRoutes(router *gin.RouterGroup) {
	router.GET("/list/recent", GetRecentLogEventsHandler)

	//废弃
	// router.GET("/list/all", GetAllEventsHandler)
	// router.GET("/list/by-namespace/:ns", GetEventsByNamespaceHandler)
	// router.GET("/list/by-object/:ns/:kind/:name", GetEventsByInvolvedObjectHandler)
	// router.GET("/summary/type", GetEventTypeStatsHandler)
	
}

// =======================================================================================
// 📄 register.go（external/uiapi/node）
//
// ✨ 文件说明：
//     注册 Node 相关的 HTTP 路由入口，挂载于 /uiapi/node
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package node

import "github.com/gin-gonic/gin"

// RegisterNodeRoutes 注册 Node API 路由
func RegisterNodeRoutes(router *gin.RouterGroup) {
	router.GET("/overview", GetNodeOverviewHandler)
	router.GET("/get/:name", GetNodeDetailHandler)

	//废弃
	// router.GET("/list", GetAllNodesHandler)
	// router.GET("/metrics", GetNodeMetricsSummaryHandler)
	
	
}

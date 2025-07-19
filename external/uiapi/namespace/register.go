// =======================================================================================
// 📄 register.go（external/uiapi/namespace）
//
// ✨ 文件说明：
//     将命名空间相关的 handler 注册到 Gin 路由下，挂载路径为 /uiapi/namespace
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package namespace

import "github.com/gin-gonic/gin"

// RegisterNamespaceRoutes 注册 Namespace 相关路由
func RegisterNamespaceRoutes(rg *gin.RouterGroup) {
	rg.GET("/list", ListAllNamespacesHandler)
	// rg.GET("/get/:name", GetNamespaceByNameHandler)
	// rg.GET("/list/active", ListActiveNamespacesHandler)
	// rg.GET("/list/terminating", ListTerminatingNamespacesHandler)
	// rg.GET("/summary/status", GetNamespaceStatusSummaryHandler)
}

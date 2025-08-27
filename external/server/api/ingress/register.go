// =======================================================================================
// 📄 register.go（external/uiapi/ingress）
//
// ✨ 文件说明：
//     注册 Ingress 模块所有 REST 接口到 /uiapi/ingress 路由下。
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package ingress

import "github.com/gin-gonic/gin"

// RegisterIngressRoutes 注册 /uiapi/ingress 路由
func RegisterIngressRoutes(router *gin.RouterGroup) {
	router.GET("/list/all", GetAllIngressesHandler)

	//废弃
	// router.GET("/list/by-namespace/:ns", GetIngressesByNamespaceHandler)
	// router.GET("/get/:ns/:name", GetIngressByNameHandler)
	// router.GET("/list/ready", GetReadyIngressesHandler)
}

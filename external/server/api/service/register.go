// =======================================================================================
// 📄 register.go（external/uiapi/service）
//
// ✨ 文件说明：
//     注册 Service 模块的所有子路由至 /uiapi/service 路径下。
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package service

import "github.com/gin-gonic/gin"

// RegisterServiceRoutes 注册 Service 模块相关接口路由
func RegisterServiceRoutes(router *gin.RouterGroup) {
	router.GET("/list/all", GetAllServicesHandler)

	//废弃
	// router.GET("/list/by-namespace/:ns", GetServicesByNamespaceHandler)
	// router.GET("/get/:ns/:name", GetServiceByNameHandler)
	// router.GET("/list/external", GetExternalServicesHandler)
	// router.GET("/list/headless", GetHeadlessServicesHandler)
}

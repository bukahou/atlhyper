// =======================================================================================
// 📄 register.go（external/uiapi/configmap）
//
// ✨ 文件说明：
//     注册 ConfigMap 相关的所有路由接口：
//     - 按命名空间查询列表
//     - 查询详情
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package configmap

import (
	"github.com/gin-gonic/gin"
)

// RegisterConfigMapRoutes 将 ConfigMap 模块相关的路由注册到指定分组
func RegisterConfigMapRoutes(router *gin.RouterGroup) {
	router.GET("/list", ListAllConfigMapsHandler)                          // ✅ 所有命名空间
	router.GET("/list/by-namespace/:ns", ListConfigMapsByNamespaceHandler) // ✅ 指定命名空间
	router.GET("/get/:ns/:name", GetConfigMapDetailHandler)                // ✅ 详情查询
}

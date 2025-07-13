// =======================================================================================
// 📄 register.go（external/uiapi/deployment）
//
// ✨ 文件功能说明：
//     定义 Deployment 模块的 HTTP 路由注册逻辑，将 handler 中的各接口绑定至 REST 路由。
//     路径前缀：/uiapi/deployment/**
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 时间：2025年7月
// =======================================================================================

package deployment

import (
	"github.com/gin-gonic/gin"
)

// RegisterDeploymentRoutes 注册 Deployment 相关的路由
func RegisterDeploymentRoutes(rg *gin.RouterGroup) {
	rg.GET("/list/all", GetAllDeploymentsHandler)
	rg.GET("/list/by-namespace/:ns", GetDeploymentsByNamespaceHandler)
	rg.GET("/get/:ns/:name", GetDeploymentByNameHandler)
	rg.GET("/list/unavailable", GetUnavailableDeploymentsHandler)
	rg.GET("/list/progressing", GetProgressingDeploymentsHandler)
	rg.POST("/scale", ScaleDeploymentHandler)
}

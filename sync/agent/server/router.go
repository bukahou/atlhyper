package server

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册 Agent 所有路由到传入的路由组 r（路径前缀由外部决定）
func RegisterRoutes(r *gin.RouterGroup) {
	// 不再直接挂 ""，而是统一挂载在 "/agent"
	RegisterAllAgentRoutes(r.Group("/agent"))
}

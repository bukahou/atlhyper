// gateway/handler/api/overview/register.go
package overview

import (
	"github.com/gin-gonic/gin"
)

// Register 注册 Overview 相关路由
func Register(router *gin.RouterGroup) {
	g := router.Group("/overview")
	{
		g.GET("/cluster/list", GetClusterListHandler)
		g.POST("/cluster/detail", GetClusterDetailHandler)
	}
}

// gateway/handler/api/event/register.go
package event

import (
	"github.com/gin-gonic/gin"
)

// Register 注册事件相关路由
func Register(router *gin.RouterGroup) {
	g := router.Group("/event")
	{
		g.POST("/list", GetListHandler)
	}
}

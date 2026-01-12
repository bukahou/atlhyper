// gateway/handler/api/metrics/register.go
package metrics

import (
	"github.com/gin-gonic/gin"
)

// Register 注册指标相关路由
func Register(router *gin.RouterGroup) {
	g := router.Group("/metrics")
	{
		g.POST("/overview", GetOverviewHandler)
		g.POST("/node/detail", GetNodeDetailHandler)
	}
}

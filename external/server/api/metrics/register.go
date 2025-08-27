package metrics

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册指标相关的路由
func RegisterMetricsRoutes(r *gin.RouterGroup) {
	r.GET("/latest", GetInMemoryLatestHandler) 
	
}

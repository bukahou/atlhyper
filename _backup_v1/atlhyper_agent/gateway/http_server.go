// gateway/http_server.go
// HTTP 服务器路由注册
package gateway

import (
	"AtlHyper/atlhyper_agent/source/metrics"

	"github.com/gin-gonic/gin"
)

// RegisterIngestRoutes 注册数据摄入路由
func RegisterIngestRoutes(g *gin.RouterGroup) {
	metricsGroup := g.Group("/metrics")
	RegisterMetricsRoutes(metricsGroup)
}

// RegisterMetricsRoutes 注册 Metrics 相关路由
func RegisterMetricsRoutes(r *gin.RouterGroup) {
	r.POST("/v1/snapshot", metrics.HandlePostMetrics)
}

// internal/ingest/server/routes.go
package server

import (
	"NeuroController/internal/ingest/receivers"

	"github.com/gin-gonic/gin"
)

func RegisterIngestRoutes(g *gin.RouterGroup) {
	metricsGroup := g.Group("/metrics")
	RegisterMetricsRoutes(metricsGroup)
}

func RegisterMetricsRoutes(r *gin.RouterGroup) {
	// 直接把 handler 作为 Gin 的 HandlerFunc 绑定（无需闭包传参）
	r.POST("/v1/snapshot", receivers.HandlePostMetrics)
}

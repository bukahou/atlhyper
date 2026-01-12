package gateway

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	uiapi "AtlHyper/atlhyper_master/gateway/handler/api"
	"AtlHyper/atlhyper_master/gateway/handler/control"
	"AtlHyper/atlhyper_master/gateway/handler/ingest"
	"AtlHyper/atlhyper_master/gateway/middleware/audit"
)

// ============================================================================
// 🚨 errorOnlyLogger —— 仅记录 4xx / 5xx 错误访问日志
// ============================================================================
func errorOnlyLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := c.Writer.Status()
		if status >= 400 {
			latency := time.Since(start)
			msg := ""
			if len(c.Errors) > 0 {
				msg = c.Errors.String()
			}
			log.Printf("access_error method=%s path=%s status=%d latency=%s ip=%s err=%q",
				c.Request.Method, c.Request.URL.Path, status, latency, c.ClientIP(), msg)
		}
	}
}

// ============================================================================
// 🚀 InitRouter —— 初始化 HTTP 路由
// ============================================================================
// 说明：
//   - 前端已分离部署，本服务仅提供 API
//   - 告警配置（邮件/Slack）通过数据库管理，默认不触发
// ============================================================================
func InitRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(errorOnlyLogger())

	// ========================================================================
	// 💓 健康检查端点
	// ========================================================================
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ========================================================================
	// 🧩 UI API 模块（前端接口）
	// ------------------------------------------------------------------------
	// 提供集群监控、用户管理、配置、资源查询等 REST 接口
	// 所有 /uiapi/* 接口均带审计日志
	// ========================================================================
	api := r.Group("/uiapi")
	api.Use(audit.Auto(true))
	uiapi.RegisterUIAPIRoutes(api)

	// ========================================================================
	// 📥 Ingest 模块（Agent 数据上报入口）
	// ------------------------------------------------------------------------
	// 接收 Agent 推送的快照数据（事件、PodList、Metrics 等）
	// ========================================================================
	ing := r.Group("/ingest")
	ingest.RegisterIngestRoutes(ing)
	control.RegisterControlRoutes(ing)

	return r
}

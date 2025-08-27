// internal/bootstrapgo/agent_server.go
package bootstrapgo

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	ingestserver "NeuroController/internal/ingest/server"
	agentserver "NeuroController/sync/agent/server"
)

func StartAgentServer() {
	// 关闭 Gin 默认访问日志
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 仅记录错误请求（4xx/5xx）
	r.Use(func(c *gin.Context) {
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
	})

	// ===== 路由挂载 =====
	agentGroup := r.Group("/agent")
	agentserver.RegisterAllAgentRoutes(agentGroup) // ← 若原函数还带 metricsStore 参数，请同步去掉

	ingGroup := r.Group("/ingest")
	ingestserver.RegisterIngestRoutes(ingGroup) // ← 去掉旧的 store / maxBodyBytes 参数

	// ===== 启动 =====
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("❌ Agent 启动失败: %v", err)
	}
}

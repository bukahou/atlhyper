package bootstrapgo

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	ingestserver "NeuroController/internal/ingest/server"
	"NeuroController/internal/ingest/store"
	agentserver "NeuroController/sync/agent/server"

	"github.com/gin-gonic/gin"
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
			// 单行结构化风格
			log.Printf("access_error method=%s path=%s status=%d latency=%s ip=%s err=%q",
				c.Request.Method, c.Request.URL.Path, status, latency, c.ClientIP(), msg)
		}
	})

	// ===== Ingest 初始化（内存存储 + 定时清理）=====
	metricsStore := store.NewStore(10 * time.Minute) // 仅保留最近10分钟
	ctx, cancel := context.WithCancel(context.Background())
	metricsStore.StartJanitor(ctx, time.Minute) // 每1分钟清理一次过期数据

	// 优雅退出：收到信号时停止清理 goroutine
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		cancel()
	}()

	// ===== 路由挂载 =====
	agentGroup := r.Group("/agent")
	agentserver.RegisterAllAgentRoutes(agentGroup, metricsStore)

	ingGroup := r.Group("/ingest")
	ingestserver.RegisterIngestRoutes(ingGroup, metricsStore, 2<<20)

	// ===== 启动 =====
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("❌ Agent 启动失败: %v", err)
	}
}



// func StartAgentServer() {
// 	r := gin.Default()

// 	// ===== Ingest 初始化（内存存储 + 定时清理）=====
// 	metricsStore := store.NewStore(10 * time.Minute) // 仅保留最近10分钟
// 	ctx, cancel := context.WithCancel(context.Background())
// 	metricsStore.StartJanitor(ctx, time.Minute) // 每1分钟清理一次过期数据

// 	// 优雅退出：收到信号时停止清理 goroutine
// 	go func() {
// 		ch := make(chan os.Signal, 1)
// 		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
// 		<-ch
// 		cancel()
// 	}()

// 	// ===== 路由挂载 =====
// 	// 1) 原有 Agent 接口
// 	agentGroup := r.Group("/agent")
// 	// agentserver.RegisterAllAgentRoutes(agentGroup)
// 	agentserver.RegisterAllAgentRoutes(agentGroup, metricsStore)


// 	// 2) 新增 Ingest 接口
// 	ingGroup := r.Group("/ingest")
// 	ingestserver.RegisterIngestRoutes(ingGroup, metricsStore, 2<<20)

// 	// ===== 启动 =====
// 	if err := r.Run(":8082"); err != nil {
// 		log.Fatalf("❌ Agent 启动失败: %v", err)
// 	}
// }


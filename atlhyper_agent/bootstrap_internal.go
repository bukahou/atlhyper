package internal

import (
	"AtlHyper/atlhyper_agent/agent_store"
	"AtlHyper/atlhyper_agent/bootstrap"
	push "AtlHyper/atlhyper_agent/external"
	ingestserver "AtlHyper/atlhyper_agent/external/ingest/server"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// StartInternalSystems 启动 NeuroController 内部运行所需的所有基础子系统。
// 包括：
//   - 事件清理器（用于周期性处理原始 Kubernetes 事件）
//   - 日志写入器（将清理后的事件写入持久化日志文件）
//   - 集群健康检查器（周期性探测 API Server 健康状态）
//
// 该函数应在主程序启动时调用，以确保所有后台服务正常运行。
func StartInternalSystems() {
	// 打印启动日志，标记内部系统组件初始化流程开始
	log.Println("🚀 启动内部系统组件 ...")

	agent_store.Bootstrap()
	log.Println("✅ agent_store 初始化完成（全局单例 + 周期清理）")

	// ✅ 启动清理器：周期性清洗并压缩事件日志，形成可判定异常的结构化事件池
	bootstrap.StartCleanSystem()

	// ✅ 启动集群健康检查器：持续检查 Kubernetes API Server 的可用性
	bootstrap.Startclientchecker()

		// ✅ 启动上报器与 Agent HTTP（从 main.go 移到这里，确保只启动一次）
	go push.StartPusher()
	go StartAgentServer()


	// 所有子系统完成启动
	log.Println("✅ 所有内部组件启动完成。")
}


func StartAgentServer() {
	// 设置 Gin 为 Release 模式（关闭默认访问日志）
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// 自定义日志：仅记录 4xx/5xx 错误请求
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

	// /ingest 路由：只负责接收 Metrics 插件推送的数据快照
	ingGroup := r.Group("/ingest")
	ingestserver.RegisterIngestRoutes(ingGroup) 

	// ===== 启动服务 =====
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("❌ Agent 启动失败: %v", err)
	}
}

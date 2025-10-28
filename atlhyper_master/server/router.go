// // atlhyper_master/server/router.go
// package server

// import (
// 	"log"
// 	"time"

// 	"github.com/gin-gonic/gin"

// 	"AtlHyper/atlhyper_master/aiservice"
// 	"AtlHyper/atlhyper_master/control"
// 	"AtlHyper/atlhyper_master/ingest"
// 	uiapi "AtlHyper/atlhyper_master/server/api"
// 	"AtlHyper/atlhyper_master/server/audit"
// 	"AtlHyper/config"
// )

// // 仅记录 4xx/5xx 的访问日志（与 Agent 侧风格一致，单行）
// func errorOnlyLogger() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		start := time.Now()
// 		c.Next()

// 		status := c.Writer.Status()
// 		if status >= 400 {
// 			latency := time.Since(start)
// 			msg := ""
// 			if len(c.Errors) > 0 {
// 				msg = c.Errors.String()
// 			}
// 			log.Printf("access_error method=%s path=%s status=%d latency=%s ip=%s err=%q",
// 				c.Request.Method, c.Request.URL.Path, status, latency, c.ClientIP(), msg)
// 		}
// 	}
// }

// func InitRouter() *gin.Engine {
// 	gin.SetMode(gin.ReleaseMode)

// 	r := gin.New()
// 	r.Use(gin.Recovery())
// 	r.Use(errorOnlyLogger()) // ✅ 只打印 4xx/5xx；2xx/3xx（含 200/204/304）全部静默

// 	// ---- 静态与前端入口（保留你的写法） ----
// 	r.GET("/", func(c *gin.Context) { c.File("web/dist/index.html") })
// 	r.Static("/Atlhyper", "web/dist")
// 	r.GET("/Atlhyper", func(c *gin.Context) { c.File("web/dist/index.html") })
// 	r.NoRoute(func(c *gin.Context) { c.File("web/dist/index.html") })

// 	// ---- API ----
// 	api := r.Group("/uiapi")
// 	api.Use(audit.Auto(true))
// 	uiapi.RegisterUIAPIRoutes(api)

// 	// ---- Ingest ----
// 	ing := r.Group("/ingest")
// 	// ing.Use(audit.Auto(true)) // 需要审计再打开
// 	ingest.RegisterIngestRoutes(ing)

// 	// 维持你现在的控制路由（挂在 /ingest/ops/*，Agent 不会 404）
// 	control.RegisterControlRoutes(ing)

// 	// ✅ ---- AI Service ----
// 	ai := r.Group("/ai")
// 	aiservice.RegisterAISRoutes(ai)
// 	log.Println("🤖 AI Service routes registered under /ai/*")

// 	if config.GlobalConfig.Webhook.Enable {
// 		// webhook.RegisterWebhookRoutes(r.Group("/webhook"))
// 	} else {
// 		log.Println("⛔️ Webhook Server 已被禁用")
// 	}
// 	return r
// }

// ============================================================================
// 🌐 AtlHyper Master HTTP Router
// ----------------------------------------------------------------------------
// - 统一注册所有对外服务端点：UI API / Ingest / Control / AI / Webhook
// - 启动前端入口路由与静态资源
// - 启用错误级访问日志，仅记录 4xx/5xx
// ============================================================================

package server

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"AtlHyper/atlhyper_master/aiservice"        // 🤖 AI Service 模块
	"AtlHyper/atlhyper_master/control"          // ⚙️ 控制路由（Pod 操作、日志等）
	"AtlHyper/atlhyper_master/ingest"           // 📥 Agent 数据上报入口
	uiapi "AtlHyper/atlhyper_master/server/api" // 🧩 UI 前端接口
	"AtlHyper/atlhyper_master/server/audit"     // 🧾 审计中间件
	"AtlHyper/config"
)

// ============================================================================
// 🚨 errorOnlyLogger —— 仅记录 4xx / 5xx 错误访问日志
// ----------------------------------------------------------------------------
// 与 Agent 端保持一致风格：单行结构化输出。
// 2xx / 3xx （正常请求）全部静默不打印。
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
// 🚀 InitRouter —— 初始化 HTTP 路由与所有模块的注册
// ----------------------------------------------------------------------------
// 作用：
//   1. 注册静态页面与前端入口（Vue / Web UI）
//   2. 注册业务 API 模块（UI API, Ingest, Control, AI）
//   3. 加载审计、日志、Webhook 等扩展逻辑
// ============================================================================
func InitRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(errorOnlyLogger()) // ✅ 启用错误日志中间件

	// ============================================================================
	// 🏠 [1] 静态页面与前端入口
	// ----------------------------------------------------------------------------
	// 保留原有 Web Dist 入口逻辑：
	//   - /             → index.html
	//   - /Atlhyper     → 静态资源目录
	//   - NoRoute       → 默认跳转到前端入口
	// ============================================================================
	r.GET("/", func(c *gin.Context) { c.File("web/dist/index.html") })
	r.Static("/Atlhyper", "web/dist")
	r.GET("/Atlhyper", func(c *gin.Context) { c.File("web/dist/index.html") })
	r.NoRoute(func(c *gin.Context) { c.File("web/dist/index.html") })

	// ============================================================================
	// 🧩 [2] UI API 模块（前端接口）
	// ----------------------------------------------------------------------------
	// 作用：提供集群监控、用户管理、配置、资源查询等 REST 接口。
	// 所有 /uiapi/* 接口均带审计日志。
	// ============================================================================
	api := r.Group("/uiapi")
	api.Use(audit.Auto(true))
	uiapi.RegisterUIAPIRoutes(api)

	// ============================================================================
	// 📥 [3] Ingest 模块（Agent 数据上报入口）
	// ----------------------------------------------------------------------------
	// 作用：接收 Agent 推送的快照类数据（事件、PodList、Metrics 等）。
	// ============================================================================
	ing := r.Group("/ingest")
	// ing.Use(audit.Auto(true)) // 可选开启审计
	ingest.RegisterIngestRoutes(ing)

	// ============================================================================
	// ⚙️ [4] Control 模块（Agent 操作控制接口）
	// ----------------------------------------------------------------------------
	// 作用：提供 /ingest/ops/* 路由，用于 Agent 侧或 Web 侧的操作调用。
	// ============================================================================
	control.RegisterControlRoutes(ing)

	// ============================================================================
	// 🤖 [5] AI Service 模块（AI 分析交互接口）
	// ----------------------------------------------------------------------------
	// 作用：提供 /ai/* 路由，供 AI Service 拉取分析上下文或提交分析任务。
	// ============================================================================
	ai := r.Group("/ai")
	aiservice.RegisterAISRoutes(ai)
	log.Println("🤖 AI Service routes registered under /ai/*")

	// ============================================================================
	// 🔗 [6] Webhook 模块（外部事件回调）
	// ----------------------------------------------------------------------------
	// 作用：统一接收来自 GitHub / DockerHub 等外部事件触发。
	// ============================================================================
	if config.GlobalConfig.Webhook.Enable {
		// webhook.RegisterWebhookRoutes(r.Group("/webhook"))
	} else {
		log.Println("⛔️ Webhook Server 已被禁用")
	}

	// ============================================================================
	// ✅ 路由初始化完成
	// ============================================================================
	return r
}

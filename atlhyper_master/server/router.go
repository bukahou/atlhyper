// package server

// import (
// 	"AtlHyper/atlhyper_master/control"
// 	"AtlHyper/atlhyper_master/ingest"
// 	uiapi "AtlHyper/atlhyper_master/server/api" // 📦 UI REST 接口注册模块
// 	"AtlHyper/atlhyper_master/server/audit"
// 	"AtlHyper/config"

// 	// 📦 Webhook 路由模块（CI/CD）
// 	"log"

// 	"github.com/gin-gonic/gin"
// )

// func InitRouter() *gin.Engine {
//     r := gin.Default()

//     // 1) 根路径：直接返回前端首页（不再 302）
//     r.GET("/", func(c *gin.Context) {
//         c.File("web/dist/index.html")
//     })

//     // 2) 前端静态资源挂在 /Atlhyper（与你的 Ingress 设计兼容）
//     r.Static("/Atlhyper", "web/dist")

//     // 3) 访问 /Atlhyper（无 /）时直接出首页，避免多一次 302
//     r.GET("/Atlhyper", func(c *gin.Context) {
//         c.File("web/dist/index.html")
//     })

//     // 4) 任意未命中路由 → 直接给前端首页，避免再重定向
//     r.NoRoute(func(c *gin.Context) {
//         c.File("web/dist/index.html")
//     })

//     // 5) API
//     api := r.Group("/uiapi")
//     api.Use(audit.Auto(true))
//     uiapi.RegisterUIAPIRoutes(api)

//     ing := r.Group("/ingest")
// 	// 如果希望也记审计，可以打开下一行：
// 	// ing.Use(audit.Auto(true))
// 	ingest.RegisterIngestRoutes(ing)

//     // ctl := r.Group("/control")
//     control.RegisterControlRoutes(ing)

//     if config.GlobalConfig.Webhook.Enable {
//         // webhook.RegisterWebhookRoutes(r.Group("/webhook"))
//     } else {
//         log.Println("⛔️ Webhook Server 已被禁用")
//     }
//     return r
// }

// server/router.go
package server

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"AtlHyper/atlhyper_master/control"
	"AtlHyper/atlhyper_master/ingest"
	uiapi "AtlHyper/atlhyper_master/server/api"
	"AtlHyper/atlhyper_master/server/audit"
	"AtlHyper/config"
)

// 仅记录 4xx/5xx 的访问日志（与 Agent 侧风格一致，单行）
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

func InitRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(errorOnlyLogger()) // ✅ 只打印 4xx/5xx；2xx/3xx（含 200/204/304）全部静默

	// ---- 静态与前端入口（保留你的写法） ----
	r.GET("/", func(c *gin.Context) { c.File("web/dist/index.html") })
	r.Static("/Atlhyper", "web/dist")
	r.GET("/Atlhyper", func(c *gin.Context) { c.File("web/dist/index.html") })
	r.NoRoute(func(c *gin.Context) { c.File("web/dist/index.html") })

	// ---- API ----
	api := r.Group("/uiapi")
	api.Use(audit.Auto(true))
	uiapi.RegisterUIAPIRoutes(api)

	// ---- Ingest ----
	ing := r.Group("/ingest")
	// ing.Use(audit.Auto(true)) // 需要审计再打开
	ingest.RegisterIngestRoutes(ing)

	// 维持你现在的控制路由（挂在 /ingest/ops/*，Agent 不会 404）
	control.RegisterControlRoutes(ing)

	if config.GlobalConfig.Webhook.Enable {
		// webhook.RegisterWebhookRoutes(r.Group("/webhook"))
	} else {
		log.Println("⛔️ Webhook Server 已被禁用")
	}
	return r
}

package server

import (
	"AtlHyper/atlhyper_master/control"
	"AtlHyper/atlhyper_master/ingest"
	uiapi "AtlHyper/atlhyper_master/server/api" // 📦 UI REST 接口注册模块
	"AtlHyper/atlhyper_master/server/audit"
	"AtlHyper/config"

	// 📦 Webhook 路由模块（CI/CD）
	"log"

	"github.com/gin-gonic/gin"
)




func InitRouter() *gin.Engine {
    r := gin.Default()

    // 1) 根路径：直接返回前端首页（不再 302）
    r.GET("/", func(c *gin.Context) {
        c.File("web/dist/index.html")
    })

    // 2) 前端静态资源挂在 /Atlhyper（与你的 Ingress 设计兼容）
    r.Static("/Atlhyper", "web/dist")

    // 3) 访问 /Atlhyper（无 /）时直接出首页，避免多一次 302
    r.GET("/Atlhyper", func(c *gin.Context) {
        c.File("web/dist/index.html")
    })

    // 4) 任意未命中路由 → 直接给前端首页，避免再重定向
    r.NoRoute(func(c *gin.Context) {
        c.File("web/dist/index.html")
    })

    // 5) API
    api := r.Group("/uiapi")
    api.Use(audit.Auto(true))
    uiapi.RegisterUIAPIRoutes(api)


    ing := r.Group("/ingest")
	// 如果希望也记审计，可以打开下一行：
	// ing.Use(audit.Auto(true))
	ingest.RegisterIngestRoutes(ing)

    ctl := r.Group("/control")
    control.RegisterControlRoutes(ctl)

    if config.GlobalConfig.Webhook.Enable {
        // webhook.RegisterWebhookRoutes(r.Group("/webhook"))
    } else {
        log.Println("⛔️ Webhook Server 已被禁用")
    }
    return r
}

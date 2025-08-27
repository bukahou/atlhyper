// 📄 external/server/router.go
//
// 🌐 Gin 路由初始化模块
//
// 说明：
//     - 初始化并返回 Gin 路由引擎（*gin.Engine）
//     - 负责注册静态页面资源、前端 UI API 接口、Webhook 接口等
//
// 用法：
//     - 在 main.go 中调用 InitRouter() 以启动 HTTP 服务
//
// 作者：@bukahou
// 更新时间：2025年7月

package server

import (
	"NeuroController/config"
	"NeuroController/external/ingest"
	uiapi "NeuroController/external/server/api" // 📦 UI REST 接口注册模块
	"NeuroController/external/server/audit"

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


    if config.GlobalConfig.Webhook.Enable {
        // webhook.RegisterWebhookRoutes(r.Group("/webhook"))
    } else {
        log.Println("⛔️ Webhook Server 已被禁用")
    }
    return r
}


// InitRouter 初始化并返回 Gin 路由引擎
// func InitRouter() *gin.Engine {
// 	// ✅ 创建默认路由引擎，内置 Logger 与 Recovery 中间件
// 	router := gin.Default()

// 	// ✅ 挂载静态资源目录：/Atlhper 对应本地 ./web 目录
// 	//     浏览器访问 /Atlhper/index.html 会映射为 web/index.html 文件
// 	router.Static("/Atlhyper", "web/dist")

// 	// ✅ 首页重定向：访问 /Atlhper 会被 302 跳转至 /Atlhper/index.html
// 	router.GET("/Atlhyper", func(c *gin.Context) {
// 		c.Redirect(302, "/Atlhyper/index.html")
// 	})

// 	// ✅ 注册 UI API 路由（如 /uiapi/node/list 等）
// 	// uiapi.RegisterUIAPIRoutes(router.Group("/uiapi"))
// 	api := router.Group("/uiapi")
//     api.Use(audit.Auto(true)) // true = 高风险成功也记；false = 只记失败
//     uiapi.RegisterUIAPIRoutes(api)

// 	// ✅ 可选注册 Webhook 路由（如 /webhook/dockerhub 等）
// 	if config.GlobalConfig.Webhook.Enable {
// 		// webhook.RegisterWebhookRoutes(router.Group("/webhook"))
// 	} else {
// 		log.Println("⛔️ Webhook Server 已被禁用")
// 	}

// 	// ✅ 返回构建完成的路由引擎
// 	return router
// }
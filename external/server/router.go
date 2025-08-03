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
	"NeuroController/external/uiapi" // 📦 UI REST 接口注册模块

	// 📦 Webhook 路由模块（CI/CD）
	"log"

	"github.com/gin-gonic/gin"
)

// InitRouter 初始化并返回 Gin 路由引擎
func InitRouter() *gin.Engine {
	// ✅ 创建默认路由引擎，内置 Logger 与 Recovery 中间件
	router := gin.Default()

	// ✅ 挂载静态资源目录：/Atlhper 对应本地 ./web 目录
	//     浏览器访问 /Atlhper/index.html 会映射为 web/index.html 文件
	router.Static("/Atlhyper", "web/dist")


	// ✅ 首页重定向：访问 /Atlhper 会被 302 跳转至 /Atlhper/index.html
	router.GET("/Atlhyper", func(c *gin.Context) {
		c.Redirect(302, "/Atlhyper/index.html")
	})


	// ✅ 注册 UI API 路由（如 /uiapi/node/list 等）
	uiapi.RegisterUIAPIRoutes(router.Group("/uiapi"))

	// ✅ 可选注册 Webhook 路由（如 /webhook/dockerhub 等）
	if config.GlobalConfig.Webhook.Enable {
		// webhook.RegisterWebhookRoutes(router.Group("/webhook"))
	} else {
		log.Println("⛔️ Webhook Server 已被禁用")
	}

	// ✅ 返回构建完成的路由引擎
	return router
}

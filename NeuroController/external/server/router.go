// 📄 external/server/router.go

package server

import (
	"NeuroController/config"
	"NeuroController/external/uiapi"
	"NeuroController/external/webhook"
	"log"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	router := gin.Default()

	// ✅ 正确挂载静态资源路径：web 是相对当前工作目录
	router.Static("/Atlhper", "web")

	// ✅ 重定向 /Atlhper → /Atlhper/index.html
	router.GET("/Atlhper", func(c *gin.Context) {
		c.Redirect(302, "/Atlhper/index.html")
	})

	// ✅ 注册 UI API 接口
	uiapi.RegisterUIAPIRoutes(router.Group("/uiapi"))

	// ✅ 注册 webhook（根据配置）
	if config.GlobalConfig.Webhook.Enable {
		webhook.RegisterWebhookRoutes(router.Group("/webhook"))
	} else {
		log.Println("⛔️ Webhook Server 已被禁用")
	}

	return router
}

// 📄 external/server/router.go

package server

import (
	"NeuroController/external/webhook"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	router := gin.Default()

	// Webhook 路由组：/webhook/**
	webhook.RegisterWebhookRoutes(router.Group("/webhook"))

	// 健康检查等可拓展：
	// router.GET("/healthz", func(c *gin.Context) {
	// 	c.String(200, "ok")
	// })

	return router
}

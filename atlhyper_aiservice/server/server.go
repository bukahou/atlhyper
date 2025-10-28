// atlhyper_aiservice/server/server.go
package server

import (
	"AtlHyper/atlhyper_aiservice/router"
	"log"

	"github.com/gin-gonic/gin"
)

// StartHTTPServer 启动 Gin 服务
func StartHTTPServer() {
	r := gin.Default()
	router.RegisterRoutes(r)

	port := ":8089"
	log.Printf("🌐 AI Service HTTP Server 正在监听 %s ...", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("❌ HTTP 服务启动失败: %v", err)
	}
}


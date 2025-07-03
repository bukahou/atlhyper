// 📄 external/server/server.go

package server

import (
	"log"
	"net/http"
)

// StartHTTPServer 启动 Gin HTTP 服务器
func StartHTTPServer() {
	router := InitRouter()

	addr := ":8081" // 或使用 config 管理
	log.Printf("🚀 Webhook Server 启动监听 %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("❌ 启动 HTTP 服务失败: %v", err)
	}
}

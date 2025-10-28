//atlhyper_master/server/server.go

package server

import (
	"log"
	"net/http"
)

// corsMiddleware 是一个 HTTP 中间件，用于处理跨域请求（CORS）
// ✅ 允许任意来源、指定的方法和头部，支持预检请求（OPTIONS）
func corsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 允许所有域名跨域访问（生产环境可改为指定域名）
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// 允许的 HTTP 方法
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// 允许的请求头
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 如果是预检请求（OPTIONS），直接返回 200
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 非预检请求则继续处理
		h.ServeHTTP(w, r)
	})
}

// StartHTTPServer 启动 Gin HTTP 服务器（端口: 8081）
// ✅ 加载 InitRouter() 构建的所有路由，自动绑定 CORS 支持
func StartHTTPServer() {
	// 初始化 Gin 路由
	router := InitRouter()

	// 启动监听地址
	addr := ":8081"
	log.Printf("🚀 Web UI API Server 启动监听 %s", addr)

	// 启动 HTTP 服务（加上 CORS 中间件）
	if err := http.ListenAndServe(addr, corsMiddleware(router)); err != nil {
		log.Fatalf("❌ 启动 HTTP 服务失败: %v", err)
	}
}

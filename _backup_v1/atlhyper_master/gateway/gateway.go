//atlhyper_master/server/server.go

package gateway

import (
	"log"
	"net/http"

	"AtlHyper/atlhyper_master/config"
)

// corsMiddleware 是一个 HTTP 中间件，用于处理跨域请求（CORS）
// 配置通过 config.GlobalConfig.CORS 读取
func corsMiddleware(h http.Handler) http.Handler {
	cors := config.GlobalConfig.CORS

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置 CORS 头（从配置读取）
		w.Header().Set("Access-Control-Allow-Origin", cors.AllowOrigins)
		w.Header().Set("Access-Control-Allow-Methods", cors.AllowMethods)
		w.Header().Set("Access-Control-Allow-Headers", cors.AllowHeaders)

		// 允许凭证（仅当 AllowOrigins 不是 "*" 时才能设为 true）
		if cors.AllowCredentials && cors.AllowOrigins != "*" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// 如果是预检请求（OPTIONS），直接返回 200
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 非预检请求则继续处理
		h.ServeHTTP(w, r)
	})
}

// StartHTTPServer 启动 Gin HTTP 服务器
// ✅ 加载 InitRouter() 构建的所有路由，自动绑定 CORS 支持
// 端口通过 config.GlobalConfig.Server.Port 配置（环境变量 SERVER_PORT，默认 8080）
func StartHTTPServer() {
	// 初始化 Gin 路由
	router := InitRouter()

	// 启动监听地址（从统一配置读取）
	addr := ":" + config.GlobalConfig.Server.Port
	log.Printf("🚀 Web UI API Server 启动监听 %s", addr)

	// 启动 HTTP 服务（加上 CORS 中间件）
	if err := http.ListenAndServe(addr, corsMiddleware(router)); err != nil {
		log.Fatalf("❌ 启动 HTTP 服务失败: %v", err)
	}
}

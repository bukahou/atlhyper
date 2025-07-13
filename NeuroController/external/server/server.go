// 📄 external/server/server.go

package server

import (
	"log"
	"net/http"
)

func corsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// StartHTTPServer 启动 Gin HTTP 服务器 + CORS 支持
func StartHTTPServer() {
	router := InitRouter()
	addr := ":8081"
	log.Printf("🚀 Web UI API Server 启动监听 %s", addr)

	if err := http.ListenAndServe(addr, corsMiddleware(router)); err != nil {
		log.Fatalf("❌ 启动 HTTP 服务失败: %v", err)
	}
}

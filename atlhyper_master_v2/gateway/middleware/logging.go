// atlhyper_master_v2/gateway/middleware/logging.go
// 日志中间件
package middleware

import (
	"log"
	"net/http"
	"time"
)

// responseWriter 包装 ResponseWriter 以获取状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush 实现 http.Flusher 接口（SSE 流式响应需要）
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Logging 日志中间件
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 处理请求
		next.ServeHTTP(rw, r)

		// 记录日志
		duration := time.Since(start)
		log.Printf("[Gateway] %s %s %d %v",
			r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

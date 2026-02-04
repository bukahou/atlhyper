// atlhyper_master_v2/gateway/middleware/logging.go
// 日志中间件
package middleware

import (
	"net/http"
	"time"

	"AtlHyper/common/logger"
)

var log = logger.Module("Gateway")

// 慢请求阈值
const slowRequestThreshold = 1 * time.Second

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
// 智能日志级别：
// - 非 2xx 响应 → INFO
// - 慢请求 (>1s) → INFO
// - 非 GET 请求 → INFO
// - 其他 → DEBUG
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// 处理请求
		next.ServeHTTP(rw, r)

		// 计算耗时
		duration := time.Since(start)

		// 判断是否需要输出 INFO 日志
		isError := rw.statusCode < 200 || rw.statusCode >= 400
		isSlow := duration > slowRequestThreshold
		isWriteRequest := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions

		if isError || isSlow || isWriteRequest {
			log.Info("HTTP 请求",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration", logger.Duration(duration),
			)
		} else {
			log.Debug("HTTP 请求",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration", logger.Duration(duration),
			)
		}
	})
}

package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"

	"AtlHyper/atlhyper_master_v2/gateway/clientip"
)

// RateLimitByClientIP 按客户端地址限流（每分钟 perMinute 次，超出返回 429）。
// 客户端地址见 clientip.FromRequest：没配可信头时所有请求共用 TCP 对端（入口代理）这一个键。
func RateLimitByClientIP(perMinute int, clientIPHeader string) func(http.Handler) http.Handler {
	return httprate.Limit(perMinute, time.Minute,
		httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
			return clientip.FromRequest(r, clientIPHeader), nil
		}),
		httprate.WithLimitHandler(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"请求过于频繁，请稍后再试"}`))
		}),
	)
}

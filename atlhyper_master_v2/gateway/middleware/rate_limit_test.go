package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitByClientIP(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RateLimitByClientIP(2, "CF-Connecting-IP")(ok)

	call := func(ip, xff string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/user/login", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		req.Header.Set("CF-Connecting-IP", ip)
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if call("203.0.113.1", "") != http.StatusOK || call("203.0.113.1", "") != http.StatusOK {
		t.Fatal("上限内的请求应放行")
	}
	if got := call("203.0.113.1", "198.51.100.9"); got != http.StatusTooManyRequests {
		t.Fatalf("超过上限 = %d, want 429（换 X-Forwarded-For 不能绕过）", got)
	}
	if got := call("203.0.113.2", ""); got != http.StatusOK {
		t.Errorf("另一个客户端不受影响，实际 %d", got)
	}
}

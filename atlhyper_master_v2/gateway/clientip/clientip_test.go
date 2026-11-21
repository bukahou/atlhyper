package clientip

import (
	"net/http/httptest"
	"testing"
)

func TestFromRequest(t *testing.T) {
	cases := []struct {
		name    string
		header  string // 配置的可信头名
		headers map[string]string
		remote  string
		want    string
	}{
		{"没配可信头：取对端地址", "", map[string]string{"X-Forwarded-For": "1.2.3.4"}, "10.0.0.5:4321", "10.0.0.5"},
		{"配了可信头且存在", "CF-Connecting-IP", map[string]string{"CF-Connecting-IP": "203.0.113.7"}, "10.0.0.5:4321", "203.0.113.7"},
		{"可信头里是 IPv6", "CF-Connecting-IP", map[string]string{"CF-Connecting-IP": "2001:db8::1"}, "10.0.0.5:4321", "2001:db8::1"},
		{"配了可信头但请求没带：回落对端地址", "CF-Connecting-IP", nil, "10.0.0.5:4321", "10.0.0.5"},
		{"可信头的值不是 IP：回落对端地址", "CF-Connecting-IP", map[string]string{"CF-Connecting-IP": "evil, 1.2.3.4"}, "10.0.0.5:4321", "10.0.0.5"},
		{"X-Forwarded-For 一律不信", "CF-Connecting-IP", map[string]string{"X-Forwarded-For": "1.2.3.4"}, "10.0.0.5:4321", "10.0.0.5"},
		{"对端地址没有端口", "", nil, "10.0.0.5", "10.0.0.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = c.remote
			for k, v := range c.headers {
				req.Header.Set(k, v)
			}
			if got := FromRequest(req, c.header); got != c.want {
				t.Errorf("FromRequest = %q, want %q", got, c.want)
			}
		})
	}
}

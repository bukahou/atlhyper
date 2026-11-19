// Package clientip 取发起请求的客户端地址：全项目只有这一处。
//
// ⛔ 不信 X-Forwarded-For：它的第一段是客户端自己写的，任何人都能伪造。
// 只信一个由部署配置指定、由最外层入口（如 CDN）覆盖写入的头；没配或请求里没有时，用 TCP 对端地址。
package clientip

import (
	"net"
	"net/http"
	"strings"
)

// FromRequest 返回客户端 IP。trustedHeader 为空表示不信任任何头。
func FromRequest(r *http.Request, trustedHeader string) string {
	if trustedHeader != "" {
		if v := strings.TrimSpace(r.Header.Get(trustedHeader)); v != "" {
			if ip := net.ParseIP(v); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

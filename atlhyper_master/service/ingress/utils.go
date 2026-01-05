// ui_interfaces/ingress/utils.go
package ingress

import (
	"strconv"
	"strings"

	mod "AtlHyper/model/k8s"
)

// backendServiceName —— 获取 backend 的 service 名（若非 Service 类型则空）
func backendServiceName(b *mod.BackendRef) string {
	if b == nil || b.Service == nil {
		return ""
	}
	return b.Service.Name
}

// backendServicePortString —— 统一端口为字符串（支持端口名/端口号）
func backendServicePortString(b *mod.BackendRef) string {
	if b == nil || b.Service == nil {
		return ""
	}
	if b.Service.PortName != "" {
		return b.Service.PortName
	}
	if b.Service.PortNumber != 0 {
		return strconv.Itoa(int(b.Service.PortNumber))
	}
	return ""
}

// joinHostsFromTLS —— 逗号拼接所有 TLS hosts（去重）
func joinHostsFromTLS(tls []mod.IngressTLS) string {
	if len(tls) == 0 {
		return ""
	}
	set := make(map[string]struct{})
	for _, t := range tls {
		for _, h := range t.Hosts {
			hs := strings.TrimSpace(h)
			if hs == "" {
				continue
			}
			set[hs] = struct{}{}
		}
	}
	if len(set) == 0 {
		return ""
	}
	out := make([]string, 0, len(set))
	for h := range set {
		out = append(out, h)
	}
	// 稍微稳定化：按字典序
	sortStrings(out)
	return strings.Join(out, ", ")
}

// 本地小排序，避免额外依赖
func sortStrings(a []string) {
	if len(a) < 2 {
		return
	}
	for i := 0; i < len(a)-1; i++ {
		for j := i + 1; j < len(a); j++ {
			if a[j] < a[i] {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}

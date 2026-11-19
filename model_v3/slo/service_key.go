package slo

import "strings"

// ServiceKey 入口指标里一个后端服务的标识："namespace/service"。
//
// 用 K8s 原生的命名空间 + 服务名，而不是某个入口实现的内部名（如 Traefik 的
// "ns-svc-port@kubernetes"）—— 换实现不变。Agent 产出 SLO 数据、解析路由资源，
// Master 建路由映射、AIOps 定实体键，都只认这一份格式。
func ServiceKey(namespace, service string) string {
	if service == "" {
		return ""
	}
	if namespace == "" {
		return service
	}
	return namespace + "/" + service
}

// SplitServiceKey ServiceKey 的逆运算；没有命名空间时 namespace 为空串
func SplitServiceKey(key string) (namespace, service string) {
	if i := strings.IndexByte(key, '/'); i >= 0 {
		return key[:i], key[i+1:]
	}
	return "", key
}

// MatchesHost 用户按「域名」指认的入口是否就是这一个。
//
// 域名来自 Agent 从路由资源解析出的 DisplayName；集群没有路由资源时页面以 ServiceKey 代替域名，
// 两种写法都要认。SLO 页面、AIOps 的燃烧率都按这一个规则找入口。
func (s *IngressSLO) MatchesHost(host string) bool {
	return host != "" && (s.DisplayName == host || s.ServiceKey == host)
}

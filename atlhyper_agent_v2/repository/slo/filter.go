// Package slo SLO 数据仓库
//
// filter.go - Stage 1: 过滤
//
// 排除不应计入 SLO 的流量:
//   - K8s 健康检查 (route_name="probe")
//   - Linkerd proxy admin 端口 (srv_port="4191")
//   - 系统 namespace (linkerd, linkerd-viz, kube-system, otel 等)
//
// 过滤后的数据按 direction 分类:
//   - inbound → ServiceMetrics 聚合
//   - outbound → ServiceEdge 提取
package slo

import (
	"AtlHyper/atlhyper_agent_v2/sdk"
)

// filter 过滤 OTelRawMetrics，排除非业务流量
//
// 返回过滤后的副本，不修改原始数据。
func (r *sloRepository) filter(raw *sdk.OTelRawMetrics) *sdk.OTelRawMetrics {
	result := &sdk.OTelRawMetrics{}

	// ---- Linkerd responses ----
	for _, m := range raw.LinkerdResponses {
		if r.shouldExcludeLinkerd(m.RouteName, m.SrvPort, m.Namespace) {
			continue
		}
		result.LinkerdResponses = append(result.LinkerdResponses, m)
	}

	// ---- Linkerd latency buckets ----
	for _, m := range raw.LinkerdLatencyBuckets {
		if r.shouldExcludeNamespace(m.Namespace) {
			continue
		}
		result.LinkerdLatencyBuckets = append(result.LinkerdLatencyBuckets, m)
	}

	// ---- Linkerd latency sums ----
	for _, m := range raw.LinkerdLatencySums {
		if r.shouldExcludeNamespace(m.Namespace) {
			continue
		}
		result.LinkerdLatencySums = append(result.LinkerdLatencySums, m)
	}

	// ---- Linkerd latency counts ----
	for _, m := range raw.LinkerdLatencyCounts {
		if r.shouldExcludeNamespace(m.Namespace) {
			continue
		}
		result.LinkerdLatencyCounts = append(result.LinkerdLatencyCounts, m)
	}

	// ---- Ingress 指标不需要过滤 namespace（按 serviceKey 维度） ----
	result.IngressRequests = raw.IngressRequests
	result.IngressLatencyBuckets = raw.IngressLatencyBuckets
	result.IngressLatencySums = raw.IngressLatencySums
	result.IngressLatencyCounts = raw.IngressLatencyCounts

	return result
}

// shouldExcludeLinkerd 判断 Linkerd 请求指标是否应排除
func (r *sloRepository) shouldExcludeLinkerd(routeName, srvPort, namespace string) bool {
	// 1. 排除 K8s 健康检查
	if routeName == "probe" {
		return true
	}
	// 2. 排除 Linkerd proxy admin 端口
	if srvPort == "4191" {
		return true
	}
	// 3. 排除系统 namespace
	return r.shouldExcludeNamespace(namespace)
}

// shouldExcludeNamespace 判断 namespace 是否在排除列表中
func (r *sloRepository) shouldExcludeNamespace(namespace string) bool {
	for _, ns := range r.excludeNamespaces {
		if namespace == ns {
			return true
		}
	}
	return false
}

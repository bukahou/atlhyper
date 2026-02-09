// Package otel OTel Collector 采集客户端实现
//
// parser.go - Prometheus 文本格式解析
//
// 逐行扫描 OTel Collector 输出的 Prometheus 文本，
// 按指标名前缀分类为 OTelRawMetrics 的各个子结构。
//
// 处理逻辑:
//  1. 跳过注释和空行
//  2. 提取 metric_name{labels} value
//  3. 按前缀分发:
//     - otel_response_total          → LinkerdResponseMetric
//     - otel_response_latency_ms_*   → LinkerdLatency*Metric
//     - otel_traefik_*               → Ingress*Metric (归一化)
//     - otel_nginx_ingress_*         → Ingress*Metric (归一化)
//     其他指标丢弃
//  4. 入口指标归一化: 不同 Controller 的标签映射到统一结构
package otel

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// 指标名前缀常量
const (
	// Linkerd 指标（OTel 前缀）
	prefixLinkerdResponse       = "otel_response_total"
	prefixLinkerdLatencyBucket  = "otel_response_latency_ms_bucket"
	prefixLinkerdLatencySum     = "otel_response_latency_ms_sum"
	prefixLinkerdLatencyCount   = "otel_response_latency_ms_count"

	// Traefik 指标（OTel 前缀）
	prefixTraefikRequests       = "otel_traefik_service_requests_total"
	prefixTraefikLatencyBucket  = "otel_traefik_service_request_duration_seconds_bucket"
	prefixTraefikLatencySum     = "otel_traefik_service_request_duration_seconds_sum"
	prefixTraefikLatencyCount   = "otel_traefik_service_request_duration_seconds_count"

	// Nginx 指标（OTel 前缀）
	prefixNginxRequests         = "otel_nginx_ingress_controller_requests"
	prefixNginxLatencyBucket    = "otel_nginx_ingress_controller_request_duration_seconds_bucket"
	prefixNginxLatencySum       = "otel_nginx_ingress_controller_request_duration_seconds_sum"
	prefixNginxLatencyCount     = "otel_nginx_ingress_controller_request_duration_seconds_count"
)

// 正则: 匹配 metric_name{...} value 或 metric_name value
var metricLineRegex = regexp.MustCompile(`^(\w+)(\{[^}]*\})?\s+(.+)$`)

// parsePrometheus 解析 Prometheus 文本格式为 OTelRawMetrics
//
// 逐行扫描，按指标名分发到对应切片。
// 不关心的指标（如 go_*, process_* 等）直接跳过。
func parsePrometheus(r io.Reader) (*sdk.OTelRawMetrics, error) {
	result := &sdk.OTelRawMetrics{}

	scanner := bufio.NewScanner(r)
	// 增大缓冲区，OTel Collector 输出可能有很长的行
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		// 跳过注释和空行
		if line == "" || line[0] == '#' {
			continue
		}

		// 快速前缀过滤：只处理 otel_ 开头的指标
		if !strings.HasPrefix(line, "otel_") {
			continue
		}

		// 提取 metric_name, labels, value
		matches := metricLineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		name := matches[1]
		labelsRaw := matches[2] // 可能为空（无标签）
		valueStr := matches[3]

		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		// 去掉标签外层的 { }
		var labels map[string]string
		if labelsRaw != "" {
			labels = parseLabels(labelsRaw[1 : len(labelsRaw)-1])
		} else {
			labels = make(map[string]string)
		}

		// 按指标名分发
		switch name {
		// ---- Linkerd ----
		case prefixLinkerdResponse:
			result.LinkerdResponses = append(result.LinkerdResponses, parseLinkerdResponse(labels, value))

		case prefixLinkerdLatencyBucket:
			result.LinkerdLatencyBuckets = append(result.LinkerdLatencyBuckets, parseLinkerdLatencyBucket(labels, value))

		case prefixLinkerdLatencySum:
			result.LinkerdLatencySums = append(result.LinkerdLatencySums, parseLinkerdLatencySum(labels, value))

		case prefixLinkerdLatencyCount:
			result.LinkerdLatencyCounts = append(result.LinkerdLatencyCounts, parseLinkerdLatencyCount(labels, value))

		// ---- Traefik (归一化为 Ingress*) ----
		case prefixTraefikRequests:
			result.IngressRequests = append(result.IngressRequests, parseTraefikRequest(labels, value))

		case prefixTraefikLatencyBucket:
			result.IngressLatencyBuckets = append(result.IngressLatencyBuckets, parseTraefikLatencyBucket(labels, value))

		case prefixTraefikLatencySum:
			result.IngressLatencySums = append(result.IngressLatencySums, parseTraefikLatencySum(labels, value))

		case prefixTraefikLatencyCount:
			result.IngressLatencyCounts = append(result.IngressLatencyCounts, parseTraefikLatencyCount(labels, value))

		// ---- Nginx (归一化为 Ingress*) ----
		case prefixNginxRequests:
			result.IngressRequests = append(result.IngressRequests, parseNginxRequest(labels, value))

		case prefixNginxLatencyBucket:
			result.IngressLatencyBuckets = append(result.IngressLatencyBuckets, parseNginxLatencyBucket(labels, value))

		case prefixNginxLatencySum:
			result.IngressLatencySums = append(result.IngressLatencySums, parseNginxLatencySum(labels, value))

		case prefixNginxLatencyCount:
			result.IngressLatencyCounts = append(result.IngressLatencyCounts, parseNginxLatencyCount(labels, value))

			// 其他指标丢弃
		}
	}

	return result, scanner.Err()
}

// =============================================================================
// 标签解析
// =============================================================================

// parseLabels 解析 key="value",key="value" 格式的标签字符串
func parseLabels(s string) map[string]string {
	result := make(map[string]string)
	// 状态机解析，处理逗号分隔的 key="value" 对
	for len(s) > 0 {
		// 找 =
		eqIdx := strings.IndexByte(s, '=')
		if eqIdx < 0 {
			break
		}
		key := s[:eqIdx]
		s = s[eqIdx+1:]

		// 值必须以 " 开头
		if len(s) == 0 || s[0] != '"' {
			break
		}
		s = s[1:]

		// 找结束的 "（处理转义）
		var val strings.Builder
		escaped := false
		i := 0
		for i < len(s) {
			if escaped {
				val.WriteByte(s[i])
				escaped = false
				i++
				continue
			}
			if s[i] == '\\' {
				escaped = true
				i++
				continue
			}
			if s[i] == '"' {
				break
			}
			val.WriteByte(s[i])
			i++
		}

		result[key] = val.String()

		// 跳过结束的 " 和可能的逗号
		if i < len(s) {
			s = s[i+1:] // skip closing "
		} else {
			break
		}
		s = strings.TrimLeft(s, ",")
	}
	return result
}

// =============================================================================
// Linkerd 解析函数
// =============================================================================

func parseLinkerdResponse(labels map[string]string, value float64) sdk.LinkerdResponseMetric {
	return sdk.LinkerdResponseMetric{
		Namespace:      labels["namespace"],
		Deployment:     labels["deployment"],
		Pod:            labels["pod"],
		Direction:      labels["direction"],
		StatusCode:     labels["status_code"],
		Classification: labels["classification"],
		RouteName:      labels["route_name"],
		SrvPort:        labels["srv_port"],
		DstNamespace:   labels["dst_namespace"],
		DstDeployment:  labels["dst_deployment"],
		TLS:            labels["tls"],
		Value:          value,
	}
}

func parseLinkerdLatencyBucket(labels map[string]string, value float64) sdk.LinkerdLatencyBucketMetric {
	return sdk.LinkerdLatencyBucketMetric{
		Namespace:  labels["namespace"],
		Deployment: labels["deployment"],
		Pod:        labels["pod"],
		Direction:  labels["direction"],
		Le:         labels["le"],
		Value:      value,
	}
}

func parseLinkerdLatencySum(labels map[string]string, value float64) sdk.LinkerdLatencySumMetric {
	return sdk.LinkerdLatencySumMetric{
		Namespace:     labels["namespace"],
		Deployment:    labels["deployment"],
		Pod:           labels["pod"],
		Direction:     labels["direction"],
		DstNamespace:  labels["dst_namespace"],
		DstDeployment: labels["dst_deployment"],
		Value:         value,
	}
}

func parseLinkerdLatencyCount(labels map[string]string, value float64) sdk.LinkerdLatencyCountMetric {
	return sdk.LinkerdLatencyCountMetric{
		Namespace:     labels["namespace"],
		Deployment:    labels["deployment"],
		Pod:           labels["pod"],
		Direction:     labels["direction"],
		DstNamespace:  labels["dst_namespace"],
		DstDeployment: labels["dst_deployment"],
		Value:         value,
	}
}

// =============================================================================
// Traefik 归一化函数
// =============================================================================

// normalizeTraefikServiceKey 归一化 Traefik service 标识
// "atlantis-atlantis-web-3000@kubernetes" → "atlantis-atlantis-web-3000"
func normalizeTraefikServiceKey(service string) string {
	if idx := strings.IndexByte(service, '@'); idx >= 0 {
		return service[:idx]
	}
	return service
}

func parseTraefikRequest(labels map[string]string, value float64) sdk.IngressRequestMetric {
	return sdk.IngressRequestMetric{
		ServiceKey: normalizeTraefikServiceKey(labels["service"]),
		Code:       labels["code"],
		Method:     labels["method"],
		Value:      value,
	}
}

func parseTraefikLatencyBucket(labels map[string]string, value float64) sdk.IngressLatencyBucketMetric {
	return sdk.IngressLatencyBucketMetric{
		ServiceKey: normalizeTraefikServiceKey(labels["service"]),
		Le:         labels["le"],
		Value:      value,
	}
}

func parseTraefikLatencySum(labels map[string]string, value float64) sdk.IngressLatencySumMetric {
	return sdk.IngressLatencySumMetric{
		ServiceKey: normalizeTraefikServiceKey(labels["service"]),
		Value:      value,
	}
}

func parseTraefikLatencyCount(labels map[string]string, value float64) sdk.IngressLatencyCountMetric {
	return sdk.IngressLatencyCountMetric{
		ServiceKey: normalizeTraefikServiceKey(labels["service"]),
		Value:      value,
	}
}

// =============================================================================
// Nginx 归一化函数
// =============================================================================

// normalizeNginxServiceKey 归一化 Nginx service 标识
// namespace="atlantis", service="atlantis-web", service_port="3000"
// → "atlantis-atlantis-web-3000"
func normalizeNginxServiceKey(labels map[string]string) string {
	ns := labels["namespace"]
	svc := labels["service"]
	port := labels["service_port"]
	if ns == "" || svc == "" || port == "" {
		return ""
	}
	return ns + "-" + svc + "-" + port
}

func parseNginxRequest(labels map[string]string, value float64) sdk.IngressRequestMetric {
	return sdk.IngressRequestMetric{
		ServiceKey: normalizeNginxServiceKey(labels),
		Code:       labels["status"],
		Method:     labels["method"],
		Value:      value,
	}
}

func parseNginxLatencyBucket(labels map[string]string, value float64) sdk.IngressLatencyBucketMetric {
	return sdk.IngressLatencyBucketMetric{
		ServiceKey: normalizeNginxServiceKey(labels),
		Le:         labels["le"],
		Value:      value,
	}
}

func parseNginxLatencySum(labels map[string]string, value float64) sdk.IngressLatencySumMetric {
	return sdk.IngressLatencySumMetric{
		ServiceKey: normalizeNginxServiceKey(labels),
		Value:      value,
	}
}

func parseNginxLatencyCount(labels map[string]string, value float64) sdk.IngressLatencyCountMetric {
	return sdk.IngressLatencyCountMetric{
		ServiceKey: normalizeNginxServiceKey(labels),
		Value:      value,
	}
}

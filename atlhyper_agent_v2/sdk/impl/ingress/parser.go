// Package ingress Ingress Controller 客户端实现
//
// parser.go - Prometheus 指标文本解析
//
// 解析 Prometheus 文本格式的指标，支持三种 Ingress Controller:
//   - Traefik: traefik_service_requests_total / traefik_service_request_duration_seconds_*
//   - Nginx: nginx_ingress_controller_requests / nginx_ingress_controller_request_duration_seconds_*
//   - Kong: kong_http_requests_total / kong_request_latency_ms_*
package ingress

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// 正则表达式
var (
	labelRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)
	valueRegex = regexp.MustCompile(`\}\s+(\d+\.?\d*)`)
)

// parseMetrics 解析 Prometheus 格式指标
func parseMetrics(reader io.Reader, ingressType string) (*sdk.IngressMetrics, error) {
	metrics := &sdk.IngressMetrics{
		Timestamp:  time.Now().Unix(),
		Counters:   make([]sdk.IngressCounterMetric, 0),
		Histograms: make([]sdk.IngressHistogramMetric, 0),
	}

	// 临时存储聚合数据
	counterRequests := make(map[string]int64)  // "host|status" -> value
	counterErrors := make(map[string]int64)    // "host|status" -> value
	histogramBuckets := make(map[string]map[string]int64) // host -> le -> count
	histogramSum := make(map[string]float64)
	histogramCount := make(map[string]int64)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		cleaned := cleanMetric(line, ingressType)
		if cleaned == nil {
			continue
		}

		switch cleaned.metricType {
		case "counter_requests":
			key := cleaned.host + "|" + cleaned.status
			counterRequests[key] += int64(cleaned.value)

		case "counter_errors":
			key := cleaned.host + "|" + cleaned.status
			counterErrors[key] += int64(cleaned.value)

		case "histogram_bucket":
			if histogramBuckets[cleaned.host] == nil {
				histogramBuckets[cleaned.host] = make(map[string]int64)
			}
			leKey := strconv.FormatFloat(cleaned.le, 'f', -1, 64)
			histogramBuckets[cleaned.host][leKey] += int64(cleaned.value)

		case "histogram_sum":
			histogramSum[cleaned.host] += cleaned.value

		case "histogram_count":
			histogramCount[cleaned.host] += int64(cleaned.value)
		}
	}

	// 组装 Counter
	for key, value := range counterRequests {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		metrics.Counters = append(metrics.Counters, sdk.IngressCounterMetric{
			Host:       parts[0],
			Status:     parts[1],
			MetricType: "requests",
			Value:      value,
		})
	}
	for key, value := range counterErrors {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		metrics.Counters = append(metrics.Counters, sdk.IngressCounterMetric{
			Host:       parts[0],
			Status:     parts[1],
			MetricType: "errors",
			Value:      value,
		})
	}

	// 组装 Histogram
	for host, buckets := range histogramBuckets {
		metrics.Histograms = append(metrics.Histograms, sdk.IngressHistogramMetric{
			Host:    host,
			Buckets: buckets,
			Sum:     histogramSum[host],
			Count:   histogramCount[host],
		})
	}

	return metrics, scanner.Err()
}

// =============================================================================
// 内部解析结构
// =============================================================================

// parsedMetric 解析后的单条指标
type parsedMetric struct {
	metricType string  // counter_requests / counter_errors / histogram_bucket / histogram_sum / histogram_count
	host       string
	status     string
	le         float64
	value      float64
}

// cleanMetric 根据 Ingress Controller 类型解析指标行
func cleanMetric(line string, ingressType string) *parsedMetric {
	if ingressType != "" {
		switch ingressType {
		case "nginx":
			return parseNginxMetric(line)
		case "traefik":
			return parseTraefikMetric(line)
		case "kong":
			return parseKongMetric(line)
		}
	}

	// 自动检测
	if strings.HasPrefix(line, "traefik_") {
		return parseTraefikMetric(line)
	}
	if strings.HasPrefix(line, "nginx_ingress_") {
		return parseNginxMetric(line)
	}
	if strings.HasPrefix(line, "kong_") {
		return parseKongMetric(line)
	}

	return nil
}

// =============================================================================
// Traefik 解析
// =============================================================================

func parseTraefikMetric(line string) *parsedMetric {
	if strings.HasPrefix(line, "traefik_service_requests_total{") {
		return parseTraefikCounter(line)
	}
	if strings.HasPrefix(line, "traefik_service_request_duration_seconds_bucket{") {
		return parseHistogramBucket(line)
	}
	if strings.HasPrefix(line, "traefik_service_request_duration_seconds_sum{") {
		return parseHistogramSum(line)
	}
	if strings.HasPrefix(line, "traefik_service_request_duration_seconds_count{") {
		return parseHistogramCount(line)
	}
	return nil
}

func parseTraefikCounter(line string) *parsedMetric {
	labels := extractLabels(line)
	host := labels["service"]
	if host == "" {
		return nil
	}
	value := extractValue(line)
	if value < 0 {
		return nil
	}
	return &parsedMetric{
		metricType: "counter_requests",
		host:       host,
		status:     labels["code"],
		value:      value,
	}
}

// =============================================================================
// Nginx 解析
// =============================================================================

func parseNginxMetric(line string) *parsedMetric {
	if strings.HasPrefix(line, "nginx_ingress_controller_requests{") {
		return parseCounter(line, "requests")
	}
	if strings.HasPrefix(line, "nginx_ingress_controller_request_duration_seconds_bucket{") {
		return parseHistogramBucket(line)
	}
	if strings.HasPrefix(line, "nginx_ingress_controller_request_duration_seconds_sum{") {
		return parseHistogramSum(line)
	}
	if strings.HasPrefix(line, "nginx_ingress_controller_request_duration_seconds_count{") {
		return parseHistogramCount(line)
	}
	return nil
}

// =============================================================================
// Kong 解析
// =============================================================================

func parseKongMetric(line string) *parsedMetric {
	if strings.HasPrefix(line, "kong_http_requests_total{") {
		return parseKongCounter(line)
	}
	if strings.HasPrefix(line, "kong_request_latency_ms_bucket{") {
		m := parseHistogramBucket(line)
		if m != nil {
			m.le = m.le / 1000.0 // Kong 使用毫秒，转换为秒
		}
		return m
	}
	if strings.HasPrefix(line, "kong_request_latency_ms_sum{") {
		return parseHistogramSum(line)
	}
	if strings.HasPrefix(line, "kong_request_latency_ms_count{") {
		return parseHistogramCount(line)
	}
	return nil
}

func parseKongCounter(line string) *parsedMetric {
	labels := extractLabels(line)
	host := labels["service"]
	if host == "" {
		return nil
	}
	value := extractValue(line)
	if value < 0 {
		return nil
	}
	return &parsedMetric{
		metricType: "counter_requests",
		host:       host,
		status:     labels["code"],
		value:      value,
	}
}

// =============================================================================
// 通用解析函数
// =============================================================================

func parseCounter(line, metricType string) *parsedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		return nil
	}
	value := extractValue(line)
	if value < 0 {
		return nil
	}
	return &parsedMetric{
		metricType: "counter_" + metricType,
		host:       host,
		status:     labels["status"],
		value:      value,
	}
}

func parseHistogramBucket(line string) *parsedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		host = labels["service"]
	}
	if host == "" {
		return nil
	}

	leStr := labels["le"]
	if leStr == "" || leStr == "+Inf" {
		return nil
	}
	le, err := strconv.ParseFloat(leStr, 64)
	if err != nil {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	return &parsedMetric{
		metricType: "histogram_bucket",
		host:       host,
		le:         le,
		value:      value,
	}
}

func parseHistogramSum(line string) *parsedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		host = labels["service"]
	}
	if host == "" {
		return nil
	}
	value := extractValue(line)
	if value < 0 {
		return nil
	}
	return &parsedMetric{
		metricType: "histogram_sum",
		host:       host,
		value:      value,
	}
}

func parseHistogramCount(line string) *parsedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		host = labels["service"]
	}
	if host == "" {
		return nil
	}
	value := extractValue(line)
	if value < 0 {
		return nil
	}
	return &parsedMetric{
		metricType: "histogram_count",
		host:       host,
		value:      value,
	}
}

// extractLabels 提取标签
func extractLabels(line string) map[string]string {
	result := make(map[string]string)
	matches := labelRegex.FindAllStringSubmatch(line, -1)
	for _, m := range matches {
		if len(m) == 3 {
			result[m[1]] = m[2]
		}
	}
	return result
}

// extractValue 提取值
func extractValue(line string) float64 {
	matches := valueRegex.FindStringSubmatch(line)
	if len(matches) < 2 {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			v, err := strconv.ParseFloat(parts[len(parts)-1], 64)
			if err == nil {
				return v
			}
		}
		return -1
	}
	v, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return -1
	}
	return v
}

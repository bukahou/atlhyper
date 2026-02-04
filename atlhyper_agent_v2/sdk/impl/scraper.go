// atlhyper_agent_v2/sdk/impl/scraper.go
// Prometheus 指标采集实现
package impl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var scraperLog = logger.Module("Scraper")

// metricsScraper Prometheus 指标采集器实现
type metricsScraper struct {
	client      *http.Client
	k8sClient   sdk.K8sClient
	ingressType string // nginx / traefik / kong
}

// NewMetricsScraper 创建指标采集器
// ingressType 会根据指标前缀自动检测，无需配置
func NewMetricsScraper(k8sClient sdk.K8sClient, timeout time.Duration) sdk.MetricsScraper {
	return &metricsScraper{
		client: &http.Client{
			Timeout: timeout,
		},
		k8sClient: k8sClient,
	}
}

// Scrape 从 URL 采集指标
func (s *metricsScraper) Scrape(ctx context.Context, url string) (*model.IngressMetrics, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return s.parseMetrics(resp.Body)
}

// DiscoverIngressURL 自动发现 Ingress Controller 的指标 URL
// 扫描所有命名空间，通过标签识别常见的 Ingress Controller
// 返回: url, ingressType (nginx/traefik/kong), error
func (s *metricsScraper) DiscoverIngressURL(ctx context.Context) (string, string, error) {
	// 列出所有命名空间的 Pod
	pods, err := s.k8sClient.ListPods(ctx, "", sdk.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("list pods: %w", err)
	}

	// 常见 Ingress Controller 的标签 -> (端口, 类型名)
	type ingressInfo struct {
		port        string
		ingressType string
	}
	ingressLabels := map[string]ingressInfo{
		"ingress-nginx": {"10254", "nginx"},  // nginx-ingress 默认指标端口
		"traefik":       {"9100", "traefik"}, // traefik 默认指标端口
		"kong":          {"8100", "kong"},    // kong 默认指标端口
	}

	for _, pod := range pods {
		// 跳过非 Running 状态的 Pod
		if pod.Status.Phase != "Running" {
			continue
		}

		// 检查是否是 Ingress Controller（通过标签）
		labels := pod.Labels
		if labels == nil {
			continue
		}

		appName := labels["app.kubernetes.io/name"]
		if appName == "" {
			appName = labels["app"]
		}

		info, isIngress := ingressLabels[appName]
		if !isIngress {
			continue
		}

		// 获取指标端口和路径
		port := info.port
		path := "/metrics"

		// 优先使用 prometheus.io 注解
		if pod.Annotations != nil {
			if p := pod.Annotations["prometheus.io/port"]; p != "" {
				port = p
			}
			if p := pod.Annotations["prometheus.io/path"]; p != "" {
				path = p
			}
		}

		// 使用 Pod IP 构建 URL
		if pod.Status.PodIP != "" {
			url := fmt.Sprintf("http://%s:%s%s", pod.Status.PodIP, port, path)
			scraperLog.Debug("发现 Ingress Controller",
				"type", info.ingressType,
				"pod", pod.Name,
				"namespace", pod.Namespace,
				"url", url,
			)
			return url, info.ingressType, nil
		}
	}

	return "", "", fmt.Errorf("no ingress controller found (supported: nginx-ingress, traefik, kong)")
}

// SetIngressType 设置 Ingress 类型
func (s *metricsScraper) SetIngressType(ingressType string) {
	s.ingressType = ingressType
}

// parseMetrics 解析 Prometheus 格式的指标
func (s *metricsScraper) parseMetrics(reader io.Reader) (*model.IngressMetrics, error) {
	metrics := &model.IngressMetrics{
		Timestamp:  time.Now().Unix(),
		Counters:   make([]model.IngressCounterMetric, 0),
		Histograms: make([]model.IngressHistogramMetric, 0),
	}

	// 临时存储 counter 数据，聚合同一个 (host, status) 的所有值
	// key: "host|status", value: 累加值
	counterRequests := make(map[string]int64)
	counterErrors := make(map[string]int64)

	// 临时存储 histogram 数据
	// 使用 string 作为 le 的键，因为 JSON 不支持 float64 作为 map key
	histogramBuckets := make(map[string]map[string]int64) // host -> le(string) -> count
	histogramSum := make(map[string]float64)              // host -> sum
	histogramCount := make(map[string]int64)              // host -> count

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过注释和空行
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// 根据 Ingress Controller 类型清理指标名
		cleaned := s.cleanMetrics(line)
		if cleaned == nil {
			continue
		}

		switch cleaned.Type {
		case "counter_requests":
			// 聚合同一个 (host, status) 的所有 method/protocol 变体
			key := cleaned.Host + "|" + cleaned.Status
			counterRequests[key] += int64(cleaned.Value)

		case "counter_errors":
			// 聚合同一个 (host, status) 的所有 method/protocol 变体
			key := cleaned.Host + "|" + cleaned.Status
			counterErrors[key] += int64(cleaned.Value)

		case "histogram_bucket":
			if histogramBuckets[cleaned.Host] == nil {
				histogramBuckets[cleaned.Host] = make(map[string]int64)
			}
			// 将 float64 的 le 转换为字符串
			// 使用 += 累加所有状态码的 bucket 值
			leKey := strconv.FormatFloat(cleaned.Le, 'f', -1, 64)
			histogramBuckets[cleaned.Host][leKey] += int64(cleaned.Value)

		case "histogram_sum":
			histogramSum[cleaned.Host] += cleaned.Value

		case "histogram_count":
			histogramCount[cleaned.Host] += int64(cleaned.Value)
		}
	}

	// 组装 counter（聚合后的数据）
	for key, value := range counterRequests {
		parts := strings.SplitN(key, "|", 2)
		if len(parts) != 2 {
			continue
		}
		metrics.Counters = append(metrics.Counters, model.IngressCounterMetric{
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
		metrics.Counters = append(metrics.Counters, model.IngressCounterMetric{
			Host:       parts[0],
			Status:     parts[1],
			MetricType: "errors",
			Value:      value,
		})
	}

	// 组装 histogram
	for host, buckets := range histogramBuckets {
		metrics.Histograms = append(metrics.Histograms, model.IngressHistogramMetric{
			Host:    host,
			Buckets: buckets,
			Sum:     histogramSum[host],
			Count:   histogramCount[host],
		})
	}

	return metrics, scanner.Err()
}

// cleanedMetric 清理后的指标
type cleanedMetric struct {
	Type   string  // counter_requests / counter_errors / histogram_bucket / histogram_sum / histogram_count
	Host   string  // 域名
	Status string  // HTTP 状态码（Counter 用）
	Le     float64 // histogram bucket 上边界
	Value  float64 // 值
}

// cleanMetrics 根据 Ingress Controller 类型清理指标行
// 如果 ingressType 为空，自动检测指标前缀来判断类型
func (s *metricsScraper) cleanMetrics(line string) *cleanedMetric {
	// 如果配置了类型，使用配置的类型
	if s.ingressType != "" {
		switch s.ingressType {
		case "nginx":
			return s.parseNginxMetric(line)
		case "traefik":
			return s.parseTraefikMetric(line)
		case "kong":
			return s.parseKongMetric(line)
		}
	}

	// 自动检测：通过指标前缀判断类型
	if strings.HasPrefix(line, "traefik_") {
		return s.parseTraefikMetric(line)
	}
	if strings.HasPrefix(line, "nginx_ingress_") {
		return s.parseNginxMetric(line)
	}
	if strings.HasPrefix(line, "kong_") {
		return s.parseKongMetric(line)
	}

	return nil
}

// parseNginxMetric 解析 nginx-ingress 指标
// 格式示例:
// nginx_ingress_controller_requests{host="example.com",status="200"} 12345
// nginx_ingress_controller_request_duration_seconds_bucket{host="example.com",le="0.5"} 100
func (s *metricsScraper) parseNginxMetric(line string) *cleanedMetric {
	// 请求计数
	if strings.HasPrefix(line, "nginx_ingress_controller_requests{") {
		return s.parseCounterLine(line, "requests")
	}

	// 延迟 histogram
	if strings.HasPrefix(line, "nginx_ingress_controller_request_duration_seconds_bucket{") {
		return s.parseHistogramBucketLine(line)
	}
	if strings.HasPrefix(line, "nginx_ingress_controller_request_duration_seconds_sum{") {
		return s.parseHistogramSumLine(line)
	}
	if strings.HasPrefix(line, "nginx_ingress_controller_request_duration_seconds_count{") {
		return s.parseHistogramCountLine(line)
	}

	return nil
}

// parseTraefikMetric 解析 Traefik 指标
// 只采集 service 级别指标，过滤 entrypoint 级别指标
// 格式示例:
// traefik_service_requests_total{service="ns-svc-port@kubernetes",code="200"} 12345
// traefik_service_request_duration_seconds_bucket{service="ns-svc-port@kubernetes",le="0.5"} 100
func (s *metricsScraper) parseTraefikMetric(line string) *cleanedMetric {
	// 只采集 service 级别指标
	// 过滤 entrypoint 级别指标 (traefik_entrypoint_*)

	// 请求计数
	if strings.HasPrefix(line, "traefik_service_requests_total{") {
		return s.parseTraefikCounterLine(line)
	}

	// 延迟 histogram
	if strings.HasPrefix(line, "traefik_service_request_duration_seconds_bucket{") {
		return s.parseHistogramBucketLine(line)
	}
	if strings.HasPrefix(line, "traefik_service_request_duration_seconds_sum{") {
		return s.parseHistogramSumLine(line)
	}
	if strings.HasPrefix(line, "traefik_service_request_duration_seconds_count{") {
		return s.parseHistogramCountLine(line)
	}

	return nil
}

// parseKongMetric 解析 Kong 指标
// 格式示例:
// kong_http_requests_total{service="example",code="200"} 12345
// kong_request_latency_ms_bucket{service="example",le="100"} 100
func (s *metricsScraper) parseKongMetric(line string) *cleanedMetric {
	// 请求计数
	if strings.HasPrefix(line, "kong_http_requests_total{") {
		return s.parseKongCounterLine(line)
	}

	// 延迟 histogram (Kong 使用毫秒)
	if strings.HasPrefix(line, "kong_request_latency_ms_bucket{") {
		return s.parseKongHistogramBucketLine(line)
	}
	if strings.HasPrefix(line, "kong_request_latency_ms_sum{") {
		return s.parseHistogramSumLine(line)
	}
	if strings.HasPrefix(line, "kong_request_latency_ms_count{") {
		return s.parseHistogramCountLine(line)
	}

	return nil
}

// 正则表达式
var (
	labelRegex = regexp.MustCompile(`(\w+)="([^"]*)"`)
	valueRegex = regexp.MustCompile(`\}\s+(\d+\.?\d*)`)
)

// parseCounterLine 解析 Counter 行
func (s *metricsScraper) parseCounterLine(line, metricType string) *cleanedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	status := labels["status"]
	return &cleanedMetric{
		Type:   "counter_" + metricType,
		Host:   host,
		Status: status,
		Value:  value,
	}
}

// parseTraefikCounterLine 解析 Traefik Counter 行
func (s *metricsScraper) parseTraefikCounterLine(line string) *cleanedMetric {
	labels := extractLabels(line)

	// Traefik 使用 service 作为标识
	host := labels["service"]
	if host == "" {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	status := labels["code"]
	return &cleanedMetric{
		Type:   "counter_requests",
		Host:   host,
		Status: status,
		Value:  value,
	}
}

// parseKongCounterLine 解析 Kong Counter 行
func (s *metricsScraper) parseKongCounterLine(line string) *cleanedMetric {
	labels := extractLabels(line)

	host := labels["service"]
	if host == "" {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	status := labels["code"]
	return &cleanedMetric{
		Type:   "counter_requests",
		Host:   host,
		Status: status,
		Value:  value,
	}
}

// parseHistogramBucketLine 解析 Histogram Bucket 行
func (s *metricsScraper) parseHistogramBucketLine(line string) *cleanedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		host = labels["service"] // Traefik service metrics
	}
	if host == "" {
		return nil
	}

	leStr := labels["le"]
	if leStr == "" || leStr == "+Inf" {
		return nil // 跳过 +Inf
	}

	le, err := strconv.ParseFloat(leStr, 64)
	if err != nil {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	return &cleanedMetric{
		Type:  "histogram_bucket",
		Host:  host,
		Le:    le,
		Value: value,
	}
}

// parseKongHistogramBucketLine 解析 Kong Histogram Bucket 行（毫秒转秒）
func (s *metricsScraper) parseKongHistogramBucketLine(line string) *cleanedMetric {
	result := s.parseHistogramBucketLine(line)
	if result != nil {
		// Kong 使用毫秒，转换为秒
		result.Le = result.Le / 1000.0
	}
	return result
}

// parseHistogramSumLine 解析 Histogram Sum 行
func (s *metricsScraper) parseHistogramSumLine(line string) *cleanedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		host = labels["service"] // Traefik service metrics
	}
	if host == "" {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	return &cleanedMetric{
		Type:  "histogram_sum",
		Host:  host,
		Value: value,
	}
}

// parseHistogramCountLine 解析 Histogram Count 行
func (s *metricsScraper) parseHistogramCountLine(line string) *cleanedMetric {
	labels := extractLabels(line)
	host := labels["host"]
	if host == "" {
		host = labels["service"] // Traefik service metrics
	}
	if host == "" {
		return nil
	}

	value := extractValue(line)
	if value < 0 {
		return nil
	}

	return &cleanedMetric{
		Type:  "histogram_count",
		Host:  host,
		Value: value,
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
		// 尝试另一种格式：空格分隔
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

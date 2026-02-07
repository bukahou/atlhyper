// Package otel OTel Collector 采集客户端实现
//
// client.go - OTelClient 接口实现
//
// 从 OTel Collector 的 Prometheus 端点采集原始指标。
// 只做 HTTP 采集和文本解析，不做业务过滤/聚合。
//
// 架构位置:
//
//	SLORepository
//	    ↓ 调用
//	OTelClient (本包)
//	    ↓ 使用
//	net/http
//	    ↓
//	OTel Collector (:8889/metrics)
package otel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var log = logger.Module("OTelClient")

// client OTelClient 实现
type client struct {
	metricsURL string       // http://otel-collector.otel.svc:8889/metrics
	healthURL  string       // http://otel-collector.otel.svc:13133
	httpClient *http.Client
}

// NewOTelClient 创建 OTelClient
func NewOTelClient(metricsURL, healthURL string, timeout time.Duration) sdk.OTelClient {
	return &client{
		metricsURL: metricsURL,
		healthURL:  healthURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ScrapeMetrics 从 OTel Collector 采集原始指标
//
// HTTP GET → 解析 Prometheus 文本 → 分类为 OTelRawMetrics
// 返回 per-pod 级别的累积值，不做 delta 计算
func (c *client) ScrapeMetrics(ctx context.Context) (*sdk.OTelRawMetrics, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.metricsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("scrape otel collector: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("otel collector returned status %d", resp.StatusCode)
	}

	metrics, err := parsePrometheus(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse prometheus metrics: %w", err)
	}

	log.Debug("scraped OTel metrics: linkerd_responses=%d, linkerd_latency_buckets=%d, ingress_requests=%d",
		len(metrics.LinkerdResponses),
		len(metrics.LinkerdLatencyBuckets),
		len(metrics.IngressRequests),
	)

	return metrics, nil
}

// IsHealthy 检查 OTel Collector 健康状态
func (c *client) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", c.healthURL, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

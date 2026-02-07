// Package ingress Ingress Controller 客户端实现
//
// client.go - IngressClient 接口实现入口
//
// 本文件实现 sdk.IngressClient 接口，整合:
//   - 指标采集: HTTP 采集 Prometheus 格式指标
//   - 自动发现: 扫描 Pod 标签识别 Ingress Controller
//   - 路由采集: 采集 Traefik IngressRoute CRD
//
// 架构位置:
//
//	SLORepository
//	    ↓ 调用
//	IngressClient (本包)
//	    ↓ 使用
//	net/http → Ingress Controller (:9100/metrics)
//	K8s Dynamic API → IngressRoute CRD
package ingress

import (
	"context"
	"net/http"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/common/logger"
)

var log = logger.Module("IngressClient")

// client IngressClient 实现
type client struct {
	httpClient  *http.Client
	k8sClient   sdk.K8sClient
	ingressType string // nginx / traefik / kong
}

// NewClient 创建 IngressClient
func NewClient(k8sClient sdk.K8sClient, timeout time.Duration) sdk.IngressClient {
	return &client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		k8sClient: k8sClient,
	}
}

// ScrapeMetrics 从 Ingress Controller 采集 Prometheus 指标
func (c *client) ScrapeMetrics(ctx context.Context, url string) (*sdk.IngressMetrics, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode}
	}

	return parseMetrics(resp.Body, c.ingressType)
}

// SetIngressType 设置 Ingress 类型
func (c *client) SetIngressType(ingressType string) {
	c.ingressType = ingressType
}

// HTTPError HTTP 请求错误
type HTTPError struct {
	StatusCode int
}

func (e *HTTPError) Error() string {
	return "unexpected status: " + http.StatusText(e.StatusCode)
}

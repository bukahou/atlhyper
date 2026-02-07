// Package ingress Ingress Controller 客户端实现
//
// discover.go - 自动发现 Ingress Controller
//
// 扫描所有命名空间的 Pod，通过标签识别常见的 Ingress Controller:
//   - Traefik: app.kubernetes.io/name=traefik, 端口 9100
//   - Nginx: app.kubernetes.io/name=ingress-nginx, 端口 10254
//   - Kong: app.kubernetes.io/name=kong, 端口 8100
package ingress

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_agent_v2/sdk"
)

// ingressInfo 已知 Ingress Controller 信息
type ingressInfo struct {
	port        string
	ingressType string
}

// 已知 Ingress Controller 标签映射
var knownIngress = map[string]ingressInfo{
	"ingress-nginx": {"10254", "nginx"},
	"traefik":       {"9100", "traefik"},
	"kong":          {"8100", "kong"},
}

// DiscoverURL 自动发现 Ingress Controller 的指标端点
func (c *client) DiscoverURL(ctx context.Context) (string, string, error) {
	pods, err := c.k8sClient.ListPods(ctx, "", sdk.ListOptions{})
	if err != nil {
		return "", "", fmt.Errorf("list pods: %w", err)
	}

	for _, pod := range pods {
		if pod.Status.Phase != "Running" {
			continue
		}

		labels := pod.Labels
		if labels == nil {
			continue
		}

		appName := labels["app.kubernetes.io/name"]
		if appName == "" {
			appName = labels["app"]
		}

		info, isIngress := knownIngress[appName]
		if !isIngress {
			continue
		}

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

		if pod.Status.PodIP != "" {
			url := fmt.Sprintf("http://%s:%s%s", pod.Status.PodIP, port, path)
			log.Debug("发现 Ingress Controller",
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

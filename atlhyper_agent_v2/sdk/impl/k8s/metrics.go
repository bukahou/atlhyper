// Package k8s K8sClient 接口的具体实现
//
// metrics.go - metrics 资源操作
//
// 本文件实现 metrics API 的资源操作：
//   - NodeMetrics: List
//
// 注意: 需要集群安装 metrics-server，未安装时返回空数据
package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent_v2/sdk"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 使用 client.go 中声明的 log 变量

// =============================================================================
// Node Metrics 操作
// =============================================================================

// ListNodeMetrics 获取所有 Node 的资源使用量
//
// 返回 map[nodeName]NodeMetrics，key 为节点名称。
// 如果 metrics-server 未安装或不可用，返回空 map (不报错)。
func (c *Client) ListNodeMetrics(ctx context.Context) (map[string]sdk.NodeMetrics, error) {
	result := make(map[string]sdk.NodeMetrics)

	// metrics client 未初始化，返回空 map
	if c.metricsClient == nil {
		return result, nil
	}

	// 获取所有节点的 metrics
	metricsList, err := c.metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		// metrics-server 不可用，返回空 map (不报错)
		log.Warn("获取节点 metrics 失败", "err", err)
		return result, nil
	}

	// 转换为 map
	for _, m := range metricsList.Items {
		result[m.Name] = sdk.NodeMetrics{
			CPU:    m.Usage.Cpu().String(),
			Memory: m.Usage.Memory().String(),
		}
	}

	return result, nil
}

// =============================================================================
// Pod Metrics 操作
// =============================================================================

// ListPodMetrics 获取所有 Pod 的资源使用量
//
// 返回 map[namespace/name]PodMetrics，key 为 "namespace/name" 格式。
// 如果 metrics-server 未安装或不可用，返回空 map (不报错)。
func (c *Client) ListPodMetrics(ctx context.Context) (map[string]sdk.PodMetrics, error) {
	result := make(map[string]sdk.PodMetrics)

	// metrics client 未初始化，返回空 map
	if c.metricsClient == nil {
		return result, nil
	}

	// 获取所有 Pod 的 metrics
	metricsList, err := c.metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		// metrics-server 不可用，返回空 map (不报错)
		log.Warn("获取 Pod metrics 失败", "err", err)
		return result, nil
	}

	// 转换为 map
	for _, m := range metricsList.Items {
		key := m.Namespace + "/" + m.Name
		pm := sdk.PodMetrics{
			Namespace:  m.Namespace,
			Name:       m.Name,
			Containers: make([]sdk.ContainerMetrics, 0, len(m.Containers)),
		}
		for _, c := range m.Containers {
			pm.Containers = append(pm.Containers, sdk.ContainerMetrics{
				Name:   c.Name,
				CPU:    c.Usage.Cpu().String(),
				Memory: c.Usage.Memory().String(),
			})
		}
		result[key] = pm
	}

	return result, nil
}

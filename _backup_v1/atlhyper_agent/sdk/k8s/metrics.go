// sdk/k8s/metrics.go
// MetricsProvider K8s 实现
package k8s

import (
	"context"

	"AtlHyper/atlhyper_agent/sdk"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type k8sMetricsProvider struct {
	client *metricsclient.Clientset
}

func newK8sMetricsProvider(client *metricsclient.Clientset) *k8sMetricsProvider {
	return &k8sMetricsProvider{client: client}
}

func (m *k8sMetricsProvider) IsAvailable() bool {
	return m.client != nil
}

func (m *k8sMetricsProvider) GetPodMetrics(ctx context.Context, namespace string) (map[string]sdk.PodMetrics, error) {
	if m.client == nil {
		return nil, nil
	}

	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}

	metricsList, err := m.client.MetricsV1beta1().PodMetricses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make(map[string]sdk.PodMetrics, len(metricsList.Items))
	for _, pm := range metricsList.Items {
		key := pm.Namespace + "/" + pm.Name
		metrics := sdk.PodMetrics{
			Namespace: pm.Namespace,
			Name:      pm.Name,
		}

		// 聚合所有容器的指标
		for _, container := range pm.Containers {
			if cpu := container.Usage.Cpu(); cpu != nil {
				metrics.CPUUsage += cpu.MilliValue()
			}
			if mem := container.Usage.Memory(); mem != nil {
				metrics.MemoryUsage += mem.Value()
			}
		}

		result[key] = metrics
	}

	return result, nil
}

func (m *k8sMetricsProvider) GetNodeMetrics(ctx context.Context) (map[string]sdk.NodeMetrics, error) {
	if m.client == nil {
		return nil, nil
	}

	metricsList, err := m.client.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make(map[string]sdk.NodeMetrics, len(metricsList.Items))
	for _, nm := range metricsList.Items {
		metrics := sdk.NodeMetrics{
			Name: nm.Name,
		}

		if cpu := nm.Usage.Cpu(); cpu != nil {
			metrics.CPUUsage = cpu.MilliValue()
		}
		if mem := nm.Usage.Memory(); mem != nil {
			metrics.MemoryUsage = mem.Value()
		}

		result[nm.Name] = metrics
	}

	return result, nil
}

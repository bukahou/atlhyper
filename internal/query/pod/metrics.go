// =======================================================================================
// 📄 metrics.go
//
// ✨ 文件功能说明：
//     获取所有命名空间下 Pod 的 CPU 与内存使用量（用于聚合或热点分析）
//
// 📦 外部依赖：
//     - metrics.k8s.io/v1beta1
//     - internal/utils.GetMetricsClient()
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package pod

import (
	"NeuroController/internal/utils"
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodUsage 表示单个 Pod 的资源使用情况
type PodUsage struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CPUUsage  int64  `json:"cpu_usage_millicores"` // mCPU
	MemUsage  int64  `json:"mem_usage_bytes"`      // Bytes
}

// ListAllPodUsages 获取所有 Pod 的 CPU 与内存使用量（聚合用）
func ListAllPodUsages(ctx context.Context) ([]PodUsage, error) {
	client := utils.GetMetricsClient()

	podMetricsList, err := client.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取 PodMetrics: %w", err)
	}

	var usages []PodUsage
	for _, pm := range podMetricsList.Items {
		var totalCPU int64
		var totalMem int64
		for _, c := range pm.Containers {
			cpu := c.Usage.Cpu().MilliValue()
			mem := c.Usage.Memory().Value()
			totalCPU += cpu
			totalMem += mem
		}

		usages = append(usages, PodUsage{
			Name:      pm.Name,
			Namespace: pm.Namespace,
			CPUUsage:  totalCPU,
			MemUsage:  totalMem,
		})
	}

	return usages, nil
}

// =======================================================================================
// 📄 internal/query/node/metrics.go
//
// ✨ 文件功能说明：
//     获取节点的资源使用率（CPU、内存），需依赖 metrics-server。
//     - 节点平均 CPU 使用率（%）
//     - 节点平均内存使用率（%）
//     - DiskPressure 节点统计
//
// 🧪 示例输出：
//     - 平均 CPU 使用率: 45.1%
//     - 平均内存使用率: 62.3%
//     - 有压力节点数: 1
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package node

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

// NodeMetricsSummary 节点资源统计结果
type NodeMetricsSummary struct {
	AvgCPUUsagePercent    float64 // 所有节点平均 CPU 使用率（%）
	AvgMemoryUsagePercent float64 // 所有节点平均内存使用率（%）
	DiskPressureCount     int     // 具有 DiskPressure 的节点数量
}

// GetNodeMetricsSummary 汇总所有节点的平均资源使用率
func GetNodeMetricsSummary(ctx context.Context) (*NodeMetricsSummary, error) {
	coreClient := utils.GetCoreClient()
	metricsClient := utils.GetMetricsClient()

	nodeList, err := coreClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node 列表失败: %w", err)
	}

	metricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node metrics 失败（可能未部署 metrics-server）: %w", err)
	}

	// 创建映射：NodeName -> metrics
	metricsMap := make(map[string]metricsapi.NodeMetrics)
	for _, m := range metricsList.Items {
		metricsMap[m.Name] = m
	}

	var (
		totalCPUPercent   float64
		totalMemPercent   float64
		diskPressureCount int
		metricsNodeCount  int
	)

	for _, node := range nodeList.Items {
		metrics, ok := metricsMap[node.Name]
		if !ok {
			continue // 跳过未找到 metrics 的节点
		}

		allocatableCPU := node.Status.Allocatable.Cpu().MilliValue()
		allocatableMem := node.Status.Allocatable.Memory().Value()

		usageCPU := metrics.Usage.Cpu().MilliValue()
		usageMem := metrics.Usage.Memory().Value()

		if allocatableCPU > 0 {
			totalCPUPercent += float64(usageCPU) / float64(allocatableCPU) * 100
		}
		if allocatableMem > 0 {
			totalMemPercent += float64(usageMem) / float64(allocatableMem) * 100
		}
		metricsNodeCount++

		for _, cond := range node.Status.Conditions {
			if cond.Type == "DiskPressure" && cond.Status == "True" {
				diskPressureCount++
				break
			}
		}
	}

	if metricsNodeCount == 0 {
		return nil, fmt.Errorf("未获取到任何节点的 metrics")
	}

	return &NodeMetricsSummary{
		AvgCPUUsagePercent:    totalCPUPercent / float64(metricsNodeCount),
		AvgMemoryUsagePercent: totalMemPercent / float64(metricsNodeCount),
		DiskPressureCount:     diskPressureCount,
	}, nil
}

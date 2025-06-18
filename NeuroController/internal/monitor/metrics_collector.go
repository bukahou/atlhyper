// =======================================================================================
// 📄 internal/monitor/metrics_collector.go
//
// ✨ Description:
//     收集集群中各个 Node 的资源使用情况（CPU 和内存），
//     用于轻量级告警格式中补充实时节点状态信息，提升诊断上下文。
//
// 📊 提供函数：
//     - GetNodeResourceUsage(): 返回 map[nodeName] => CPU 占用率 + 内存使用情况
//
// 🧑‍💻 Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package monitor

import (
	"context"
	"fmt"
	"log"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ✨ NodeResourceUsage 表示某个节点的资源使用情况（CPU 和内存）
type NodeResourceUsage struct {
	CPUUsage    string // 如 "28%"
	MemoryUsage string // 如 "3.1Gi / 8.0Gi"
}

// ✅ GetNodeResourceUsage 收集所有节点的 CPU 和内存使用率
// 返回 map[nodeName] => NodeResourceUsage，用于展示在告警中
func GetNodeResourceUsage() map[string]NodeResourceUsage {
	result := make(map[string]NodeResourceUsage) // 用于存放每个节点的指标信息

	if !utils.HasMetricsServer() {
		log.Println("⚠️ [GetNodeResourceUsage] metrics-server 未启用，跳过指标采集")
		return result
	}

	metricsClient := utils.GetMetricsClient()
	kubeClient := utils.GetCoreClient()

	nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("❌ [GetNodeResourceUsage] 获取 NodeMetrics 失败: %v", err)
		return result
	}

	nodeList, err := kubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Printf("❌ [GetNodeResourceUsage] 获取 Node 列表失败: %v", err)
		return result
	}

	nodeCapacities := make(map[string]corev1.ResourceList)
	for _, node := range nodeList.Items {
		nodeCapacities[node.Name] = node.Status.Capacity
	}

	for _, item := range nodeMetricsList.Items {
		cap, ok := nodeCapacities[item.Name]
		if !ok {
			log.Printf("⚠️ [GetNodeResourceUsage] 找不到节点容量信息: %s", item.Name)
			continue
		}

		usageCPU := item.Usage[corev1.ResourceCPU]
		usageMem := item.Usage[corev1.ResourceMemory]
		capCPU := cap[corev1.ResourceCPU]
		capMem := cap[corev1.ResourceMemory]

		cpuPercent := float64(usageCPU.MilliValue()) / float64(capCPU.MilliValue()) * 100
		memUsage := fmt.Sprintf("%.1fGi / %.1fGi",
			float64(usageMem.Value())/1e9,
			float64(capMem.Value())/1e9,
		)

		result[item.Name] = NodeResourceUsage{
			CPUUsage:    fmt.Sprintf("%.0f%%", cpuPercent),
			MemoryUsage: memUsage,
		}
	}

	log.Printf("✅ [GetNodeResourceUsage] 成功收集 %d 个节点的指标数据", len(result))
	return result
}

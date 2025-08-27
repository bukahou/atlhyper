// =======================================================================================
// 📄 list.go（internal/query/pod）
//
// ✨ 文件功能说明：
//     提供 Pod 基础列表查询能力，用于获取所有命名空间或指定命名空间下的 Pod。
//     通常用于后端聚合、页面展示、筛选或状态分析等场景。
//
// 🔍 提供的功能：
//     - 获取全集群所有 Pod（ListAllPods）
//     - 获取指定命名空间下 Pod（ListPodsByNamespace）
//
// 📦 外部依赖：
//     - utils.GetCoreClient()（封装的 client-go 客户端）
//     - k8s.io/api/core/v1
//
// 📌 示例调用：
//     pods, err := pod.ListAllPods(ctx)
//     nsPods, err := pod.ListPodsByNamespace(ctx, "kube-system")
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 创建时间：2025年7月
// =======================================================================================
// 📄 internal/query/pod/list.go

package pod

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"NeuroController/internal/utils"
	pod "NeuroController/model"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListAllPods 返回集群中所有命名空间的 Pod 列表
func ListAllPods(ctx context.Context) ([]corev1.Pod, error) {
	client := utils.GetCoreClient()
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取所有 Pod 失败: %w", err)
	}
	return pods.Items, nil
}

// ListAllPodInfos 返回所有命名空间下 Pod 的简略信息（用于 UI 展示）
// func ListAllPodInfos(ctx context.Context) ([]pod.PodInfo, error) {
// 	rawPods, err := ListAllPods(ctx)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var result []pod.PodInfo
// 	for _, pod := range rawPods {
// 		result = append(result, convertPodToInfo(&pod))
// 	}
// 	return result, nil
// }
func ListAllPodInfos(ctx context.Context) ([]pod.PodInfo, error) {
	rawPods, err := ListAllPods(ctx)
	if err != nil {
		return nil, err
	}

	// 构建 Pod → NodeName 映射
	podNodeMap := make(map[string]string)
	for _, p := range rawPods {
		key := p.Namespace + "/" + p.Name
		podNodeMap[key] = p.Spec.NodeName
	}

	// 获取 Pod Metrics
	metricsClient := utils.GetMetricsClient()
	podMetricsList, err := metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("⚠️ 获取 Pod metrics 失败: %v", err)
	}

	// 构建 map：namespace/name → usage
	type usage struct {
		cpu  int64
		mem  int64
		node string
	}
	usageMap := make(map[string]usage)
	for _, m := range podMetricsList.Items {
		var totalCPU, totalMem int64
		for _, c := range m.Containers {
			totalCPU += c.Usage.Cpu().MilliValue()
			totalMem += c.Usage.Memory().Value()
		}
		key := m.Namespace + "/" + m.Name
		node := podNodeMap[key]
		usageMap[key] = usage{
			cpu:  totalCPU,
			mem:  totalMem,
			node: node,
		}
	}

	// 获取 Node 容量
	nodeList, err := utils.GetCoreClient().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("⚠️ 获取 Node 列表失败: %v", err)
	}
	nodeCapMap := make(map[string]struct {
		cpu int64
		mem int64
	})
	for _, node := range nodeList.Items {
		nodeCapMap[node.Name] = struct {
			cpu int64
			mem int64
		}{
			cpu: node.Status.Capacity.Cpu().MilliValue(),
			mem: node.Status.Capacity.Memory().Value(),
		}
	}

	// 汇总结果
	var result []pod.PodInfo
	for _, p := range rawPods {
		info := convertPodToInfo(&p)
		key := p.Namespace + "/" + p.Name

		if u, ok := usageMap[key]; ok {
			info.CPUUsage = fmt.Sprintf("%dm", u.cpu)
			// info.MemoryUsage = resource.NewQuantity(u.mem, resource.BinarySI).String()
			memQty := resource.NewQuantity(u.mem, resource.BinarySI)
			memGB := float64(memQty.Value()) / (1024 * 1024 * 1024) // byte → GiB
			if memGB >= 1 {
				info.MemoryUsage = fmt.Sprintf("%.1f GiB", memGB)
			} else {
				memMB := float64(memQty.Value()) / (1024 * 1024) // byte → MiB
				info.MemoryUsage = fmt.Sprintf("%.0f MiB", memMB)
			}


			// if cap, ok := nodeCapMap[u.node]; ok {
			// 	if cap.cpu > 0 {
			// 		info.CPUUsagePercent = fmt.Sprintf("%.1f%%", float64(u.cpu)*100/float64(cap.cpu))
			// 	}
			// 	if cap.mem > 0 {
			// 		info.MemoryPercent = fmt.Sprintf("%.1f%%", float64(u.mem)*100/float64(cap.mem))
			// 	}
			// }
			// 使用 Pod 的 limit/request 作为上限
			var totalLimitCPU int64
			var totalLimitMem int64
			for _, c := range p.Spec.Containers {
				if c.Resources.Limits.Cpu() != nil {
					totalLimitCPU += c.Resources.Limits.Cpu().MilliValue()
				}
				if c.Resources.Limits.Memory() != nil {
					totalLimitMem += c.Resources.Limits.Memory().Value()
				}
			}
			if totalLimitCPU == 0 {
				for _, c := range p.Spec.Containers {
					if c.Resources.Requests.Cpu() != nil {
						totalLimitCPU += c.Resources.Requests.Cpu().MilliValue()
					}
				}
			}
			if totalLimitMem == 0 {
				for _, c := range p.Spec.Containers {
					if c.Resources.Requests.Memory() != nil {
						totalLimitMem += c.Resources.Requests.Memory().Value()
					}
				}
			}
			if totalLimitCPU > 0 {
				info.CPUUsagePercent = fmt.Sprintf("%.1f%%", float64(u.cpu)*100/float64(totalLimitCPU))
			}
			if totalLimitMem > 0 {
				info.MemoryPercent = fmt.Sprintf("%.1f%%", float64(u.mem)*100/float64(totalLimitMem))
			}

		}

		result = append(result, info)
	}
	return result, nil
}



// convertPodToInfo 将 corev1.Pod 转换为 PodInfo（精简结构体）
func convertPodToInfo(p *corev1.Pod) pod.PodInfo {
	deployment := "-"
	for _, owner := range p.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			name := owner.Name
			if idx := strings.LastIndex(name, "-"); idx > 0 {
				deployment = name[:idx]
			} else {
				deployment = name
			}
			break
		}
	}

	ready := false
	for _, cond := range p.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			ready = true
			break
		}
	}

	restartCount := int32(0)
	if len(p.Status.ContainerStatuses) > 0 {
		restartCount = p.Status.ContainerStatuses[0].RestartCount
	}

	startTime := ""
	if p.Status.StartTime != nil {
		startTime = p.Status.StartTime.Format(time.RFC3339)
	}

	return pod.PodInfo{
		Namespace:    p.Namespace,
		Deployment:   deployment,
		Name:         p.Name,
		Ready:        ready,
		Phase:        string(p.Status.Phase),
		RestartCount: restartCount,
		StartTime:    startTime,
		PodIP:        p.Status.PodIP,
		NodeName:     p.Spec.NodeName,
	}
}



// ListPodsByNamespace 返回指定命名空间下的 Pod 列表
func ListPodsByNamespace(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	client := utils.GetCoreClient()
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Pod 失败: %w", namespace, err)
	}
	return pods.Items, nil
}
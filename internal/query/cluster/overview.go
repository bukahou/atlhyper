// =======================================================================================
// 📄 overview.go（internal/query/cluster）
//
// ✨ 文件功能说明：
//
//	提供 Kubernetes 集群的整体概要信息，用于集群首页展示。
//	包括以下内容：
//	- 节点总数与 Ready 节点数
//	- 所有命名空间下的 Pod 总数
//	- 异常状态的 Pod 数量（Pending / Failed / Unknown）
//	- Kubernetes 控制平面版本
//	- 是否部署了 metrics-server（判断是否支持 CPU/Mem 查询）
//
// 📦 外部依赖：
//   - utils.GetCoreClient()（封装的 client-go 客户端）
//   - utils.HasMetricsServer()（检测 metrics-server 是否可用）
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 创建时间：2025年7月
// =======================================================================================
package cluster

import (
	"NeuroController/internal/utils"
	"NeuroController/model/clusteroverview"
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

func GetClusterOverview(ctx context.Context) (*clusteroverview.ClusterOverview, error) {
	client := utils.GetCoreClient()

	// 1) Nodes：总数、Ready 数、累计 allocatable（总量），同时准备 per-node 总量
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取 Node 列表: %w", err)
	}
	totalNodes := len(nodes.Items)
	readyNodes := 0

	var totalCPUMilli int64
	var totalMemBytes int64

	// 逐节点总量（allocatable）
	type nodeCap struct {
		cpuMilli int64
		memBytes int64
		ready    bool
		role     string
	}
	perNodeCap := make(map[string]nodeCap, totalNodes)

	for _, node := range nodes.Items {
		ready := false
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}
		if ready {
			readyNodes++
		}

		var cpuMilli int64
		var memBytes int64
		if cpuQty, ok := node.Status.Allocatable[corev1.ResourceCPU]; ok {
			cpuMilli = cpuQty.MilliValue()
			totalCPUMilli += cpuMilli
		}
		if memQty, ok := node.Status.Allocatable[corev1.ResourceMemory]; ok {
			memBytes = memQty.Value()
			totalMemBytes += memBytes
		}

		// 简单解析 role（可选）
		role := ""
		for k := range node.Labels {
			if strings.HasPrefix(k, "node-role.kubernetes.io/") {
				role = strings.TrimPrefix(k, "node-role.kubernetes.io/")
				if role == "" {
					role = "master"
				}
				break
			}
		}

		perNodeCap[node.Name] = nodeCap{
			cpuMilli: cpuMilli,
			memBytes: memBytes,
			ready:    ready,
			role:     role,
		}
	}

	// 2) Pods：总数、异常数
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取 Pod 列表: %w", err)
	}
	totalPods := len(pods.Items)
	abnormalPods := 0
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodFailed, corev1.PodPending, corev1.PodUnknown:
			abnormalPods++
		}
	}

	// 3) 版本
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("无法获取 Kubernetes 版本: %w", err)
	}

	ov := &clusteroverview.ClusterOverview{
		TotalNodes:   totalNodes,
		ReadyNodes:   readyNodes,
		TotalPods:    totalPods,
		AbnormalPods: abnormalPods,
		K8sVersion:   versionInfo.GitVersion,
		HasMetrics:   utils.HasMetricsServer(),
	}

	// 4) 若 metrics-server 可用：汇总总使用 + 逐节点使用
	if ov.HasMetrics {
		restCfg := utils.GetRestConfig()
		mc, err := metricsclient.NewForConfig(restCfg)
		if err == nil {
			nmList, err := mc.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
			if err == nil {
				var usedCPUMilli int64
				var usedMemBytes int64

				// 先把 per-node used 放进 map
				type nodeUsed struct {
					cpuMilli int64
					memBytes int64
				}
				perNodeUsed := make(map[string]nodeUsed, len(nmList.Items))
				for _, m := range nmList.Items {
					perNodeUsed[m.Name] = nodeUsed{
						cpuMilli: m.Usage.Cpu().MilliValue(),
						memBytes: m.Usage.Memory().Value(),
					}
				}

				// 汇总总使用
				for _, u := range perNodeUsed {
					usedCPUMilli += u.cpuMilli
					usedMemBytes += u.memBytes
				}

				// 总资源汇总（cluster level）
				res := &clusteroverview.ClusterResourceSummary{
					TotalCPUMilli:    totalCPUMilli,
					UsedCPUMilli:     usedCPUMilli,
					TotalCPUCores:    float64(totalCPUMilli) / 1000.0,
					UsedCPUCores:     float64(usedCPUMilli) / 1000.0,
					TotalMemoryBytes: totalMemBytes,
					UsedMemoryBytes:  usedMemBytes,
				}
				if totalCPUMilli > 0 {
					res.CPUPercent = (float64(usedCPUMilli) / float64(totalCPUMilli)) * 100.0
				}
				if totalMemBytes > 0 {
					res.MemoryPercent = (float64(usedMemBytes) / float64(totalMemBytes)) * 100.0
				}
				ov.Resources = res

				// 逐节点组装
				ov.Nodes = make([]clusteroverview.NodeResourceUsage, 0, len(perNodeCap))
				for name, cap := range perNodeCap {
					u := perNodeUsed[name] // 若 metrics-server 里没有该节点，会是零值
					nodeItem := clusteroverview.NodeResourceUsage{
						NodeName:         name,
						TotalCPUMilli:    cap.cpuMilli,
						UsedCPUMilli:     u.cpuMilli,
						TotalCores:       float64(cap.cpuMilli) / 1000.0,
						UsedCores:        float64(u.cpuMilli) / 1000.0,
						TotalMemoryBytes: cap.memBytes,
						UsedMemoryBytes:  u.memBytes,
						Ready:            cap.ready,
						Role:             cap.role,
					}
					if cap.cpuMilli > 0 {
						nodeItem.CPUPercent = (float64(u.cpuMilli) / float64(cap.cpuMilli)) * 100.0
					}
					if cap.memBytes > 0 {
						nodeItem.MemoryPercent = (float64(u.memBytes) / float64(cap.memBytes)) * 100.0
					}
					ov.Nodes = append(ov.Nodes, nodeItem)
				}
			}
			// 如果 metrics API 失败：保持 ov.Resources=nil，ov.Nodes 也不填 used/percent
		}
	}

	// 如果没有 metrics-server：仍返回基础概览和 perNode 总量（可选）
	// 如需在无 metrics-server 时也返回 per-node 总量，可在此处填充 ov.Nodes 但 used=0、percent=0。
	if !ov.HasMetrics && len(ov.Nodes) == 0 {
		ov.Nodes = make([]clusteroverview.NodeResourceUsage, 0, len(perNodeCap))
		for name, cap := range perNodeCap {
			ov.Nodes = append(ov.Nodes, clusteroverview.NodeResourceUsage{
				NodeName:         name,
				TotalCPUMilli:    cap.cpuMilli,
				UsedCPUMilli:     0,
				CPUPercent:       0,
				TotalCores:       float64(cap.cpuMilli) / 1000.0,
				UsedCores:        0,
				TotalMemoryBytes: cap.memBytes,
				UsedMemoryBytes:  0,
				MemoryPercent:    0,
				Ready:            cap.ready,
				Role:             cap.role,
			})
		}
	}

	return ov, nil
}






// // ClusterOverview 定义了集群概要数据的结构，用于首页 UI 展示
// type ClusterOverview struct {
// 	TotalNodes   int    `json:"total_nodes"`        // 节点总数
// 	ReadyNodes   int    `json:"ready_nodes"`        // Ready 状态的节点数
// 	TotalPods    int    `json:"total_pods"`         // 所有命名空间中的 Pod 总数
// 	AbnormalPods int    `json:"abnormal_pods"`      // 异常状态（Pending/Failed/Unknown）的 Pod 数量
// 	K8sVersion   string `json:"k8s_version"`        // Kubernetes 控制平面版本
// 	HasMetrics   bool   `json:"has_metrics_server"` // 是否检测到 metrics-server
// }

// // GetClusterOverview 返回当前集群的概要信息
// // ✅ 用于首页集群概览接口，如：GET /api/cluster/overview
// func GetClusterOverview(ctx context.Context) (*ClusterOverview, error) {
// 	client := utils.GetCoreClient()

// 	// 1️⃣ 获取 Node 列表，并统计 Ready 节点数
// 	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("无法获取 Node 列表: %w", err)
// 	}
// 	totalNodes := len(nodes.Items)
// 	readyNodes := 0
// 	for _, node := range nodes.Items {
// 		for _, cond := range node.Status.Conditions {
// 			if cond.Type == "Ready" && cond.Status == "True" {
// 				readyNodes++
// 				break
// 			}
// 		}
// 	}

// 	// 2️⃣ 获取所有命名空间下的 Pod，并统计异常 Pod 数量
// 	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("无法获取 Pod 列表: %w", err)
// 	}
// 	totalPods := len(pods.Items)
// 	abnormalPods := 0
// 	for _, pod := range pods.Items {
// 		switch pod.Status.Phase {
// 		case "Failed", "Pending", "Unknown":
// 			abnormalPods++
// 		}
// 	}

// 	// 3️⃣ 获取 Kubernetes 控制平面的版本号
// 	versionInfo, err := client.Discovery().ServerVersion()
// 	if err != nil {
// 		return nil, fmt.Errorf("无法获取 Kubernetes 版本: %w", err)
// 	}

// 	// 4️⃣ 构建并返回 ClusterOverview 对象
// 	return &ClusterOverview{
// 		TotalNodes:   totalNodes,
// 		ReadyNodes:   readyNodes,
// 		TotalPods:    totalPods,
// 		AbnormalPods: abnormalPods,
// 		K8sVersion:   versionInfo.GitVersion,
// 		HasMetrics:   utils.HasMetricsServer(),
// 	}, nil
// }

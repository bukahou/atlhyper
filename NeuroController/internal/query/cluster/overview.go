package cluster

// =======================================================================================
// 📄 overview.go（internal/query/cluster）
//
// ✨ 文件功能说明：
//     提供 Kubernetes 集群的整体概要信息，用于集群首页展示。
//     包括以下内容：
//     - 节点总数与 Ready 节点数
//     - 所有命名空间下的 Pod 总数
//     - 异常状态的 Pod 数量（Pending / Failed / Unknown）
//     - Kubernetes 控制平面版本
//     - 是否部署了 metrics-server（判断是否支持 CPU/Mem 查询）
//
// 📦 外部依赖：
//     - utils.GetCoreClient()（封装的 client-go 客户端）
//     - utils.HasMetricsServer()（检测 metrics-server 是否可用）
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 创建时间：2025年7月
// =======================================================================================

import (
	"NeuroController/internal/utils"
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterOverview 定义了集群概要数据的结构，用于首页 UI 展示
type ClusterOverview struct {
	TotalNodes   int    `json:"total_nodes"`        // 节点总数
	ReadyNodes   int    `json:"ready_nodes"`        // Ready 状态的节点数
	TotalPods    int    `json:"total_pods"`         // 所有命名空间中的 Pod 总数
	AbnormalPods int    `json:"abnormal_pods"`      // 异常状态（Pending/Failed/Unknown）的 Pod 数量
	K8sVersion   string `json:"k8s_version"`        // Kubernetes 控制平面版本
	HasMetrics   bool   `json:"has_metrics_server"` // 是否检测到 metrics-server
}

// GetClusterOverview 返回当前集群的概要信息
// ✅ 用于首页集群概览接口，如：GET /api/cluster/overview
func GetClusterOverview(ctx context.Context) (*ClusterOverview, error) {
	client := utils.GetCoreClient()

	// 1️⃣ 获取 Node 列表，并统计 Ready 节点数
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取 Node 列表: %w", err)
	}
	totalNodes := len(nodes.Items)
	readyNodes := 0
	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				readyNodes++
				break
			}
		}
	}

	// 2️⃣ 获取所有命名空间下的 Pod，并统计异常 Pod 数量
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("无法获取 Pod 列表: %w", err)
	}
	totalPods := len(pods.Items)
	abnormalPods := 0
	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case "Failed", "Pending", "Unknown":
			abnormalPods++
		}
	}

	// 3️⃣ 获取 Kubernetes 控制平面的版本号
	versionInfo, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("无法获取 Kubernetes 版本: %w", err)
	}

	// 4️⃣ 构建并返回 ClusterOverview 对象
	return &ClusterOverview{
		TotalNodes:   totalNodes,
		ReadyNodes:   readyNodes,
		TotalPods:    totalPods,
		AbnormalPods: abnormalPods,
		K8sVersion:   versionInfo.GitVersion,
		HasMetrics:   utils.HasMetricsServer(),
	}, nil
}

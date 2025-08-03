// =======================================================================================
// 📄 internal/query/node/list.go
//
// ✨ 文件功能说明：
//   获取 Kubernetes 集群中所有节点的基本信息与统计信息，用于 UI 总览页或分析模块。
//   - 获取所有 Node 简要信息（名称、IP、CPU、内存、状态等）
//   - 汇总节点总数、就绪节点数量、总 CPU / 内存资源
//   - 支持通过名称获取单个节点的完整详情信息（含状态、系统、镜像等）
//
// ✅ GET /uiapi/node/list
// 🔍 获取节点总览信息（含四个概要数值 + 表格列表）
// 用于：UI Node 总览页上方统计卡片与下方节点表格展示
//
// ✅ GET /uiapi/node/get/:name
// 🔍 获取指定 Node 的完整详细信息（系统、资源、网络、镜像等）
// 用于：Node 详情页展示、排障分析等
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package node

import (
	"context"
	"fmt"
	"math"

	"NeuroController/internal/utils"
	"NeuroController/model"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// =======================================================================================
// 🔧 工具函数：获取所有 Node 对象（原始）
//
// 返回完整的 corev1.Node 列表，用于其他函数或内部模块复用
// 不做任何额外处理，仅作为原始列表输出
// =======================================================================================

func ListAllNodes(ctx context.Context) ([]corev1.Node, error) {
	client := utils.GetCoreClient()
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node 列表失败: %w", err)
	}
	return nodes.Items, nil
}

// =======================================================================================
// ✅ GET /uiapi/node/get/:name
//
// 🔍 获取指定 Node 的完整详细信息（原始结构）
// - 包含状态、系统信息、容量、IP 地址、运行镜像等
//
// 用于：Node 详情页展示，支持进一步信息可视化
// =======================================================================================

type NodeOverviewStats struct {
	TotalNodes    int     `json:"totalNodes"`
	ReadyNodes    int     `json:"readyNodes"`
	TotalCPU      int     `json:"totalCPU"`      // 单位：Core
	TotalMemoryGB float64 `json:"totalMemoryGB"` // 单位：GiB（保留 1 位小数）
}

type NodeBrief struct {
	Name       string            `json:"name"`
	Ready      bool              `json:"ready"`
	InternalIP string            `json:"internalIP"`
	OSImage    string            `json:"osImage"`
	Arch       string            `json:"architecture"`
	CPU        int               `json:"cpu"`    // 核数
	MemoryGB   float64           `json:"memory"` // 单位：GiB，保留 1 位小数
	Labels     map[string]string `json:"labels"`
	Unschedulable bool             `json:"unschedulable"` 
}

type NodeOverviewResult struct {
	Stats NodeOverviewStats `json:"stats"`
	Nodes []NodeBrief       `json:"nodes"`
}

func GetNodeOverview(ctx context.Context) (*NodeOverviewResult, error) {
	client := utils.GetCoreClient()
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node 列表失败: %w", err)
	}

	var (
		totalCPU      int
		totalMemoryGB float64
		readyCount    int
		nodeBriefs    []NodeBrief
	)

	for _, node := range nodes.Items {
		cpuQty := node.Status.Capacity[corev1.ResourceCPU]
		memQty := node.Status.Capacity[corev1.ResourceMemory]

		cpuCores := int(cpuQty.Value())
		memMi := float64(memQty.ScaledValue(resource.Mega))
		memGi := memMi / 1024

		totalCPU += cpuCores
		totalMemoryGB += memGi

		brief := NodeBrief{
			Name:       node.Name,
			Ready:      isNodeReady(node),
			InternalIP: getInternalIP(node),
			OSImage:    node.Status.NodeInfo.OSImage,
			Arch:       node.Status.NodeInfo.Architecture,
			CPU:        cpuCores,
			MemoryGB:   math.Round(memGi*10) / 10.0, // ✅ 保留 1 位小数
			Labels:     node.Labels,
			Unschedulable: node.Spec.Unschedulable,
		}
		if brief.Ready {
			readyCount++
		}
		nodeBriefs = append(nodeBriefs, brief)
	}

	result := &NodeOverviewResult{
		Stats: NodeOverviewStats{
			TotalNodes:    len(nodes.Items),
			ReadyNodes:    readyCount,
			TotalCPU:      totalCPU,
			TotalMemoryGB: math.Round(totalMemoryGB*10) / 10.0, // 保留 1 位小数
		},
		Nodes: nodeBriefs,
	}
	return result, nil
}

func isNodeReady(n corev1.Node) bool {
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func getInternalIP(n corev1.Node) string {
	for _, addr := range n.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

// =======================================================================================
// ✅ GET /uiapi/node/get/:name
//
// 🔍 获取指定 Node 的完整详细信息（原始结构）
// - 包含状态、系统信息、容量、IP 地址、运行镜像等
//
// 用于：Node 详情页展示，支持进一步信息可视化
// =======================================================================================

// func GetNodeDetail(ctx context.Context, nodeName string) (*corev1.Node, error) {
// 	client := utils.GetCoreClient()
// 	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("获取 Node %s 失败: %w", nodeName, err)
// 	}
// 	return node, nil
// }

func GetNodeDetail(ctx context.Context, nodeName string) (*model.NodeDetailInfo, error) {
	coreClient := utils.GetCoreClient()
	metricsClient := utils.GetMetricsClient()

	// 获取 Node 对象
	node, err := coreClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node %s 失败: %w", nodeName, err)
	}

	// 获取 metrics（资源使用率）
	metrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	var usage *model.NodeUsage
	if err == nil {
		allocCPU := node.Status.Allocatable.Cpu().MilliValue()
		allocMem := node.Status.Allocatable.Memory().Value()

		usedCPU := metrics.Usage.Cpu().MilliValue()
		usedMem := metrics.Usage.Memory().Value()

		var cpuPercent, memPercent float64
		if allocCPU > 0 {
			cpuPercent = float64(usedCPU) / float64(allocCPU) * 100
			cpuPercent = math.Round(cpuPercent*10) / 10
		}
		if allocMem > 0 {
			memPercent = float64(usedMem) / float64(allocMem) * 100
			memPercent = math.Round(memPercent*10) / 10
		}

		usage = &model.NodeUsage{
			CPUUsagePercent:    cpuPercent,
			MemoryUsagePercent: memPercent,
		}
	}

	// 获取当前运行的 Pods
	pods, err := coreClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	var runningPods []corev1.Pod
	if err == nil {
		for _, pod := range pods.Items {
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				runningPods = append(runningPods, pod)
			}
		}
	}

	// 获取节点事件
	events, err := coreClient.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.kind=Node,involvedObject.name=%s", nodeName),
	})
	var eventList []corev1.Event
	if err == nil {
		eventList = events.Items
	}

	// 拼装结构体
	return &model.NodeDetailInfo{
		Node:          node,
		Unschedulable: node.Spec.Unschedulable,
		Taints:        node.Spec.Taints,
		Usage:         usage,
		RunningPods:   runningPods,
		Events:        eventList,
	}, nil
}

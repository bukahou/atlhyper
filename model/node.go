package model

import (
	corev1 "k8s.io/api/core/v1"
)

type NodeUsage struct {
	CPUUsagePercent    float64 `json:"cpuUsagePercent"`
	MemoryUsagePercent float64 `json:"memoryUsagePercent"`
}

type NodeDetailInfo struct {
	Node          *corev1.Node   `json:"node"`           // 原始 node 对象（保留）
	Unschedulable bool           `json:"unschedulable"`  // 是否 cordon
	Taints        []corev1.Taint `json:"taints"`         // 污点列表
	Usage         *NodeUsage     `json:"usage"`          // 实时 CPU / 内存 使用率
	RunningPods   []corev1.Pod   `json:"runningPods"`    // 当前运行的 Pod
	Events        []corev1.Event `json:"events"`         // 节点相关事件
}

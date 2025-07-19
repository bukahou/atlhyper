// =======================================================================================
// 📄 summary.go
//
// ✨ 文件功能说明：
//     给定 Pod 列表，统计其中各类状态的数量，供 UI 概要图表示使用
//     包括：Running 、Pending 、Failed 、Succeeded 、Unknown
//
// 📝 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package pod

import corev1 "k8s.io/api/core/v1"

// PodSummary 表示给定 Pod 列表的各类状态总计
// 实用于 UI 概览面板或状态分类柱状图
// Example:
//   Running: 12, Pending: 2, Failed: 1, Succeeded: 6, Unknown: 0

type PodSummary struct {
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Failed    int `json:"failed"`
	Succeeded int `json:"succeeded"`
	Unknown   int `json:"unknown"`
}

// SummarizePodsByStatus 统计 Pod 列表中各种状态的总数
func SummarizePodsByStatus(pods []corev1.Pod) PodSummary {
	summary := PodSummary{}

	for _, pod := range pods {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			summary.Running++
		case corev1.PodPending:
			summary.Pending++
		case corev1.PodFailed:
			summary.Failed++
		case corev1.PodSucceeded:
			summary.Succeeded++
		case corev1.PodUnknown:
			summary.Unknown++
		}
	}

	return summary
}

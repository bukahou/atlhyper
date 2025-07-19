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
	"strings"
	"time"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
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

// ListPodsByNamespace 返回指定命名空间下的 Pod 列表
func ListPodsByNamespace(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	client := utils.GetCoreClient()
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Pod 失败: %w", namespace, err)
	}
	return pods.Items, nil
}

// ListAllPodInfos 返回所有命名空间下 Pod 的简略信息（用于 UI 展示）
func ListAllPodInfos(ctx context.Context) ([]PodInfo, error) {
	rawPods, err := ListAllPods(ctx)
	if err != nil {
		return nil, err
	}

	var result []PodInfo
	for _, pod := range rawPods {
		result = append(result, convertPodToInfo(&pod))
	}
	return result, nil
}

// convertPodToInfo 将 corev1.Pod 转换为 PodInfo（精简结构体）
func convertPodToInfo(pod *corev1.Pod) PodInfo {
	deployment := "-"
	for _, owner := range pod.OwnerReferences {
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
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			ready = true
			break
		}
	}

	restartCount := int32(0)
	if len(pod.Status.ContainerStatuses) > 0 {
		restartCount = pod.Status.ContainerStatuses[0].RestartCount
	}

	startTime := ""
	if pod.Status.StartTime != nil {
		startTime = pod.Status.StartTime.Format(time.RFC3339)
	}

	return PodInfo{
		Namespace:    pod.Namespace,
		Deployment:   deployment,
		Name:         pod.Name,
		Ready:        ready,
		Phase:        string(pod.Status.Phase),
		RestartCount: restartCount,
		StartTime:    startTime,
		PodIP:        pod.Status.PodIP,
		NodeName:     pod.Spec.NodeName,
	}
}

// =======================================================================================
// 📄 list.go（internal/query/event）
//
// ✨ 文件功能说明：
//     提供 Kubernetes 集群中 Event 对象的查询能力：
//     - 获取所有命名空间下的 Event（ListAllEvents）
//     - 获取指定命名空间下的 Event（ListEventsByNamespace）
//
// ✅ 示例用途：
//     - UI 告警中心展示 Warning / Failed 等事件
//     - 命名空间页面展示事件时间线
//
// 🧪 假设输出：
//     [
//       { type: "Warning", reason: "FailedScheduling", message: "节点资源不足" },
//       { type: "Normal", reason: "Pulled", message: "镜像拉取成功" },
//     ]
//
// 📦 外部依赖：
//     - utils.GetCoreClient()：全局共享的 client-go 客户端
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 时间：2025年7月
// =======================================================================================

package event

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListAllEvents 获取所有命名空间下的 Event 列表
func ListAllEvents(ctx context.Context) ([]corev1.Event, error) {
	client := utils.GetCoreClient()

	evts, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取所有 Event 失败: %w", err)
	}
	return evts.Items, nil
}

// ListEventsByNamespace 获取指定命名空间下的 Event 列表
func ListEventsByNamespace(ctx context.Context, namespace string) ([]corev1.Event, error) {
	client := utils.GetCoreClient()

	evts, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Event 失败: %w", namespace, err)
	}
	return evts.Items, nil
}

// Get events by involved object kind/name/namespace
func ListEventsByInvolvedObject(ctx context.Context, namespace, kind, name string) ([]corev1.Event, error) {
	client := utils.GetCoreClient()

	// 获取该命名空间下的所有事件
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Event 失败: %w", namespace, err)
	}

	var matched []corev1.Event
	for _, event := range events.Items {
		if event.InvolvedObject.Kind == kind && event.InvolvedObject.Name == name {
			matched = append(matched, event)
		}
	}
	return matched, nil
}

// CountEventsByType 返回全集群范围内 Event 的类型分布（如 Warning/Normal）
func CountEventsByType(ctx context.Context) (map[string]int, error) {
	client := utils.GetCoreClient()

	events, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取所有 Event 失败: %w", err)
	}

	counts := map[string]int{}
	for _, e := range events.Items {
		counts[e.Type]++
	}
	return counts, nil
}

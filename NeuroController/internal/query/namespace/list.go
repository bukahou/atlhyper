// =======================================================================================
// 📄 list.go（internal/query/namespace）
//
// ✨ 文件功能说明：
//     提供 Kubernetes 命名空间（Namespace）的查询能力，包括：
//     - 所有命名空间的基本信息获取
//     - 根据状态（Active / Terminating）筛选
//     - 获取指定命名空间详情
//     - 统计不同状态的命名空间数量（辅助 UI 圆环图）
//
// ✅ 示例用途：
//     - UI 命名空间列表页 / 下拉选择框
//     - 集群健康度评估（多少正在终止的 NS）
//
// 🧪 示例输出：
//     [
//       { name: "default", status: "Active", labels: {...}, ... },
//       { name: "dev", status: "Terminating", ... }
//     ]
//
// 📦 外部依赖：
//     - utils.GetCoreClient()：封装的 client-go CoreV1 接口
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 时间：2025年7月
// =======================================================================================

package namespace

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceWithPodCount 封装 Namespace 元信息 + Pod 数量
type NamespaceWithPodCount struct {
	Namespace corev1.Namespace
	PodCount  int
}

// ListAllNamespaces 返回所有命名空间的列表
// ListAllNamespaces 返回所有命名空间及其 Pod 数量
func ListAllNamespaces(ctx context.Context) ([]NamespaceWithPodCount, error) {
	client := utils.GetCoreClient()

	// 获取所有命名空间
	nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Namespace 列表失败: %w", err)
	}

	// 获取所有 Pod
	podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Pod 列表失败: %w", err)
	}

	// 构建命名空间 -> pod 数量映射表
	podCountMap := make(map[string]int)
	for _, pod := range podList.Items {
		podCountMap[pod.Namespace]++
	}

	// 合并结果
	var result []NamespaceWithPodCount
	for _, ns := range nsList.Items {
		count := podCountMap[ns.Name]
		result = append(result, NamespaceWithPodCount{
			Namespace: ns,
			PodCount:  count,
		})
	}

	return result, nil
}

// // GetNamespaceByName 获取指定命名空间的详细信息
// func GetNamespaceByName(ctx context.Context, name string) (*corev1.Namespace, error) {
// 	client := utils.GetCoreClient()
// 	ns, err := client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("获取 Namespace %s 失败: %w", name, err)
// 	}
// 	return ns, nil
// }

// // ListActiveNamespaces 仅返回状态为 Active 的命名空间
// func ListActiveNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
// 	all, err := ListAllNamespaces(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var actives []corev1.Namespace
// 	for _, ns := range all {
// 		if ns.Status.Phase == corev1.NamespaceActive {
// 			actives = append(actives, ns)
// 		}
// 	}
// 	return actives, nil
// }

// // ListTerminatingNamespaces 返回正在 Terminating 的命名空间列表
// func ListTerminatingNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
// 	all, err := ListAllNamespaces(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var terms []corev1.Namespace
// 	for _, ns := range all {
// 		if ns.Status.Phase != corev1.NamespaceActive {
// 			terms = append(terms, ns)
// 		}
// 	}
// 	return terms, nil
// }

// // GetNamespacePhaseStats 返回当前命名空间状态分布统计
// // ✅ 用于 UI 圆环图 / 命名空间健康状态显示
// func GetNamespacePhaseStats(ctx context.Context) (activeCount, terminatingCount int, err error) {
// 	all, err := ListAllNamespaces(ctx)
// 	if err != nil {
// 		return 0, 0, err
// 	}
// 	for _, ns := range all {
// 		if ns.Status.Phase == corev1.NamespaceActive {
// 			activeCount++
// 		} else {
// 			terminatingCount++
// 		}
// 	}
// 	return activeCount, terminatingCount, nil
// }

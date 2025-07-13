// =======================================================================================
// 📄 list.go（internal/query/ingress）
//
// ✨ 文件功能说明：
//     提供集群中 Ingress 对象的基础查询能力，包括：
//     - 获取所有命名空间下的 Ingress（ListAllIngresses）
//     - 获取指定命名空间下的 Ingress（ListIngressesByNamespace）
//     - 获取特定 Ingress 详情（GetIngressByName）
//     - 获取已就绪 Ingress（ListReadyIngresses）
//
// ✅ 示例用途：
//     - UI 展示 Ingress 路由配置（域名 / 路径 / 转发目标）
//     - 命名空间资源页面、全局 Ingress 视图等
//
// 🧪 假设输出：
//     [
//       { name: "my-ingress", namespace: "default", rules: [...], tls: [...] },
//       ...
//     ]
//
// 📦 外部依赖：
//     - utils.GetNetworkingClient()：封装的 networking/v1 客户端
//     - utils.GetCoreClient()：用于 node 状态辅助判断
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 时间：2025年7月
// =======================================================================================

package ingress

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListAllIngresses 获取所有命名空间下的 Ingress 列表
func ListAllIngresses(ctx context.Context) ([]networkingv1.Ingress, error) {
	client := utils.GetCoreClient().NetworkingV1()

	ing, err := client.Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取所有 Ingress 失败: %w", err)
	}
	return ing.Items, nil
}

// ListIngressesByNamespace 获取指定命名空间下的 Ingress 列表
func ListIngressesByNamespace(ctx context.Context, namespace string) ([]networkingv1.Ingress, error) {
	client := utils.GetCoreClient().NetworkingV1()

	ing, err := client.Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Ingress 失败: %w", namespace, err)
	}
	return ing.Items, nil
}

// GetIngressByName 获取指定命名空间和名称的 Ingress 对象
func GetIngressByName(ctx context.Context, namespace, name string) (*networkingv1.Ingress, error) {
	client := utils.GetCoreClient().NetworkingV1()

	ing, err := client.Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Ingress %s/%s 失败: %w", namespace, name, err)
	}
	return ing, nil
}

// ListReadyIngresses 返回所有状态为 Ready 的 Ingress（即至少有 1 个 LoadBalancer IP）
func ListReadyIngresses(ctx context.Context) ([]networkingv1.Ingress, error) {
	all, err := ListAllIngresses(ctx)
	if err != nil {
		return nil, err
	}

	var ready []networkingv1.Ingress
	for _, ing := range all {
		if len(ing.Status.LoadBalancer.Ingress) > 0 {
			ready = append(ready, ing)
		}
	}
	return ready, nil
}

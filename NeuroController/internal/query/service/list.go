// =======================================================================================
// 📄 list.go（internal/query/service）
//
// ✨ 文件功能说明：
//     提供集群中 Service 对象的基础查询能力，包括：
//     - 获取所有命名空间下的 Service（ListAllServices）
//     - 获取指定命名空间下的 Service（ListServicesByNamespace）
//     - 获取指定 Service 的详情（GetServiceByName）
//     - 获取所有对外暴露的 Service（ListExternalServices）
//     - 获取所有 Headless Service（ListHeadlessServices）
//
// ✅ 示例用途：
//     - UI 展示集群 Service 列表或详情
//     - 策略层判断哪些服务暴露到外部
//
// 🧪 假设输出：
//     [
//       { name: "nginx-service", type: "ClusterIP", clusterIP: "10.43.0.1", ports: [...] },
//       { name: "api-service", type: "NodePort", nodePort: 30001, ports: [...] },
//       { name: "etcd-peer", type: "ClusterIP", clusterIP: "None", ports: [...] }, // headless
//     ]
//
// 📦 外部依赖：
//     - utils.GetCoreClient()：封装的 client-go 核心客户端
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// 📅 时间：2025年7月
// =======================================================================================

package service

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListAllServices 获取所有命名空间下的 Service 列表
func ListAllServices(ctx context.Context) ([]corev1.Service, error) {
	client := utils.GetCoreClient()

	services, err := client.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取所有 Service 失败: %w", err)
	}
	return services.Items, nil
}

// ListServicesByNamespace 获取指定命名空间下的 Service 列表
func ListServicesByNamespace(ctx context.Context, namespace string) ([]corev1.Service, error) {
	client := utils.GetCoreClient()

	services, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Service 失败: %w", namespace, err)
	}
	return services.Items, nil
}

// GetServiceByName 获取某个命名空间下的具体 Service 详情
func GetServiceByName(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	client := utils.GetCoreClient()

	svc, err := client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Service %s/%s 失败: %w", namespace, name, err)
	}
	return svc, nil
}

// ListExternalServices 获取所有对外暴露的 Service（类型为 NodePort 或 LoadBalancer）
func ListExternalServices(ctx context.Context) ([]corev1.Service, error) {
	allSvcs, err := ListAllServices(ctx)
	if err != nil {
		return nil, err
	}

	var externals []corev1.Service
	for _, svc := range allSvcs {
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer || svc.Spec.Type == corev1.ServiceTypeNodePort {
			externals = append(externals, svc)
		}
	}
	return externals, nil
}

// ListHeadlessServices 获取所有 Headless Service（ClusterIP 为 None）
func ListHeadlessServices(ctx context.Context) ([]corev1.Service, error) {
	allSvcs, err := ListAllServices(ctx)
	if err != nil {
		return nil, err
	}

	var headless []corev1.Service
	for _, svc := range allSvcs {
		if svc.Spec.ClusterIP == "None" {
			headless = append(headless, svc)
		}
	}
	return headless, nil
}

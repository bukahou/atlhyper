// =======================================================================================
// 📄 service_util.go
//
// ✨ Description:
//     1️⃣ GetServiceNameFromPod(): Match a Service based on a Pod's label selector.
//     2️⃣ CheckServiceEndpointStatus(): Check whether a Service has ready Endpoints.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package utils

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// 尝试根据 Pod 的标签匹配所属的 Service 名称
//
// 🔹 Service 的 selector 标签是关联 Pod 的关键
func GetServiceNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient()

	var serviceList corev1.ServiceList
	if err := cli.List(ctx, &serviceList, client.InNamespace(pod.Namespace)); err != nil {

		return "", err
	}

	// 🔀 遍历所有 Service，尝试与 Pod 标签进行匹配
	for _, svc := range serviceList.Items {
		match := true
		for key, val := range svc.Spec.Selector {
			if podVal, ok := pod.Labels[key]; !ok || podVal != val {
				match = false
				break
			}
		}
		if match {

			CheckServiceEndpointStatus(ctx, pod.Namespace, svc.Name)
			return svc.Name, nil
		}
	}

	return "", nil // 未找到匹配的 Service
}

// 检查指定 Service 是否存在就绪的 Endpoints
func CheckServiceEndpointStatus(ctx context.Context, namespace, name string) {
	cli := GetClient()

	var endpoints corev1.Endpoints
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &endpoints); err != nil {
		return
	}

	readyCount := 0
	for _, subset := range endpoints.Subsets {
		readyCount += len(subset.Addresses)
	}

	if readyCount == 0 {
	} else {
	}
}

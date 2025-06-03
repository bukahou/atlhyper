// =======================================================================================
// 📄 service_util.go
//
// ✨ 功能说明：
//     1️⃣ GetServiceNameFromPod(): 通过 Pod 的 label 选择器匹配 Service
//     2️⃣ CheckServiceEndpointStatus(): 根据 Service 名称检查 Endpoints 是否正常
//
// 🖍️ 作者：武夏锋（@ZGMF-X10A）
// =======================================================================================

package utils

import (
	"context"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServiceNameFromPod 根据 Pod 的 Label 进行 Service 匹配分析
// 🔹 选择器是 Service 中重要的匹配元素
func GetServiceNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient()

	var serviceList corev1.ServiceList
	if err := cli.List(ctx, &serviceList, client.InNamespace(pod.Namespace)); err != nil {
		Error(ctx, "❌ 列表 Service 失败",
			zap.String("namespace", pod.Namespace),
			zap.Error(err),
		)
		return "", err
	}

	// 🔀 根据 label selector 匹配 Service
	for _, svc := range serviceList.Items {
		match := true
		for key, val := range svc.Spec.Selector {
			if podVal, ok := pod.Labels[key]; !ok || podVal != val {
				match = false
				break
			}
		}
		if match {
			Info(ctx, "✅ 匹配到 Service",
				zap.String("service", svc.Name),
				zap.String("pod", pod.Name),
			)

			CheckServiceEndpointStatus(ctx, pod.Namespace, svc.Name)
			return svc.Name, nil
		}
	}

	return "", nil // 未匹配到
}

// CheckServiceEndpointStatus 检查指定 Service 是否关联到合法 Endpoint
func CheckServiceEndpointStatus(ctx context.Context, namespace, name string) {
	cli := GetClient()

	var endpoints corev1.Endpoints
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &endpoints); err != nil {
		Warn(ctx, "⚠️ 获取 Endpoints 失败",
			zap.String("service", name),
			zap.Error(err),
		)
		return
	}

	readyCount := 0
	for _, subset := range endpoints.Subsets {
		readyCount += len(subset.Addresses)
	}

	if readyCount == 0 {
		Warn(ctx, "🚨 Service 相关 Endpoint 中未包含任何可用 Pod",
			zap.String("service", name),
			zap.String("namespace", namespace),
		)
	} else {
		Info(ctx, "✅ Endpoint 连接正常",
			zap.String("service", name),
			zap.Int("ready", readyCount),
		)
	}
}

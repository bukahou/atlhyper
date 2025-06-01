// =======================================================================================
// 📄 deployment_util.go
//
// ✨ 功能说明：
//     提供工具函数，从异常 Pod 的 OwnerReference 中追溯获取其所属 Deployment 名称。
//     支持 Pod → ReplicaSet → Deployment 的 ownerRef 链式解析。
//     使用 controller-runtime 的 client 进行资源查询，适配所有标准 K8s 部署资源。
//
// 🛠️ 提供功能：
//     - GetDeploymentNameFromPod(pod *corev1.Pod) (string, error)
//
// 📦 依赖：
//     - k8s.io/api/core/v1
//     - k8s.io/api/apps/v1
//     - controller-runtime client
//
// 📍 使用场景：
//     - Watcher 采集异常 Pod 日志并需判断其归属 Deployment
//     - Scaler 缩容前判断目标对象是否是可控 Deployment
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package utils

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"go.uber.org/zap"
)

// GetDeploymentNameFromPod 尝试从 Pod 的 ownerRef 中获取对应的 Deployment 名称
func GetDeploymentNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient()

	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			// 先获取 ReplicaSet
			rs := &appsv1.ReplicaSet{}
			err := cli.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name,
			}, rs)
			if err != nil {
				Error(ctx, "❌ 无法获取 ReplicaSet", zap.String("replicaSet", owner.Name), zap.Error(err))
				return "", fmt.Errorf("failed to get replicaset %s: %w", owner.Name, err)
			}

			// 从 ReplicaSet 再查 owner 是否为 Deployment
			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					Info(ctx, "✅ 成功获取所属 Deployment",
						zap.String("pod", pod.Name),
						zap.String("deployment", rsOwner.Name),
					)
					return rsOwner.Name, nil
				}
			}

			return "", errors.New("ReplicaSet 没有指向 Deployment 的 ownerRef")
		}
	}

	return "", errors.New("Pod 无有效的 ReplicaSet ownerRef")
}

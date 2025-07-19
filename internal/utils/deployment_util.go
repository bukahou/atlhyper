// =======================================================================================
// 📄 deployment_util.go
//
// ✨ Description:
//     Utility functions for inferring and checking the Deployment associated with a given Pod.
//
//     1️⃣ GetDeploymentNameFromPod():
//         Traces the Deployment a Pod belongs to via its ReplicaSet owner reference.
//
//     2️⃣ CheckDeploymentReplicaStatusByName():
//         Retrieves replica status for a specific Deployment (desired vs ready vs unavailable).
//
//     3️⃣ ExtractDeploymentName():
//         Infers Deployment name from Pod name using controller references or fallback pattern.
//
//     4️⃣ IsDeploymentRecovered():
//         Determines whether a Deployment has recovered based on its ReadyReplicas.
//
//     5️⃣ GetExpectedReplicaCount():
//         Returns the desired replica count for a given Deployment, or a fallback value.
//
// 🧠 Use Cases:
//     - Tracing Deployment ownership of abnormal Pods
//     - Aggregating events for alert grouping
//     - Evaluating Deployment health status
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// 根据给定的 Pod，尝试提取其关联的 Deployment 名称。
// 🧠 原理：Pod ➜ ReplicaSet ➜ Deployment（通过 ownerReference 链路）
//
// 📍 使用场景：
//   - 当某个 Pod 异常时，追溯其属于哪个 Deployment，
//     用于聚合异常、触发告警或执行副本数控制。
func GetDeploymentNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient() // ✅ 使用全局 controller-runtime client

	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			rs := &appsv1.ReplicaSet{}
			err := cli.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: owner.Name}, rs)
			if err != nil {
				return "", fmt.Errorf("获取 ReplicaSet 失败 %s: %w", owner.Name, err)
			}

			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					deployName := rsOwner.Name

					CheckDeploymentReplicaStatusByName(ctx, pod.Namespace, deployName)
					return deployName, nil
				}
			}

			return "", errors.New("ReplicaSet 缺少 Deployment ownerRef")
		}
	}

	return "", errors.New("Pod 没有有效的 ReplicaSet ownerRef")
}

// 检查指定 Deployment 的副本状态
//
// 📍 使用场景：
//   - 确定某个异常 Pod 所属 Deployment 后，验证其副本数是否存在缺失或不可用情况
func CheckDeploymentReplicaStatusByName(ctx context.Context, namespace string, name string) {
	cli := GetClient()

	var deployment appsv1.Deployment
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment); err != nil {
		return
	}

	desired := *deployment.Spec.Replicas
	ready := deployment.Status.ReadyReplicas
	unavailable := deployment.Status.UnavailableReplicas

	if ready < desired {
	}

	if unavailable > 0 {
	}
}

// 若获取失败则返回默认值
func GetExpectedReplicaCount(namespace, name string) int {
	cli := GetClient()
	var deploy appsv1.Deployment

	if err := cli.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, &deploy); err != nil {
		return 2
	}

	return int(*deploy.Spec.Replicas)
}

// IsDeploymentRecovered 判断 Deployment 的副本是否全部 Ready（已完全恢复）
func IsDeploymentRecovered(ctx context.Context, namespace, name string) (bool, error) {
	cli := GetClient() // 假设你已有封装的全局 client getter
	var deploy appsv1.Deployment

	err := cli.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &deploy)
	if err != nil {
		return false, err // 获取失败
	}

	// 比较 Ready 和 Desired 副本数
	if deploy.Status.ReadyReplicas >= *deploy.Spec.Replicas {
		return true, nil // ✅ 已恢复
	}
	return false, nil // ❌ 未恢复
}

// 从 Pod 对象反查其所属 Deployment 的名称
func ExtractDeploymentName(podName, namespace string) string {
	ctx := context.TODO()

	cli := GetClient()

	var pod corev1.Pod
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		return fallbackName(podName)
	}

	var rsName string
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			rsName = owner.Name
			break
		}
	}
	if rsName == "" {
		return fallbackName(podName)
	}

	var rs appsv1.ReplicaSet
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: rsName}, &rs); err != nil {
		return fallbackName(rsName)
	}

	for _, owner := range rs.OwnerReferences {
		if owner.Kind == "Deployment" {
			return owner.Name
		}
	}

	return fallbackName(rsName)
}

// fallbackName 从名称中去掉 hash 推测 Deployment 名
func fallbackName(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return name
	}

	return strings.Join(parts[:len(parts)-1], "-")
}

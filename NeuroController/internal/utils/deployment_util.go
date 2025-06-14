// =======================================================================================
// 📄 deployment_util.go
//
// ✨ Description:
//     1️⃣ GetDeploymentNameFromPod(): Trace the Deployment name a Pod belongs to via ReplicaSet ownerRef.
//     2️⃣ CheckDeploymentReplicaStatusByName(): Fetch and verify replica state for a specific Deployment.
//
// ✍️ Author: bukahou (@ZGMF-X10A)
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

	"go.uber.org/zap"
)

// 根据给定的 Pod，尝试提取其关联的 Deployment 名称。
// 🧠 原理：Pod ➜ ReplicaSet ➜ Deployment（通过 ownerReference 链路）
//
// 📍 使用场景：
//   - 当某个 Pod 异常时，追溯其属于哪个 Deployment，
//     用于聚合异常、触发告警或执行副本数控制。
func GetDeploymentNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient() // ✅ 使用全局 controller-runtime client

	// 🔁 遍历 Pod 的 ownerReferences
	for _, owner := range pod.OwnerReferences {
		// 🔍 如果 owner 是 ReplicaSet，则继续追溯
		if owner.Kind == "ReplicaSet" {

			// ✅ 1️⃣ 获取对应的 ReplicaSet 对象
			rs := &appsv1.ReplicaSet{}
			err := cli.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name, // 🧭 从 Pod owner 中提取
			}, rs)
			if err != nil {
				// ❌ 无法获取 ReplicaSet（可能已被删除）
				Error(ctx, "❌ 无法获取 ReplicaSet", zap.String("replicaSet", owner.Name), zap.Error(err))
				return "", fmt.Errorf("获取 ReplicaSet 失败 %s: %w", owner.Name, err)
			}

			// 🔁 继续追溯：检查该 ReplicaSet 是否由 Deployment 拥有
			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					deployName := rsOwner.Name

					// 🟢 追溯成功：找到了 Deployment
					Info(ctx, "✅ 成功解析 Pod 所属的 Deployment",
						zap.String("pod", pod.Name),
						zap.String("deployment", deployName),
					)

					// 🔁 可选：立即检查该 Deployment 的副本状态
					CheckDeploymentReplicaStatusByName(ctx, pod.Namespace, deployName)

					return deployName, nil
				}
			}

			// ❌ ReplicaSet 没有 Deployment ownerRef
			return "", errors.New("ReplicaSet 缺少 Deployment ownerRef")
		}
	}

	// ❌ Pod 没有有效的 ReplicaSet ownerRef
	return "", errors.New("Pod 没有有效的 ReplicaSet ownerRef")
}

// 检查指定 Deployment 的副本状态
//
// 📍 使用场景：
//   - 确定某个异常 Pod 所属 Deployment 后，验证其副本数是否存在缺失或不可用情况
func CheckDeploymentReplicaStatusByName(ctx context.Context, namespace string, name string) {
	cli := GetClient()

	var deployment appsv1.Deployment
	// 🔍 获取 Deployment 对象
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment); err != nil {
		// ❌ 获取失败（可能已删除或尚未创建）
		Error(ctx, "❌ 获取 Deployment 状态失败",
			zap.String("deployment", name),
			zap.Error(err),
		)
		return
	}

	// ✅ 提取副本状态信息
	desired := *deployment.Spec.Replicas                 // 期望副本数
	ready := deployment.Status.ReadyReplicas             // 实际就绪副本数
	unavailable := deployment.Status.UnavailableReplicas // 当前不可用副本数

	// 🚨 情况 1：实际副本少于期望副本
	if ready < desired {
		Warn(ctx, "🚨 Deployment 副本就绪数不足",
			zap.String("deployment", name),
			zap.Int32("desired", desired),
			zap.Int32("ready", ready),
		)
	}

	// ⚠️ 情况 2：存在不可用副本
	if unavailable > 0 {
		Warn(ctx, "⚠️ Deployment 存在不可用副本",
			zap.String("deployment", name),
			zap.Int32("unavailable", unavailable),
		)
	}
}

// 安全获取指定 Deployment 的期望副本数（默认值为 1）
// 若获取失败则返回默认值
func GetExpectedReplicaCount(namespace, name string) int {
	cli := GetClient()
	var deploy appsv1.Deployment

	if err := cli.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &deploy); err != nil {
		Warn(context.TODO(), "⚠️ 获取 Deployment 副本数失败，使用默认值 2",
			zap.String("deployment", name),
			zap.Error(err),
		)
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
	Info(context.TODO(), "🔍 调用 ExtractDeploymentName",
		zap.String("传入 podName", podName),
		zap.String("传入 namespace", namespace),
	)

	cli := GetClient() // 你已实现的封装 client

	// 获取 Pod
	var pod corev1.Pod
	if err := cli.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: podName}, &pod); err != nil {
		Warn(context.TODO(), "⚠️ 获取 Pod 失败，回退使用 podName 推测 deployment",
			zap.String("pod", podName),
			zap.Error(err),
		)
		return fallbackName(podName)
	}

	// 获取 ReplicaSet 名称
	var rsName string
	for _, owner := range pod.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			rsName = owner.Name
			break
		}
	}
	if rsName == "" {
		Warn(context.TODO(), "⚠️ Pod 未找到 ReplicaSet 归属，回退使用 podName 推测 deployment",
			zap.String("pod", podName),
		)
		return fallbackName(podName)
	}

	// 获取 ReplicaSet 对象
	var rs appsv1.ReplicaSet
	if err := cli.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: rsName}, &rs); err != nil {
		Warn(context.TODO(), "⚠️ 获取 ReplicaSet 失败，回退使用 rsName 推测 deployment",
			zap.String("rs", rsName),
			zap.Error(err),
		)
		return fallbackName(rsName)
	}

	// 获取 Deployment 名称
	for _, owner := range rs.OwnerReferences {
		if owner.Kind == "Deployment" {
			return owner.Name
		}
	}

	// 最后失败仍用 rsName 推测
	return fallbackName(rsName)
}

// fallbackName 从名称中去掉 hash 推测 Deployment 名
func fallbackName(name string) string {
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return name
	}

	Warn(context.TODO(), "⚠️ fallbackName 被调用",
		zap.String("原始 podName", name),
	)
	return strings.Join(parts[:len(parts)-1], "-")
}

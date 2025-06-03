// =======================================================================================
// 📄 deployment_util.go
//
// ✨ 功能说明：
//     1️⃣ GetDeploymentNameFromPod(): 提取 Pod 所属 Deployment 名称（通过 ReplicaSet ownerRef）
//     2️⃣ CheckDeploymentReplicaStatusByName(): 通过 Deployment 名称获取副本状态信息
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
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
// GetDeploymentNameFromPod 尝试从 Pod 的 ownerRef 中获取对应的 Deployment 名称
// 🧠 逻辑：Pod ➜ ReplicaSet ➜ Deployment 的 owner 链回溯
// 📌 使用场景：
//   - 当 Pod 异常时，需要判断它属于哪个 Deployment，便于聚合信息、发通知、执行缩容等。
func GetDeploymentNameFromPod(ctx context.Context, pod *corev1.Pod) (string, error) {
	cli := GetClient() // ✅ 获取统一的 controller-runtime client 实例

	// 🔁 遍历 Pod 的所有 OwnerReferences（可能有多个）
	for _, owner := range pod.OwnerReferences {
		// 🔍 如果 Owner 是 ReplicaSet，说明这个 Pod 是由该 ReplicaSet 创建的
		if owner.Kind == "ReplicaSet" {

			// ✅ 1️⃣ 获取对应的 ReplicaSet 对象
			rs := &appsv1.ReplicaSet{}
			err := cli.Get(ctx, client.ObjectKey{
				Namespace: pod.Namespace,
				Name:      owner.Name, // 🧭 从 Pod 的 owner 中获取 ReplicaSet 名称
			}, rs)
			if err != nil {
				// ❌ 拉取失败，可能 ReplicaSet 已被删除
				Error(ctx, "❌ 无法获取 ReplicaSet", zap.String("replicaSet", owner.Name), zap.Error(err))
				return "", fmt.Errorf("failed to get replicaset %s: %w", owner.Name, err)
			}

			// 🔁 继续遍历 ReplicaSet 的 owner，查找是否是由 Deployment 控制的
			for _, rsOwner := range rs.OwnerReferences {
				if rsOwner.Kind == "Deployment" {
					// ✅ 找到目标 Deployment！
					deployName := rsOwner.Name

					// 🟢 打印日志：成功追溯到所属 Deployment
					Info(ctx, "✅ 成功获取所属 Deployment",
						zap.String("pod", pod.Name),
						zap.String("deployment", deployName),
					)

					// 🔁 调用副本数检查函数：立即分析该 Deployment 是否副本不足
					CheckDeploymentReplicaStatusByName(ctx, pod.Namespace, deployName)

					return deployName, nil // ✅ 返回结果
				}
			}

			// ❌ 若 ReplicaSet 没有 Deployment ownerRef，则终止本分支处理
			return "", errors.New("ReplicaSet 没有指向 Deployment 的 ownerRef")
		}
	}

	// ❌ 若 Pod 无任何 ReplicaSet 类型的 OwnerReference，则说明其不是由 Deployment 创建（如 Job）
	return "", errors.New("Pod 无有效的 ReplicaSet ownerRef")
}

// CheckDeploymentReplicaStatusByName 检查给定 Deployment 的副本状态
// 📌 功能说明：
//   - 获取指定 Deployment 的目标副本数（desired）与当前就绪副本数（ready）
//   - 检查是否存在未就绪（Ready < Desired）或不可用副本（Unavailable > 0）
//
// 📍 使用场景：
//   - 在发现异常 Pod 后，回溯其所属 Deployment，并进一步检查是否副本不足或存在不可用副本
func CheckDeploymentReplicaStatusByName(ctx context.Context, namespace string, name string) {
	cli := GetClient() // ✅ 获取 controller-runtime Client

	var deployment appsv1.Deployment
	// 🔍 查询指定 namespace + name 的 Deployment 对象
	if err := cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment); err != nil {
		// ❌ 获取失败（可能已被删除或 API 异常）
		Error(ctx, "❌ 获取 Deployment 状态失败",
			zap.String("deployment", name),
			zap.Error(err),
		)
		return
	}

	// ✅ 抽取副本状态信息
	desired := *deployment.Spec.Replicas                 // 期望副本数（用户配置）
	ready := deployment.Status.ReadyReplicas             // 实际就绪副本数（K8s 当前状态）
	unavailable := deployment.Status.UnavailableReplicas // 不可用副本数（当前不能服务的 Pod 数量）

	// 🚨 1️⃣ 如果 Ready 副本数小于 Desired，说明存在未就绪副本
	if ready < desired {
		Warn(ctx, "🚨 Deployment Ready Replica 不足",
			zap.String("deployment", name),
			zap.Int32("desired", desired),
			zap.Int32("ready", ready),
		)
	}

	// ⚠️ 2️⃣ 如果存在不可用副本，则记录该状态
	if unavailable > 0 {
		Warn(ctx, "⚠️ Deployment 包含 Unavailable Replica",
			zap.String("deployment", name),
			zap.Int32("unavailable", unavailable),
		)
	}
}

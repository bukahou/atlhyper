// =======================================================================================
// 📄 scale.go
//
// ✨ 功能说明：
//     提供更新 Deployment 副本数的操作（用于 UI 后端的“扩缩容”功能）
//     由外部接口层调用，实际更新 Deployment 的 .spec.replicas 字段
//
// 📍 调用链：
//     external → interfaces → internal/operator/deployment/UpdateReplicas
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: July 2025
// =======================================================================================

package deployment

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// UpdateReplicas 修改指定 Deployment 的副本数（扩/缩容）
//
// 参数：
//   - ctx: 上下文
//   - namespace: 所在命名空间
//   - name: Deployment 名称
//   - replicas: 目标副本数（int32）
//
// 返回：
//   - error: 若失败则返回错误
// UpdateReplicas 修改 Deployment 的副本数（扩/缩容）
// UpdateReplicas 使用 StrategicMergePatch 修改 Deployment 的副本数（扩/缩容）
func UpdateReplicas(ctx context.Context, namespace, name string, replicas int32) error {
	client := utils.GetCoreClient()

	// 构造 Patch 字符串，仅修改 replicas 字段
	patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas))

	// 执行 PATCH 操作
	_, err := client.AppsV1().Deployments(namespace).Patch(
		ctx,
		name,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("PATCH 更新副本数失败: %w", err)
	}

	return nil
}
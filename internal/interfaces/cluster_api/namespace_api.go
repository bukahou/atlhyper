// =======================================================================================
// 📄 namespace_api.go (interfaces/ui_api)
//
// ✨ 文件功能说明：
//     展示 internal/query/namespace 模块实现的网络无关逻辑，提供给 external/http handler 使用：
//     - 查询全部 namespace
//     - 按名称查询
//     - 按状态分类 (active / terminating)
//     - 得到 namespace 状态统计
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package clusterapi

import (
	"context"

	"NeuroController/internal/query/namespace"
)

// GetAllNamespaces 获取所有 Namespace
func GetAllNamespaces(ctx context.Context) ([]namespace.NamespaceWithPodCount, error) {
	return namespace.ListAllNamespaces(ctx)
}

// // GetNamespaceByName 通过名称查询 Namespace
// func GetNamespaceByName(ctx context.Context, name string) (*corev1.Namespace, error) {
// 	return namespace.GetNamespaceByName(ctx, name)
// }

// // GetActiveNamespaces 获取状态为 Active 的 Namespace
// func GetActiveNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
// 	return namespace.ListActiveNamespaces(ctx)
// }

// // GetTerminatingNamespaces 获取 Terminating 的 Namespace
// func GetTerminatingNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
// 	return namespace.ListTerminatingNamespaces(ctx)
// }

// // GetNamespaceStatusStats 获取 Namespace 的状态统计数据
// func GetNamespaceStatusStats(ctx context.Context) (int, int, error) {
// 	return namespace.GetNamespacePhaseStats(ctx)
// }

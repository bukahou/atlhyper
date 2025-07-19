// =======================================================================================
// 📄 ingress_api.go（interfaces/ui_api）
//
// ✨ 文件功能说明：
//     对 internal/query/ingress 提供的 Ingress 查询逻辑进行统一封装，供 HTTP handler 调用：
//     - 获取所有 Ingress
//     - 获取指定命名空间 Ingress
//     - 获取特定 Ingress 对象
//     - 获取状态为 Ready 的 Ingress（已分配 LoadBalancer IP）
//
// 📦 依赖模块：
//     - internal/query/ingress
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package uiapi

import (
	"context"

	"NeuroController/internal/query/ingress"

	networkingv1 "k8s.io/api/networking/v1"
)

// GetAllIngresses 获取全集群所有 Ingress
func GetAllIngresses(ctx context.Context) ([]networkingv1.Ingress, error) {
	return ingress.ListAllIngresses(ctx)
}

// GetIngressesByNamespace 获取指定命名空间下的 Ingress
func GetIngressesByNamespace(ctx context.Context, ns string) ([]networkingv1.Ingress, error) {
	return ingress.ListIngressesByNamespace(ctx, ns)
}

// GetIngressByName 获取指定命名空间和名称的 Ingress 对象
func GetIngressByName(ctx context.Context, ns, name string) (*networkingv1.Ingress, error) {
	return ingress.GetIngressByName(ctx, ns, name)
}

// GetReadyIngresses 获取已分配 IP 的 Ingress
func GetReadyIngresses(ctx context.Context) ([]networkingv1.Ingress, error) {
	return ingress.ListReadyIngresses(ctx)
}

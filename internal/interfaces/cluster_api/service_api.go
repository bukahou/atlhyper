// =======================================================================================
// 📄 service_api.go（interfaces/ui_api）
//
// ✨ 文件功能说明：
//     封装 Service 查询模块的外部接口，供 handler 层调用，避免直接依赖 query 层：
//     - 获取所有 Service
//     - 命名空间筛选
//     - 获取单个 Service 详情
//     - 获取外部服务（NodePort / LoadBalancer）
//     - 获取 Headless 服务
//
// 📦 依赖模块：
//     - internal/query/service
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package clusterapi

import (
	"context"

	"NeuroController/internal/query/service"

	corev1 "k8s.io/api/core/v1"
)

// GetAllServices 获取所有命名空间下的 Service
func GetAllServices(ctx context.Context) ([]corev1.Service, error) {
	return service.ListAllServices(ctx)
}

// GetServicesByNamespace 获取指定命名空间下的 Service
func GetServicesByNamespace(ctx context.Context, namespace string) ([]corev1.Service, error) {
	return service.ListServicesByNamespace(ctx, namespace)
}

// GetServiceByName 获取某个命名空间下的 Service 详情
func GetServiceByName(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	return service.GetServiceByName(ctx, namespace, name)
}

// GetExternalServices 获取所有类型为 NodePort / LoadBalancer 的服务
func GetExternalServices(ctx context.Context) ([]corev1.Service, error) {
	return service.ListExternalServices(ctx)
}

// GetHeadlessServices 获取所有 Headless 类型的服务（ClusterIP=None）
func GetHeadlessServices(ctx context.Context) ([]corev1.Service, error) {
	return service.ListHeadlessServices(ctx)
}

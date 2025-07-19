// =======================================================================================
// 📄 deployment_api.go（interfaces/ui_api）
//
// ✨ 文件功能说明：
//     定义 Deployment 的 REST 接口，包括：
//     - 所有 / 指定命名空间列表
//     - 获取特定名称
//     - 获取不可用 Deployment
//     - 获取状态为 progressing 的 Deployment
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package uiapi

import (
	deploymentop "NeuroController/internal/operator/deployment"
	"NeuroController/internal/query/deployment"
	"context"

	appsv1 "k8s.io/api/apps/v1"
)

// GetAllDeployments 获取所有命名空间的 Deployment
func GetAllDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	return deployment.ListAllDeployments(ctx)
}

// GetDeploymentsByNamespace 获取指定命名空间的 Deployment
func GetDeploymentsByNamespace(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	return deployment.ListDeploymentsByNamespace(ctx, namespace)
}

// GetDeploymentByName 获取指定命名空间与名称的 Deployment
func GetDeploymentByName(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	return deployment.GetDeploymentByName(ctx, namespace, name)
}

// GetUnavailableDeployments 获取副本未全部 Ready 的 Deployment
func GetUnavailableDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	return deployment.ListUnavailableDeployments(ctx)
}

// GetProgressingDeployments 获取处于 progressing 状态的 Deployment
func GetProgressingDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	return deployment.ListProgressingDeployments(ctx)
}

// UpdateDeploymentReplicas 修改指定 Deployment 的副本数
//
// 参数：
//   - ctx: 上下文
//   - namespace: Deployment 所在命名空间
//   - name: Deployment 名称
//   - replicas: 目标副本数（int32）
//
// 返回：
//   - error: 若失败则返回错误
func UpdateDeploymentReplicas(ctx context.Context, namespace, name string, replicas int32) error {
	return deploymentop.UpdateReplicas(ctx, namespace, name, replicas)
}

// UpdateDeploymentImage 更新指定 Deployment 的所有容器镜像
// 参数：
//   - ctx: 上下文
//   - namespace: Deployment 所在命名空间
//   - name: Deployment 名称
//   - newImage: 新的容器镜像名称
// 返回：
//   - error: 若失败则返回错误
func UpdateDeploymentImage(ctx context.Context, namespace, name, newImage string) error {
	return deploymentop.UpdateAllContainerImages(ctx, namespace, name, newImage)
}
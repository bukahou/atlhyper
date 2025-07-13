package deployment

import (
	"context"
	"fmt"

	"NeuroController/internal/utils"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListAllDeployments 获取所有命名空间下的 Deployment 列表
func ListAllDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	client := utils.GetCoreClient()

	deployments, err := client.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取所有 Deployment 失败: %w", err)
	}
	return deployments.Items, nil
}

// ListDeploymentsByNamespace 获取指定命名空间下的 Deployment 列表
func ListDeploymentsByNamespace(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	client := utils.GetCoreClient()

	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取命名空间 %s 的 Deployment 失败: %w", namespace, err)
	}
	return deployments.Items, nil
}

// GetDeploymentByName 获取指定命名空间与名称的 Deployment 对象
func GetDeploymentByName(ctx context.Context, namespace, name string) (*appsv1.Deployment, error) {
	client := utils.GetCoreClient()

	dep, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Deployment %s/%s 失败: %w", namespace, name, err)
	}
	return dep, nil
}

// ListUnavailableDeployments 返回所有副本未全部 Ready 的 Deployment
func ListUnavailableDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	all, err := ListAllDeployments(ctx)
	if err != nil {
		return nil, err
	}

	var result []appsv1.Deployment
	for _, dep := range all {
		if dep.Status.ReadyReplicas < dep.Status.Replicas {
			result = append(result, dep)
		}
	}
	return result, nil
}

// ListProgressingDeployments 返回状态为 Progressing 的 Deployment（基于 Conditions）
func ListProgressingDeployments(ctx context.Context) ([]appsv1.Deployment, error) {
	all, err := ListAllDeployments(ctx)
	if err != nil {
		return nil, err
	}

	var result []appsv1.Deployment
	for _, dep := range all {
		for _, cond := range dep.Status.Conditions {
			if cond.Type == appsv1.DeploymentProgressing && cond.Status == "True" {
				result = append(result, dep)
				break
			}
		}
	}
	return result, nil
}

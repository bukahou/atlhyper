package uiapi

import (
	"context"

	"NeuroController/internal/query/configmap"

	corev1 "k8s.io/api/core/v1"
)

// GetAllConfigMaps 返回所有命名空间下的 ConfigMap（如需）
func GetAllConfigMaps(ctx context.Context) ([]corev1.ConfigMap, error) {
	return configmap.ListAllConfigMaps(ctx)
}

// GetConfigMapsByNamespace 返回指定命名空间下的所有 ConfigMap
func GetConfigMapsByNamespace(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	return configmap.ListConfigMapsByNamespace(ctx, namespace)
}

// GetConfigMapDetail 返回指定命名空间和名称的 ConfigMap
func GetConfigMapDetail(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	return configmap.GetConfigMap(ctx, namespace, name)
}

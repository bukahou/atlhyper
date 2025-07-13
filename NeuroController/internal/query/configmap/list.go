package configmap

import (
	"context"

	"NeuroController/internal/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var systemConfigMapNames = map[string]struct{}{
	"kube-root-ca.crt": {},
}

// ListConfigMapsByNamespace 返回指定命名空间下的所有（自定义）ConfigMap 列表
func ListConfigMapsByNamespace(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	configMapInterface := utils.GetCoreClient().CoreV1().ConfigMaps(namespace)

	listOptions := metav1.ListOptions{}

	configMapList, listErr := configMapInterface.List(ctx, listOptions)
	if listErr != nil {
		return nil, listErr
	}

	var filtered []corev1.ConfigMap
	for _, cm := range configMapList.Items {
		if _, isSystem := systemConfigMapNames[cm.Name]; isSystem {
			continue
		}
		filtered = append(filtered, cm)
	}

	return filtered, nil
}

// ListAllConfigMaps 返回所有命名空间下的所有（自定义）ConfigMap 列表
func ListAllConfigMaps(ctx context.Context) ([]corev1.ConfigMap, error) {
	configMapInterface := utils.GetCoreClient().CoreV1().ConfigMaps("")

	listOptions := metav1.ListOptions{}

	configMapList, listErr := configMapInterface.List(ctx, listOptions)
	if listErr != nil {
		return nil, listErr
	}

	var filtered []corev1.ConfigMap
	for _, cm := range configMapList.Items {
		if _, isSystem := systemConfigMapNames[cm.Name]; isSystem {
			continue
		}
		filtered = append(filtered, cm)
	}

	return filtered, nil
}

// GetConfigMap 获取指定命名空间和名称的 ConfigMap（系统 ConfigMap 返回 nil）
func GetConfigMap(ctx context.Context, namespace string, name string) (*corev1.ConfigMap, error) {
	if _, isSystem := systemConfigMapNames[name]; isSystem {
		// 直接返回 nil，避免前端访问系统资源
		return nil, nil
	}

	configMapInterface := utils.GetCoreClient().CoreV1().ConfigMaps(namespace)

	getOptions := metav1.GetOptions{}

	configMap, getErr := configMapInterface.Get(ctx, name, getOptions)
	if getErr != nil {
		return nil, getErr
	}

	return configMap, nil
}

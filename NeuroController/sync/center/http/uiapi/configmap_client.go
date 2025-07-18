package uiapi

import (
	"NeuroController/sync/center/http"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// ===============================
// 📌 GET /agent/uiapi/configmaps/all
// ===============================

func GetAllConfigMaps() ([]corev1.ConfigMap, error) {
	var result []corev1.ConfigMap
	err := http.GetFromAgent("/agent/uiapi/configmaps/all", &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/configmaps/by-namespace/:ns
// ===============================

func GetConfigMapsByNamespace(namespace string) ([]corev1.ConfigMap, error) {
	var result []corev1.ConfigMap
	path := fmt.Sprintf("/agent/uiapi/configmaps/by-namespace/%s", namespace)
	err := http.GetFromAgent(path, &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/configmaps/detail/:ns/:name
// ===============================

func GetConfigMapDetail(namespace, name string) (*corev1.ConfigMap, error) {
	var result corev1.ConfigMap
	path := fmt.Sprintf("/agent/uiapi/configmaps/detail/%s/%s", namespace, name)
	err := http.GetFromAgent(path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

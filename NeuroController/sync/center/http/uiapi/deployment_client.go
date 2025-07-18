package uiapi

import (
	"NeuroController/sync/center/http"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
)

// ===============================
// 📌 GET /agent/uiapi/deployments/all
func GetAllDeployments() ([]appsv1.Deployment, error) {
	var result []appsv1.Deployment
	err := http.GetFromAgent("/agent/uiapi/deployments/all", &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/deployments/by-namespace/:ns
func GetDeploymentsByNamespace(namespace string) ([]appsv1.Deployment, error) {
	var result []appsv1.Deployment
	path := fmt.Sprintf("/agent/uiapi/deployments/by-namespace/%s", namespace)
	err := http.GetFromAgent(path, &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/deployments/detail/:ns/:name
func GetDeploymentByName(namespace, name string) (*appsv1.Deployment, error) {
	var result appsv1.Deployment
	path := fmt.Sprintf("/agent/uiapi/deployments/detail/%s/%s", namespace, name)
	err := http.GetFromAgent(path, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ===============================
// 📌 GET /agent/uiapi/deployments/unavailable
func GetUnavailableDeployments() ([]appsv1.Deployment, error) {
	var result []appsv1.Deployment
	err := http.GetFromAgent("/agent/uiapi/deployments/unavailable", &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/deployments/progressing
func GetProgressingDeployments() ([]appsv1.Deployment, error) {
	var result []appsv1.Deployment
	err := http.GetFromAgent("/agent/uiapi/deployments/progressing", &result)
	return result, err
}

// ===============================
// 📌 POST /agent/uiapi/deployments/scale/:ns/:name/:replicas
func UpdateDeploymentReplicas(namespace, name string, replicas int32) error {
	path := fmt.Sprintf("/agent/uiapi/deployments/scale/%s/%s/%d", namespace, name, replicas)
	return http.PostToAgent(path, nil)
}

// ===============================
// 📌 POST /agent/uiapi/deployments/image/:ns/:name/:image
func UpdateDeploymentImage(namespace, name, image string) error {
	path := fmt.Sprintf("/agent/uiapi/deployments/image/%s/%s/%s", namespace, name, image)
	return http.PostToAgent(path, nil)
}

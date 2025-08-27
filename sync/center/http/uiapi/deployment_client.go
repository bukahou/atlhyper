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
// func GetDeploymentsByNamespace(namespace string) ([]appsv1.Deployment, error) {
// 	var result []appsv1.Deployment
// 	path := fmt.Sprintf("/agent/uiapi/deployments/by-namespace/%s", namespace)
// 	err := http.GetFromAgent(path, &result)
// 	return result, err
// }

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
// func GetUnavailableDeployments() ([]appsv1.Deployment, error) {
// 	var result []appsv1.Deployment
// 	err := http.GetFromAgent("/agent/uiapi/deployments/unavailable", &result)
// 	return result, err
// }

// ===============================
// 📌 GET /agent/uiapi/deployments/progressing
// func GetProgressingDeployments() ([]appsv1.Deployment, error) {
// 	var result []appsv1.Deployment
// 	err := http.GetFromAgent("/agent/uiapi/deployments/progressing", &result)
// 	return result, err
// }

// ===============================
// 📌 POST /agent/uiapi/deployments/replicas
// UpdateDeploymentReplicas 修改指定 Deployment 的副本数
// 参数：
//   - namespace: Deployment 所在命名空间
//   - name: Deployment 名称
//   - replicas: 目标副本数（int32）
// 返回：
//   - error: 若失败则返回错误
// ===============================
func UpdateDeploymentReplicas(namespace, name string, replicas int32) error {
	req := map[string]interface{}{
		"namespace": namespace,
		"name":      name,
		"replicas":  replicas,
	}
	return http.PostToAgent("/agent/uiapi/deployments/replicas", req)
}

// ===============================
// 📌 POST /agent/uiapi/deployments/image
// UpdateDeploymentImage 更新指定 Deployment 的所有容器镜像
// 参数：
//   - namespace: Deployment 所在命名空间
//   - name: Deployment 名称
//   - image: 新的容器镜像名称
// 返回：
//   - error: 若失败则返回错误	
// ===============================
func UpdateDeploymentImage(namespace, name, image string) error {
	req := map[string]interface{}{
		"namespace": namespace,
		"name":      name,
		"image":     image,
	}
	return http.PostToAgent("/agent/uiapi/deployments/image", req)
}

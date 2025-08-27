package uiapi

import (
	"NeuroController/sync/center/http"

	corev1 "k8s.io/api/core/v1"
)

// ===============================
// ✅ GET /agent/uiapi/service/list
// 获取所有 Service
// ===============================
func GetAllServices() ([]corev1.Service, error) {
	var result []corev1.Service
	err := http.GetFromAgent("/agent/uiapi/service/list", &result)
	return result, err
}

// ===============================
// ✅ GET /agent/uiapi/service/list/by-namespace/:ns
// 根据命名空间获取 Service
// ===============================
// func GetServicesByNamespace(ns string) ([]corev1.Service, error) {
// 	var result []corev1.Service
// 	endpoint := fmt.Sprintf("/agent/uiapi/service/list/by-namespace/%s", url.PathEscape(ns))
// 	err := http.GetFromAgent(endpoint, &result)
// 	return result, err
// }

// ===============================
// ✅ GET /agent/uiapi/service/describe?namespace=xx&name=xx
// 获取指定 Service 详情
// ===============================
// func GetServiceByName(namespace, name string) (*corev1.Service, error) {
// 	var result corev1.Service
// 	endpoint := fmt.Sprintf("/agent/uiapi/service/describe?namespace=%s&name=%s",
// 		url.QueryEscape(namespace), url.QueryEscape(name))
// 	err := http.GetFromAgent(endpoint, &result)
// 	return &result, err
// }

// ===============================
// ✅ GET /agent/uiapi/service/list/external
// 获取对外暴露的 Service（类型为 NodePort / LoadBalancer）
// ===============================
// func GetExternalServices() ([]corev1.Service, error) {
// 	var result []corev1.Service
// 	err := http.GetFromAgent("/agent/uiapi/service/list/external", &result)
// 	return result, err
// }

// ===============================
// ✅ GET /agent/uiapi/service/list/headless
// 获取 Headless Service（clusterIP=None）
// ===============================
// func GetHeadlessServices() ([]corev1.Service, error) {
// 	var result []corev1.Service
// 	err := http.GetFromAgent("/agent/uiapi/service/list/headless", &result)
// 	return result, err
// }

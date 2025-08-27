package uiapi

import (
	"NeuroController/sync/center/http"

	networkingv1 "k8s.io/api/networking/v1"
)

// ===============================
// 📌 GET /agent/uiapi/ingress/list/all
// ===============================
func GetAllIngresses() ([]networkingv1.Ingress, error) {
	var result []networkingv1.Ingress
	err := http.GetFromAgent("/agent/uiapi/ingress/list/all", &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/ingress/list/by-namespace/:ns
// ===============================
// func GetIngressesByNamespace(namespace string) ([]networkingv1.Ingress, error) {
// 	var result []networkingv1.Ingress
// 	path := fmt.Sprintf("/agent/uiapi/ingress/list/by-namespace/%s", namespace)
// 	err := http.GetFromAgent(path, &result)
// 	return result, err
// }

// ===============================
// 📌 GET /agent/uiapi/ingress/detail/:ns/:name
// ===============================
// func GetIngressByName(namespace, name string) (*networkingv1.Ingress, error) {
// 	var result networkingv1.Ingress
// 	path := fmt.Sprintf("/agent/uiapi/ingress/detail/%s/%s", namespace, name)
// 	err := http.GetFromAgent(path, &result)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &result, nil
// }

// ===============================
// 📌 GET /agent/uiapi/ingress/list/ready
// ===============================
// func GetReadyIngresses() ([]networkingv1.Ingress, error) {
// 	var result []networkingv1.Ingress
// 	err := http.GetFromAgent("/agent/uiapi/ingress/list/ready", &result)
// 	return result, err
// }

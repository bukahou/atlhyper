package uiapi

import (
	"NeuroController/internal/query/namespace"
	"NeuroController/sync/center/http"
)

// ===============================
// 📌 GET /agent/uiapi/namespace/list
// ===============================
func GetAllNamespaces() ([]namespace.NamespaceWithPodCount, error) {
	// var result []corev1.Namespace
	var result []namespace.NamespaceWithPodCount

	err := http.GetFromAgent("/agent/uiapi/namespace/list", &result)
	return result, err
}

package uiapi

import (
	"NeuroController/sync/center/http"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// ===============================
// 📌 GET /agent/uiapi/event/list/all
// ===============================
func GetAllEvents() ([]corev1.Event, error) {
	var result []corev1.Event
	err := http.GetFromAgent("/agent/uiapi/event/list/all", &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/event/list/by-namespace/:ns
// ===============================
func GetEventsByNamespace(namespace string) ([]corev1.Event, error) {
	var result []corev1.Event
	path := fmt.Sprintf("/agent/uiapi/event/list/by-namespace/%s", namespace)
	err := http.GetFromAgent(path, &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/event/list/by-object/:ns/:kind/:name
// ===============================
func GetEventsByInvolvedObject(namespace, kind, name string) ([]corev1.Event, error) {
	var result []corev1.Event
	path := fmt.Sprintf("/agent/uiapi/event/list/by-object/%s/%s/%s", namespace, kind, name)
	err := http.GetFromAgent(path, &result)
	return result, err
}

// ===============================
// 📌 GET /agent/uiapi/event/stats/type-count
// ===============================
func GetEventTypeCounts() (map[string]int, error) {
	var result map[string]int
	err := http.GetFromAgent("/agent/uiapi/event/stats/type-count", &result)
	return result, err
}

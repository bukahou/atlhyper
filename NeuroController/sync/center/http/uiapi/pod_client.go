package uiapi

import (
	"NeuroController/sync/center/http"
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
)

// ✅ Pod 精简信息（复用 Agent 定义）
type PodInfo struct {
	Namespace    string `json:"namespace"`
	Deployment   string `json:"deployment"`
	Name         string `json:"name"`
	Ready        bool   `json:"ready"`
	Phase        string `json:"phase"`
	RestartCount int32  `json:"restartCount"`
	StartTime    string `json:"startTime"`
	PodIP        string `json:"podIP"`
	NodeName     string `json:"nodeName"`
}

// ✅ Pod 资源使用信息（复用 Agent 定义）
type PodUsage struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CPUUsage  int64  `json:"cpu_usage_millicores"`
	MemUsage  int64  `json:"mem_usage_bytes"`
}

// ✅ Pod Describe 信息（复用 Agent 定义）
type PodDescribeInfo struct {
	Pod     *corev1.Pod     `json:"pod"`
	Events  []corev1.Event  `json:"events"`
	Usage   *PodUsageInfo   `json:"usage,omitempty"`
	Service *corev1.Service `json:"service,omitempty"`
	Message string          `json:"message,omitempty"`
	Logs    string          `json:"logs,omitempty"`
}

type PodUsageInfo struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

//
// =======================================
// ✅ 获取所有 Pod（列表）
// GET /agent/uiapi/pod/list
// =======================================
func GetAllPods() ([]corev1.Pod, error) {
	var result []corev1.Pod
	err := http.GetFromAgent("/agent/uiapi/pod/list", &result)
	return result, err
}

//
// =======================================
// ✅ 获取指定命名空间下 Pod
// GET /agent/uiapi/pod/list/by-namespace/:ns
// =======================================
func GetPodsByNamespace(ns string) ([]corev1.Pod, error) {
	var result []corev1.Pod
	err := http.GetFromAgent("/agent/uiapi/pod/list/by-namespace/"+url.PathEscape(ns), &result)
	return result, err
}

//
// =======================================
// ✅ 获取 Pod 状态摘要
// GET /agent/uiapi/pod/summary
// =======================================
func GetPodStatusSummary() (map[string]int, error) {
	var result map[string]int
	err := http.GetFromAgent("/agent/uiapi/pod/summary", &result)
	return result, err
}

//
// =======================================
// ✅ 获取 Pod 资源使用情况
// GET /agent/uiapi/pod/usage
// =======================================
func GetPodUsages() ([]PodUsage, error) {
	var result []PodUsage
	err := http.GetFromAgent("/agent/uiapi/pod/usage", &result)
	return result, err
}

//
// =======================================
// ✅ 获取 Pod 精简信息
// GET /agent/uiapi/pod/infos
// =======================================
func GetAllPodInfos() ([]PodInfo, error) {
	var result []PodInfo
	err := http.GetFromAgent("/agent/uiapi/pod/infos", &result)
	return result, err
}

//
// =======================================
// ✅ 获取 Pod 描述信息（状态 + event + usage + logs + svc）
// GET /agent/uiapi/pod/describe?namespace=xx&name=xx
// =======================================
func GetPodDescribe(namespace, name string) (*PodDescribeInfo, error) {
	var result PodDescribeInfo
	endpoint := fmt.Sprintf("/agent/uiapi/pod/describe?namespace=%s&name=%s",
		url.QueryEscape(namespace), url.QueryEscape(name))
	err := http.GetFromAgent(endpoint, &result)
	return &result, err
}

//
// =======================================
// ✅ 重启 Pod（实际上是删除）
// POST /agent/uiapi/pod/restart
// =======================================
func RestartPod(namespace, name string) error {
	form := url.Values{}
	form.Set("namespace", namespace)
	form.Set("name", name)
	return http.PostFormToAgent("/agent/uiapi/pod/restart", form, nil)
}

//
// =======================================
// ✅ 获取 Pod 日志
// GET /agent/uiapi/pod/logs?namespace=xx&name=xx&container=xx&tailLines=100
// =======================================
func GetPodLogs(namespace, name, container string, tailLines int64) (string, error) {
	endpoint := fmt.Sprintf("/agent/uiapi/pod/logs?namespace=%s&name=%s&container=%s&tailLines=%d",
		url.QueryEscape(namespace), url.QueryEscape(name), url.QueryEscape(container), tailLines)

	var result string
	err := http.GetTextFromAgent(endpoint, &result)
	return result, err
}

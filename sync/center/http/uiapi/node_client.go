package uiapi

import (
	"NeuroController/model"
	"NeuroController/sync/center/http"
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// =============================================
// 📦 模型定义（与 Agent 返回结构保持一致）
// =============================================

type NodeOverviewStats struct {
	TotalNodes    int     `json:"totalNodes"`
	ReadyNodes    int     `json:"readyNodes"`
	TotalCPU      int     `json:"totalCPU"`
	TotalMemoryGB float64 `json:"totalMemoryGB"`
}

type NodeBrief struct {
	Name       string            `json:"name"`
	Ready      bool              `json:"ready"`
	InternalIP string            `json:"internalIP"`
	OSImage    string            `json:"osImage"`
	Arch       string            `json:"architecture"`
	CPU        int               `json:"cpu"`
	MemoryGB   float64           `json:"memory"`
	Labels     map[string]string `json:"labels"`
	Unschedulable  bool              `json:"unschedulable"`
}

type NodeOverviewResult struct {
	Stats NodeOverviewStats `json:"stats"`
	Nodes []NodeBrief       `json:"nodes"`
}

// NodeMetricsSummary 结构（与 Agent 保持一致）
type NodeMetricsSummary struct {
	AvgCPUUsagePercent    float64 `json:"AvgCPUUsagePercent"`
	AvgMemoryUsagePercent float64 `json:"AvgMemoryUsagePercent"`
	DiskPressureCount     int     `json:"DiskPressureCount"`
}

//
// =============================================
// ✅ GET /agent/uiapi/node/overview
// =============================================
func GetNodeOverview() (*NodeOverviewResult, error) {
	var result NodeOverviewResult
	err := http.GetFromAgent("/agent/uiapi/node/overview", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

//
// =============================================
// ✅ GET /agent/uiapi/node/list
// =============================================
func GetAllNodes() ([]corev1.Node, error) {
	var result []corev1.Node
	err := http.GetFromAgent("/agent/uiapi/node/list", &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

//
// =============================================
// ✅ GET /agent/uiapi/node/get/:name
// =============================================
func GetNodeDetail(name string) (*model.NodeDetailInfo, error) {
	var result model.NodeDetailInfo
	url := fmt.Sprintf("/agent/uiapi/node/get/%s", name)
	err := http.GetFromAgent(url, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

//
// =============================================
// ✅ GET /agent/uiapi/node/metrics-summary
// =============================================
func GetNodeMetricsSummary() (*NodeMetricsSummary, error) {
	var result NodeMetricsSummary
	err := http.GetFromAgent("/agent/uiapi/node/metrics-summary", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SetNodeSchedulable 向 Agent 发送调度状态切换请求（封锁或解封 Node）
// 参数：
//   - name: 节点名称
//   - unschedulable: true 表示封锁（cordon），false 表示解封（uncordon）
// 返回：
//   - error: 如果请求失败或 Agent 返回错误，将返回详细错误信息
func SetNodeSchedulable(name string, unschedulable bool) error {
	type toggleNodeRequest struct {
		Name          string `json:"name"`
		Unschedulable bool   `json:"unschedulable"`
	}

	payload := toggleNodeRequest{
		Name:          name,
		Unschedulable: unschedulable,
	}

	// ✅ 调整为 2 个参数调用
	err := http.PostToAgent("/agent/uiapi/node/schedulable", payload)
	if err != nil {
		return fmt.Errorf("向 Agent 发送调度切换请求失败: %w", err)
	}

	return nil
}


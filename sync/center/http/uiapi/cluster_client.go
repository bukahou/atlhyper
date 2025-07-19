package uiapi

import (
	"NeuroController/sync/center/http"
)

// ClusterOverview 定义了集群概要数据的结构，用于首页 UI 展示
type ClusterOverview struct {
	TotalNodes   int    `json:"total_nodes"`        // 节点总数
	ReadyNodes   int    `json:"ready_nodes"`        // Ready 状态的节点数
	TotalPods    int    `json:"total_pods"`         // 所有命名空间中的 Pod 总数
	AbnormalPods int    `json:"abnormal_pods"`      // 异常状态（Pending/Failed/Unknown）的 Pod 数量
	K8sVersion   string `json:"k8s_version"`        // Kubernetes 控制平面版本
	HasMetrics   bool   `json:"has_metrics_server"` // 是否检测到 metrics-server
}

// ✅ 获取集群概要信息（只从第一个 Agent 获取）
func GetClusterOverview() (*ClusterOverview, error) {
	var result ClusterOverview
	err := http.GetFromAgent("/agent/uiapi/cluster/overview", &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

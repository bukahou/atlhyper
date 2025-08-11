package master_metrics

import (
	model "NeuroController/model/metrics"
	"NeuroController/sync/center/http"
)

// GetLatestNodeMetrics 从 agent 获取所有节点的最新指标数据
func GetLatestNodeMetrics() (map[string][]*model.NodeMetricsSnapshot, error) {
	var result map[string][]*model.NodeMetricsSnapshot
	err := http.GetFromAgent("/agent/dataapi/latest", &result)
	return result, err
}
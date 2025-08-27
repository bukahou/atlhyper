// internal/data_api/metrics.go
package data_api

import (
	agentstore "NeuroController/internal/agent_store"
	nmetrics "NeuroController/model/metrics"
)

func GetAllLatestNodeMetrics() map[string]nmetrics.NodeMetricsSnapshot {
	m := agentstore.GetAllLatestCopy()

	return m
}

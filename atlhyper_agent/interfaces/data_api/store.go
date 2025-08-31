// internal/data_api/metrics.go
package data_api

import (
	agentstore "AtlHyper/atlhyper_agent/agent_store"
	nmetrics "AtlHyper/model/metrics"
)

func GetAllLatestNodeMetrics() map[string]nmetrics.NodeMetricsSnapshot {
	m := agentstore.GetAllLatestCopy()

	return m
}

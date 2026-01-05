// internal/data_api/metrics.go
package data_api

import (
	agentstore "AtlHyper/atlhyper_agent/agent_store"
	"AtlHyper/model/collect"
)

func GetAllLatestNodeMetrics() map[string]collect.NodeMetricsSnapshot {
	m := agentstore.GetAllLatestCopy()

	return m
}

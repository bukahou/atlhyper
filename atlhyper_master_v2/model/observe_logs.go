// atlhyper_master_v2/model/observe_logs.go
// Observe Logs 查询参数与结果类型
package model

import "AtlHyper/model_v3/log"

// LogSnapshotQueryOpts 快照日志查询选项
type LogSnapshotQueryOpts struct {
	Service   string
	Level     string
	Scope     string
	StartTime string
	EndTime   string
	Offset    int
	Limit     int
}

// LogSnapshotResult 快照日志查询结果
type LogSnapshotResult struct {
	Logs   []log.Entry
	Total  int
	Facets log.Facets
}

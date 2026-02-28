// atlhyper_master_v2/service/query/observe_logs.go
// 快照日志查询实现（过滤 + 分页 + facets）
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/log"
)

// QueryLogsFromSnapshot 从快照查询日志（过滤+分页+facets）
//
// facets 基于全量数据（过滤前）计算。
// 过滤按 service/level/scope/时间范围执行，然后分页。
func (q *QueryService) QueryLogsFromSnapshot(ctx context.Context, clusterID string, opts model.LogSnapshotQueryOpts) (*model.LogSnapshotResult, error) {
	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil || otel == nil || len(otel.RecentLogs) == 0 {
		return nil, err
	}

	// facets 基于全量数据（过滤前）
	facets := computeLogFacets(otel.RecentLogs)

	logs := otel.RecentLogs

	// 按 service 过滤
	if opts.Service != "" {
		filtered := logs[:0:0]
		for _, l := range logs {
			if l.ServiceName == opts.Service {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	// 按 level 过滤
	if opts.Level != "" {
		filtered := logs[:0:0]
		for _, l := range logs {
			if l.Severity == opts.Level {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	// 按 scope 过滤
	if opts.Scope != "" {
		filtered := logs[:0:0]
		for _, l := range logs {
			if l.ScopeName == opts.Scope {
				filtered = append(filtered, l)
			}
		}
		logs = filtered
	}

	// 按时间范围过滤
	if opts.StartTime != "" {
		if startT, err := time.Parse(time.RFC3339Nano, opts.StartTime); err == nil {
			filtered := logs[:0:0]
			for _, l := range logs {
				if !l.Timestamp.Before(startT) {
					filtered = append(filtered, l)
				}
			}
			logs = filtered
		}
	}
	if opts.EndTime != "" {
		if endT, err := time.Parse(time.RFC3339Nano, opts.EndTime); err == nil {
			filtered := logs[:0:0]
			for _, l := range logs {
				if !l.Timestamp.After(endT) {
					filtered = append(filtered, l)
				}
			}
			logs = filtered
		}
	}

	// 分页
	total := len(logs)
	offset := opts.Offset
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if offset >= total {
		logs = logs[:0]
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		logs = logs[offset:end]
	}

	return &model.LogSnapshotResult{
		Logs:   logs,
		Total:  total,
		Facets: facets,
	}, nil
}

// computeLogFacets 从全量日志计算 serviceName / severity / scopeName 分面统计
func computeLogFacets(logs []log.Entry) log.Facets {
	svcMap := make(map[string]int64)
	sevMap := make(map[string]int64)
	scopeMap := make(map[string]int64)
	for i := range logs {
		svcMap[logs[i].ServiceName]++
		sevMap[logs[i].Severity]++
		scopeMap[logs[i].ScopeName]++
	}
	toFacets := func(m map[string]int64) []log.Facet {
		out := make([]log.Facet, 0, len(m))
		for v, c := range m {
			if v != "" {
				out = append(out, log.Facet{Value: v, Count: c})
			}
		}
		return out
	}
	return log.Facets{
		Services:   toFacets(svcMap),
		Severities: toFacets(sevMap),
		Scopes:     toFacets(scopeMap),
	}
}

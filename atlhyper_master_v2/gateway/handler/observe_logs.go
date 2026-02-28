// atlhyper_master_v2/gateway/handler/observe_logs.go
// Logs 信号域 Handler 方法
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"AtlHyper/model_v3/command"
	"AtlHyper/model_v3/log"
)

// LogsQuery POST /api/v2/observe/logs/query
//
// 简单查询（无全文搜索）→ 快照直读 RecentLogs
// 全文搜索 → Command/MQ 透传 Agent
func (h *ObserveHandler) LogsQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	clusterID, _ := body["cluster_id"].(string)
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	query, _ := body["query"].(string)
	traceId, _ := body["trace_id"].(string)
	spanId, _ := body["span_id"].(string)

	// 快速路径：无全文搜索 且 无 trace/span 关联查询时从快照 RecentLogs 直读
	if query == "" && traceId == "" && spanId == "" {
		otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
		if err == nil && otel != nil && len(otel.RecentLogs) > 0 {
			// facets 基于全量数据（过滤前）
			facets := computeLogFacets(otel.RecentLogs)

			logs := otel.RecentLogs
			// 按 service 过滤
			if svc, _ := body["service"].(string); svc != "" {
				filtered := logs[:0:0]
				for _, l := range logs {
					if l.ServiceName == svc {
						filtered = append(filtered, l)
					}
				}
				logs = filtered
			}
			// 按 level 过滤
			if level, _ := body["level"].(string); level != "" {
				filtered := logs[:0:0]
				for _, l := range logs {
					if l.Severity == level {
						filtered = append(filtered, l)
					}
				}
				logs = filtered
			}
			// 按 scope 过滤
			if scope, _ := body["scope"].(string); scope != "" {
				filtered := logs[:0:0]
				for _, l := range logs {
					if l.ScopeName == scope {
						filtered = append(filtered, l)
					}
				}
				logs = filtered
			}

			// histogram 基于过滤后全量数据（时间范围过滤和分页之前）
			histogram := extractLogHistogram(logs)

			// 按时间范围过滤（brush 选区，histogram 之后、分页之前）
			if startStr, _ := body["start_time"].(string); startStr != "" {
				if startT, err := time.Parse(time.RFC3339Nano, startStr); err == nil {
					filtered := logs[:0:0]
					for _, l := range logs {
						if !l.Timestamp.Before(startT) {
							filtered = append(filtered, l)
						}
					}
					logs = filtered
				}
			}
			if endStr, _ := body["end_time"].(string); endStr != "" {
				if endT, err := time.Parse(time.RFC3339Nano, endStr); err == nil {
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
			offset := 0
			limit := 50
			if v, ok := body["offset"].(float64); ok && v > 0 {
				offset = int(v)
			}
			if v, ok := body["limit"].(float64); ok && v > 0 {
				limit = int(v)
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
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data": map[string]interface{}{
					"logs":      logs,
					"total":     total,
					"facets":    facets,
					"histogram": histogram,
				},
			})
			return
		}
	}

	// 全文搜索 → Command/MQ
	delete(body, "cluster_id")
	h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
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

// extractLogHistogram 提取过滤后全量日志的 timestamp + severity（供前端绘制柱状图）
func extractLogHistogram(logs []log.Entry) []map[string]string {
	out := make([]map[string]string, len(logs))
	for i := range logs {
		out[i] = map[string]string{
			"timestamp": logs[i].Timestamp.Format(time.RFC3339Nano),
			"severity":  logs[i].Severity,
		}
	}
	return out
}

// LogsSummary GET /api/v2/observe/logs/summary (Dashboard: 快照直读)
func (h *ObserveHandler) LogsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.LogsSummary == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.LogsSummary,
	})
}

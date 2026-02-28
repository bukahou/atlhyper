// atlhyper_master_v2/gateway/handler/observe_logs.go
// Logs 信号域 Handler 方法
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/command"
)

// LogsQuery POST /api/v2/observe/logs/query
//
// 简单查询（无全文搜索）→ Service 层快照直读
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
	startTime, _ := body["start_time"].(string)
	endTime, _ := body["end_time"].(string)

	// 快速路径：无全文搜索、无 trace/span 关联、无绝对时间过滤时从快照直读
	if query == "" && traceId == "" && spanId == "" && startTime == "" && endTime == "" {
		svc, _ := body["service"].(string)
		level, _ := body["level"].(string)
		scope, _ := body["scope"].(string)
		offset := 0
		limit := 50
		if v, ok := body["offset"].(float64); ok && v > 0 {
			offset = int(v)
		}
		if v, ok := body["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}

		result, err := h.querySvc.QueryLogsFromSnapshot(r.Context(), clusterID, model.LogSnapshotQueryOpts{
			Service: svc,
			Level:   level,
			Scope:   scope,
			Offset:  offset,
			Limit:   limit,
		})
		if err == nil && result != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"message": "获取成功",
				"data": map[string]interface{}{
					"logs":   result.Logs,
					"total":  result.Total,
					"facets": result.Facets,
				},
			})
			return
		}
	}

	// 全文搜索 → Command/MQ
	delete(body, "cluster_id")
	h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
}

// LogsHistogram GET /api/v2/observe/logs/histogram
//
// 直方图始终走 ClickHouse 聚合查询，返回 ~30 个预聚合桶
func (h *ObserveHandler) LogsHistogram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	q := r.URL.Query()
	params := map[string]interface{}{
		"sub_action": "histogram",
	}
	if v := q.Get("since"); v != "" {
		params["since"] = v
	}
	if v := q.Get("service"); v != "" {
		params["service"] = v
	}
	if v := q.Get("level"); v != "" {
		params["level"] = v
	}
	if v := q.Get("scope"); v != "" {
		params["scope"] = v
	}
	if v := q.Get("query"); v != "" {
		params["query"] = v
	}
	if v := q.Get("start_time"); v != "" {
		params["start_time"] = v
	}
	if v := q.Get("end_time"); v != "" {
		params["end_time"] = v
	}

	// cacheTTL 根据时间范围决定
	minutes := 15
	if since := q.Get("since"); since != "" {
		if m, valid := parseTimeRangeMinutes(since); valid {
			minutes = m
		}
	} else if st := q.Get("start_time"); st != "" {
		// 绝对时间：根据跨度计算 cacheTTL
		if startT, err := time.Parse(time.RFC3339Nano, st); err == nil {
			if et := q.Get("end_time"); et != "" {
				if endT, err := time.Parse(time.RFC3339Nano, et); err == nil {
					minutes = int(endT.Sub(startT).Minutes())
				}
			}
		}
	}

	h.executeQuery(w, r, clusterID, command.ActionQueryLogs, params, cacheTTLForMinutes(minutes))
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

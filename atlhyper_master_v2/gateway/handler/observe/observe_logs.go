// atlhyper_master_v2/gateway/handler/observe_logs.go
// Logs 信号域 Handler 方法
package observe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/model_v3/command"
)

// LogsQuery POST /api/v2/observe/logs/query
//
// 所有日志查询统一走 Command → Agent → ClickHouse（Kibana 模式）
// 日志不缓存在 Master 内存中，按需实时查询
func (h *ObserveHandler) LogsQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 请求体只认这些字段：查询条件原样下发给 Agent，⛔ 不把浏览器给的任意键带过去
	var body struct {
		ClusterID string `json:"cluster_id"`
		command.QueryLogsParams
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.ClusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}
	body.SubAction = command.LogsList
	h.executeQuery(w, r, body.ClusterID, command.ActionQueryLogs, body.QueryLogsParams, 0)
}

// LogsHistogram GET /api/v2/observe/logs/histogram
//
// 直方图始终走 ClickHouse 聚合查询，返回 ~30 个预聚合桶
func (h *ObserveHandler) LogsHistogram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	q := r.URL.Query()
	params := command.QueryLogsParams{
		SubAction:  command.LogsHistogram,
		Service:    q.Get("service"),
		Level:      q.Get("level"),
		Scope:      q.Get("scope"),
		Query:      q.Get("query"),
		TimeWindow: command.TimeWindow{StartTime: q.Get("start_time"), EndTime: q.Get("end_time")},
	}
	if v := q.Get("since"); v != "" {
		m, ok := parseTimeRangeMinutes(v)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "since 格式不合法（如 15m、1h、24h）")
			return
		}
		params.Since = fmt.Sprintf("%dm", m)
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
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.LogsSummary == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.LogsSummary,
	})
}

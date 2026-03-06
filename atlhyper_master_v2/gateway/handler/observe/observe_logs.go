// atlhyper_master_v2/gateway/handler/observe_logs.go
// Logs 信号域 Handler 方法
package observe

import (
	"encoding/json"
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

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	clusterID, _ := body["cluster_id"].(string)
	if clusterID == "" {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	delete(body, "cluster_id")
	h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
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

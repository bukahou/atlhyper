// atlhyper_master_v2/gateway/handler/observe_apm.go
// APM 信号域 Handler 方法（Traces / Services / Stats / TimeSeries）
package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/model_v3/command"
)

// TracesList GET /api/v2/observe/traces
// 默认 15m 从快照直读；自定义时间走 Command/MQ 查询 ClickHouse
func (h *ObserveHandler) TracesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 自定义时间范围 → Command/MQ
	timeRange := r.URL.Query().Get("time_range")
	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := map[string]interface{}{
			"sub_action": "list_traces",
			"since":      fmt.Sprintf("%dm", minutes),
			"limit":      500,
		}
		if svc := r.URL.Query().Get("service"); svc != "" {
			params["service"] = svc
		}
		if op := r.URL.Query().Get("operation"); op != "" {
			params["operation"] = op
		}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || len(otel.RecentTraces) == 0 {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	traces := otel.RecentTraces

	// 客户端过滤
	if svc := r.URL.Query().Get("service"); svc != "" {
		filtered := traces[:0:0]
		for _, t := range traces {
			if t.RootService == svc {
				filtered = append(filtered, t)
			}
		}
		traces = filtered
	}
	if op := r.URL.Query().Get("operation"); op != "" {
		filtered := traces[:0:0]
		for _, t := range traces {
			if strings.Contains(t.RootOperation, op) {
				filtered = append(filtered, t)
			}
		}
		traces = filtered
	}
	if v := r.URL.Query().Get("min_duration"); v != "" {
		if minMs, err := strconv.ParseFloat(v, 64); err == nil {
			filtered := traces[:0:0]
			for _, t := range traces {
				if t.DurationMs >= minMs {
					filtered = append(filtered, t)
				}
			}
			traces = filtered
		}
	}

	// 分页
	total := len(traces)
	offset := 0
	limit := total
	if v := r.URL.Query().Get("offset"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			offset = i
		}
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			limit = i
		}
	}
	if offset >= total {
		traces = nil
	} else {
		end := offset + limit
		if end > total {
			end = total
		}
		traces = traces[offset:end]
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data": map[string]interface{}{
			"traces": traces,
			"total":  total,
		},
	})
}

// TracesServices GET /api/v2/observe/traces/services (Dashboard: 快照直读 / 自定义时间: Command)
func (h *ObserveHandler) TracesServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := map[string]interface{}{"sub_action": "list_services", "since": fmt.Sprintf("%dm", minutes)}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.APMServices == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.APMServices,
	})
}

// TracesTopology GET /api/v2/observe/traces/topology (Dashboard: 快照直读 / 自定义时间: Command)
func (h *ObserveHandler) TracesTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := map[string]interface{}{"sub_action": "get_topology", "since": fmt.Sprintf("%dm", minutes)}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.APMTopology == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.APMTopology,
	})
}

// TracesOperations GET /api/v2/observe/traces/operations (Dashboard: 快照直读 / 自定义时间: Command)
// 返回操作级聚合统计，支持 ?service=xxx 过滤
func (h *ObserveHandler) TracesOperations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := map[string]interface{}{"sub_action": "list_operations", "since": fmt.Sprintf("%dm", minutes)}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.APMOperations == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	ops := otel.APMOperations

	// 按 service 过滤
	if svc := r.URL.Query().Get("service"); svc != "" {
		filtered := ops[:0:0]
		for _, o := range ops {
			if o.ServiceName == svc {
				filtered = append(filtered, o)
			}
		}
		ops = filtered
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    ops,
	})
}

// TracesDetail GET /api/v2/observe/traces/{id}
func (h *ObserveHandler) TracesDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析 trace ID
	traceID := strings.TrimPrefix(r.URL.Path, "/api/v2/observe/traces/")
	traceID = strings.TrimSuffix(traceID, "/")
	if traceID == "" {
		writeError(w, http.StatusBadRequest, "trace_id is required")
		return
	}

	params := map[string]interface{}{
		"trace_id": traceID,
	}

	h.executeQuery(w, r, clusterID, command.ActionQueryTraceDetail, params, 30*time.Second)
}

// TracesStats GET /api/v2/observe/traces/stats
// 通过 Command/MQ 查询 HTTP 状态码分布和数据库操作统计
func (h *ObserveHandler) TracesStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	subAction := r.URL.Query().Get("sub_action")
	service := r.URL.Query().Get("service")
	if service == "" {
		writeError(w, http.StatusBadRequest, "service is required")
		return
	}

	minutes := 15
	if tr := r.URL.Query().Get("time_range"); tr != "" {
		if m, valid := parseTimeRangeMinutes(tr); valid {
			minutes = m
		}
	}

	params := map[string]interface{}{
		"sub_action": subAction,
		"service":    service,
		"since":      fmt.Sprintf("%dm", minutes),
	}

	h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
}

// APMServiceSeries GET /api/v2/observe/traces/services/{name}/series
// ≤60min: 从 Concentrator 预聚合读取；>60min: Command/MQ 查询 ClickHouse
func (h *ObserveHandler) APMServiceSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析路径: /api/v2/observe/traces/services/{name}/series
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/observe/traces/services/")
	path = strings.TrimSuffix(path, "/series")
	path = strings.TrimSuffix(path, "/")
	serviceName := path
	if serviceName == "" {
		writeError(w, http.StatusBadRequest, "service name is required")
		return
	}

	minutes := 60
	if v := r.URL.Query().Get("minutes"); v != "" {
		if m, err := strconv.Atoi(v); err == nil && m > 0 {
			minutes = m
		}
	}

	// >60min: 走 Command/MQ 查询 ClickHouse（Concentrator 只存 60 分钟）
	if minutes > 60 {
		params := map[string]interface{}{
			"sub_action": "service_series",
			"service":    serviceName,
			"since":      fmt.Sprintf("%dm", minutes),
		}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	// ≤60min: 优先从 Concentrator 预聚合读取，无数据时回退 ClickHouse
	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err == nil && otel != nil && otel.APMTimeSeries != nil {
		for _, s := range otel.APMTimeSeries {
			if s.ServiceName == serviceName {
				points := filterAPMPointsByMinutes(s.Points, minutes)
				if len(points) > 0 {
					writeJSON(w, http.StatusOK, map[string]interface{}{
						"message": "获取成功",
						"data": map[string]interface{}{
							"service":   serviceName,
							"namespace": s.Namespace,
							"points":    points,
						},
					})
					return
				}
			}
		}
	}

	// Concentrator 无数据（如最近无流量），回退到 ClickHouse 查询
	params := map[string]interface{}{
		"sub_action": "service_series",
		"service":    serviceName,
		"since":      fmt.Sprintf("%dm", minutes),
	}
	h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
}

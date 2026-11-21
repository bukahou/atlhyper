// atlhyper_master_v2/gateway/handler/observe_apm.go
// APM 信号域 Handler 方法（Traces / Services / Stats / TimeSeries）
package observe

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/model_v3/command"
)

// TracesList GET /api/v2/observe/traces
// 默认 15m 从快照直读；自定义时间走 Command/MQ 查询 ClickHouse
func (h *ObserveHandler) TracesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 绝对时间 / 自定义时间范围 → Command/MQ
	timeRange := r.URL.Query().Get("time_range")
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")

	if startTime != "" && endTime != "" {
		params := traceListParams(r)
		params.TimeWindow = command.TimeWindow{StartTime: startTime, EndTime: endTime}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 30*time.Second)
		return
	}

	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := traceListParams(r)
		params.TimeWindow = command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || len(otel.RecentTraces) == 0 {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	traces := otel.RecentTraces

	// 客户端过滤
	if svc := r.URL.Query().Get("service"); svc != "" {
		filtered := traces[:0:0]
		for _, t := range traces {
			if sliceContains(t.Services, svc) {
				filtered = append(filtered, t)
			}
		}
		traces = filtered
	}
	// 快照路径不按 operation 过滤：RootOperation 是入口服务的操作名，
	// 子服务 operation 与之不同会被误过滤。Service 过滤已缩小范围足够。
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

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")

	if startTime != "" && endTime != "" {
		params := command.QueryTracesParams{SubAction: command.TraceListServices, TimeWindow: command.TimeWindow{StartTime: startTime, EndTime: endTime}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 30*time.Second)
		return
	}

	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := command.QueryTracesParams{SubAction: command.TraceListServices, TimeWindow: command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.APMServices == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.APMServices,
	})
}

// TracesTopology GET /api/v2/observe/traces/topology (Dashboard: 快照直读 / 自定义时间: Command)
func (h *ObserveHandler) TracesTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")

	if startTime != "" && endTime != "" {
		params := command.QueryTracesParams{SubAction: command.TraceTopology, TimeWindow: command.TimeWindow{StartTime: startTime, EndTime: endTime}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 30*time.Second)
		return
	}

	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := command.QueryTracesParams{SubAction: command.TraceTopology, TimeWindow: command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.APMTopology == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.APMTopology,
	})
}

// TracesOperations GET /api/v2/observe/traces/operations (Dashboard: 快照直读 / 自定义时间: Command)
// 返回操作级聚合统计，支持 ?service=xxx 过滤
func (h *ObserveHandler) TracesOperations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")

	if startTime != "" && endTime != "" {
		params := command.QueryTracesParams{SubAction: command.TraceListOperations, TimeWindow: command.TimeWindow{StartTime: startTime, EndTime: endTime}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 30*time.Second)
		return
	}

	if timeRange != "" && timeRange != "15m" {
		minutes, ok := parseTimeRangeMinutes(timeRange)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params := command.QueryTracesParams{SubAction: command.TraceListOperations, TimeWindow: command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
		return
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil || otel.APMOperations == nil {
		handler.WriteError(w, http.StatusNotFound, "数据尚未就绪")
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

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    ops,
	})
}

// TracesDetail GET /api/v2/observe/traces/{id}
func (h *ObserveHandler) TracesDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析 trace ID
	traceID := strings.TrimPrefix(r.URL.Path, "/api/v2/observe/traces/")
	traceID = strings.TrimSuffix(traceID, "/")
	if traceID == "" {
		handler.WriteError(w, http.StatusBadRequest, "trace_id is required")
		return
	}

	h.executeQuery(w, r, clusterID, command.ActionQueryTraceDetail, command.QueryTraceDetailParams{TraceID: traceID}, 30*time.Second)
}

// TracesCorrelations GET /api/v2/observe/traces/correlations
//
// 相关性分析（对齐 Elastic APM Correlations）：慢/错的入口 span 相比
// 全量，哪些属性值显著超标。mode = latency | failure（默认 failure）。
func (h *ObserveHandler) TracesCorrelations(w http.ResponseWriter, r *http.Request) {
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
	params := command.QueryTracesParams{
		SubAction:  command.TraceCorrelations,
		Service:    q.Get("service"),
		Operation:  q.Get("operation"),
		Mode:       q.Get("mode"),
		TimeWindow: command.TimeWindow{StartTime: q.Get("start_time"), EndTime: q.Get("end_time")},
	}
	// 与其他观测接口同一个时间窗上限
	if tr := q.Get("time_range"); tr != "" {
		minutes, ok := parseTimeRangeMinutes(tr)
		if !ok {
			handler.WriteError(w, http.StatusBadRequest, "invalid time_range")
			return
		}
		params.Since = fmt.Sprintf("%dm", minutes)
	}
	// 统计结果短缓存：相关性分析是重查询，同参数 30 秒内不重算
	h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 30*time.Second)
}

// TracesStats GET /api/v2/observe/traces/stats
// 通过 Command/MQ 查询 HTTP 状态码分布和数据库操作统计
func (h *ObserveHandler) TracesStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	subAction := r.URL.Query().Get("sub_action")
	if subAction != command.TraceHTTPStats && subAction != command.TraceDBStats {
		handler.WriteError(w, http.StatusBadRequest, "sub_action 须为 http_stats 或 db_stats")
		return
	}
	service := r.URL.Query().Get("service")
	if service == "" {
		handler.WriteError(w, http.StatusBadRequest, "service is required")
		return
	}

	startTime := r.URL.Query().Get("start_time")
	endTime := r.URL.Query().Get("end_time")

	if startTime != "" && endTime != "" {
		params := command.QueryTracesParams{SubAction: subAction, Service: service, TimeWindow: command.TimeWindow{StartTime: startTime, EndTime: endTime}}
		h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 30*time.Second)
		return
	}

	minutes := 15
	if tr := r.URL.Query().Get("time_range"); tr != "" {
		if m, valid := parseTimeRangeMinutes(tr); valid {
			minutes = m
		}
	}

	params := command.QueryTracesParams{SubAction: subAction, Service: service, TimeWindow: command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}}

	h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
}

// APMServiceSeries GET /api/v2/observe/traces/services/{name}/series
// ≤60min: 从 Concentrator 预聚合读取；>60min: Command/MQ 查询 ClickHouse
func (h *ObserveHandler) APMServiceSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		handler.WriteError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析路径: /api/v2/observe/traces/services/{name}/series
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/observe/traces/services/")
	path = strings.TrimSuffix(path, "/series")
	path = strings.TrimSuffix(path, "/")
	serviceName := path
	if serviceName == "" {
		handler.WriteError(w, http.StatusBadRequest, "service name is required")
		return
	}

	minutes := windowMinutesParam(r, "minutes", 60)

	// >60min: 走 Command/MQ 查询 ClickHouse（Concentrator 只存 60 分钟）
	if minutes > 60 {
		params := command.QueryTracesParams{SubAction: command.TraceServiceSeries, Service: serviceName, TimeWindow: command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}}
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
					handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
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
	params := command.QueryTracesParams{SubAction: command.TraceServiceSeries, Service: serviceName, TimeWindow: command.TimeWindow{Since: fmt.Sprintf("%dm", minutes)}}
	h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, cacheTTLForMinutes(minutes))
}

// traceListParams 链路列表的筛选条件（时间范围由调用方补）
func traceListParams(r *http.Request) command.QueryTracesParams {
	q := r.URL.Query()
	return command.QueryTracesParams{
		SubAction:  command.TraceListTraces,
		Limit:      500,
		Service:    q.Get("service"),
		Operation:  q.Get("operation"),
		StatusCode: q.Get("status_code"),
		Method:     q.Get("method"),
	}
}

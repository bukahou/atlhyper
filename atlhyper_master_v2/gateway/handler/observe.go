// atlhyper_master_v2/gateway/handler/observe.go
// 可观测性查询 Handler（Traces / Logs / Metrics / SLO）
//
// 13 个 Dashboard 端点从快照直读（O(1)，<10ms），
// 仅 2 个端点保留 Command 机制：TracesDetail（Trace 详情）和 LogsQuery（日志搜索）。
package handler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/model_v3/command"
)

// ObserveHandler 可观测性查询 Handler
//
// Dashboard + Detail 端点（13 个）直读快照/预聚合时序，
// 仅 TracesDetail + LogsQuery（2 个）保留 Command 机制。
type ObserveHandler struct {
	svc      service.Ops
	querySvc service.Query
	bus      mq.Producer
	cache    *observeCache
}

// NewObserveHandler 创建 ObserveHandler
func NewObserveHandler(svc service.Ops, querySvc service.Query, bus mq.Producer) *ObserveHandler {
	return &ObserveHandler{
		svc:      svc,
		querySvc: querySvc,
		bus:      bus,
		cache:    newObserveCache(),
	}
}

// ================================================================
// TTL 缓存
// ================================================================

type cacheEntry struct {
	data      json.RawMessage
	expiresAt time.Time
}

type observeCache struct {
	mu    sync.RWMutex
	items map[string]*cacheEntry
}

func newObserveCache() *observeCache {
	c := &observeCache{items: make(map[string]*cacheEntry)}
	// 后台清理过期条目
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c.cleanup()
		}
	}()
	return c
}

func (c *observeCache) get(key string) (json.RawMessage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *observeCache) set(key string, data json.RawMessage, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &cacheEntry{data: data, expiresAt: time.Now().Add(ttl)}
}

func (c *observeCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.expiresAt) {
			delete(c.items, k)
		}
	}
}

// ================================================================
// 统一执行方法
// ================================================================

func (h *ObserveHandler) executeQuery(
	w http.ResponseWriter, r *http.Request,
	clusterID, action string,
	params map[string]interface{},
	cacheTTL time.Duration,
) {
	// 1. 检查缓存
	cacheKey := buildCacheKey(clusterID, action, params)
	if data, ok := h.cache.get(cacheKey); ok {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "获取成功",
			"data":    data,
		})
		return
	}

	// 2. 创建指令
	resp, err := h.svc.CreateCommand(&operations.CreateCommandRequest{
		ClusterID: clusterID,
		Action:    action,
		Params:    params,
		Source:    "web",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建查询指令失败: "+err.Error())
		return
	}

	// 3. 同步等待结果（30 秒超时）
	result, err := h.bus.WaitCommandResult(r.Context(), resp.CommandID, 30*time.Second)
	if err != nil || result == nil {
		writeError(w, http.StatusGatewayTimeout, "查询超时，请稍后重试")
		return
	}

	// 4. 检查执行结果
	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "查询失败"
		}
		writeError(w, http.StatusInternalServerError, errMsg)
		return
	}

	// 5. JSON 透传：result.Output 是 Agent 返回的 JSON 字符串
	rawData := json.RawMessage(result.Output)

	// 6. 写缓存
	h.cache.set(cacheKey, rawData, cacheTTL)

	// 7. 返回
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    rawData,
	})
}

// buildCacheKey 构建缓存 key
func buildCacheKey(clusterID, action string, params map[string]interface{}) string {
	if len(params) == 0 {
		return clusterID + ":" + action
	}
	b, _ := json.Marshal(params)
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%s:%s:%x", clusterID, action, hash[:8])
}

// requireClusterID 提取并校验 cluster_id 参数
func requireClusterID(r *http.Request) (string, bool) {
	clusterID := r.URL.Query().Get("cluster_id")
	return clusterID, clusterID != ""
}

// ================================================================
// Metrics Handlers
// ================================================================

// MetricsSummary GET /api/v2/observe/metrics/summary (Dashboard: 快照直读)
func (h *ObserveHandler) MetricsSummary(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || otel == nil || otel.MetricsSummary == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.MetricsSummary,
	})
}

// MetricsNodes GET /api/v2/observe/metrics/nodes (Dashboard: 快照直读)
func (h *ObserveHandler) MetricsNodes(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || otel == nil || otel.MetricsNodes == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.MetricsNodes,
	})
}

// MetricsNodeRoute GET /api/v2/observe/metrics/nodes/{name}[/series]
//
// 单节点详情: 从快照 MetricsNodes 中过滤
// 节点时序: 优先从预聚合时序读取，≤15min 降级到 OTel Ring Buffer
func (h *ObserveHandler) MetricsNodeRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	// 解析路径: /api/v2/observe/metrics/nodes/{name}[/series]
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/observe/metrics/nodes/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		writeError(w, http.StatusBadRequest, "node name is required")
		return
	}

	parts := strings.SplitN(path, "/", 2)
	nodeName := parts[0]

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	if len(parts) == 2 && parts[1] == "series" {
		// GET /api/v2/observe/metrics/nodes/{name}/series
		metric := r.URL.Query().Get("metric")
		minutes := 30
		if v := r.URL.Query().Get("minutes"); v != "" {
			if m, err := strconv.Atoi(v); err == nil && m > 0 {
				minutes = m
			}
		}

		// 层 1: Ring Buffer（≤15min）— 任意指标，10s 精度
		if minutes <= 15 {
			since := time.Now().Add(-time.Duration(minutes) * time.Minute)
			entries, err := h.querySvc.GetOTelTimeline(r.Context(), clusterID, since)
			if err == nil && len(entries) > 0 {
				series := buildNodeMetricsSeries(entries, nodeName, metric)
				if len(series.Points) > 0 {
					writeJSON(w, http.StatusOK, map[string]interface{}{
						"message": "获取成功",
						"data":    series,
					})
					return
				}
			}
		}

		// 层 2: Concentrator 预聚合（≤60min）— 25 个关键指标，1min 精度
		if otel.NodeMetricsSeries != nil {
			for _, ns := range otel.NodeMetricsSeries {
				if ns.NodeName == nodeName {
					points := filterNodePointsByMinutes(ns.Points, minutes)
					writeJSON(w, http.StatusOK, map[string]interface{}{
						"message": "获取成功",
						"data": map[string]interface{}{
							"metric": metric,
							"points": extractNodeMetricPoints(points, metric),
						},
					})
					return
				}
			}
		}

		// 层 3: Command/MQ → ClickHouse（>60min，暂返回未就绪）
		writeError(w, http.StatusNotFound, "时序数据未就绪")
	} else {
		// GET /api/v2/observe/metrics/nodes/{name} — 从快照过滤
		if otel.MetricsNodes != nil {
			for _, node := range otel.MetricsNodes {
				if node.NodeName == nodeName {
					writeJSON(w, http.StatusOK, map[string]interface{}{
						"message": "获取成功",
						"data":    node,
					})
					return
				}
			}
		}
		writeError(w, http.StatusNotFound, "节点未找到")
	}
}

// ================================================================
// Logs Handlers
// ================================================================

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

	// 快速路径：无全文搜索时从快照 RecentLogs 直读
	if query == "" {
		otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
		if err == nil && otel != nil && len(otel.RecentLogs) > 0 {
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
					"logs":  logs,
					"total": total,
				},
			})
			return
		}
	}

	// 全文搜索 → Command/MQ
	delete(body, "cluster_id")
	h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
}

// ================================================================
// Traces Handlers
// ================================================================

// TracesList GET /api/v2/observe/traces
// 从快照 RecentTraces 直读，支持客户端过滤（service / min_duration / operation）
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

// TracesServices GET /api/v2/observe/traces/services (Dashboard: 快照直读)
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

// TracesTopology GET /api/v2/observe/traces/topology (Dashboard: 快照直读)
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

// ================================================================
// SLO Handlers
// ================================================================

// SLOIngress GET /api/v2/observe/slo/ingress (Dashboard: 快照直读)
func (h *ObserveHandler) SLOIngress(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || otel == nil || otel.SLOIngress == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.SLOIngress,
	})
}

// SLOServices GET /api/v2/observe/slo/services (Dashboard: 快照直读)
func (h *ObserveHandler) SLOServices(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || otel == nil || otel.SLOServices == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.SLOServices,
	})
}

// SLOEdges GET /api/v2/observe/slo/edges (Dashboard: 快照直读)
func (h *ObserveHandler) SLOEdges(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || otel == nil || otel.SLOEdges == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.SLOEdges,
	})
}

// SLOTimeSeries GET /api/v2/observe/slo/timeseries
// 优先从预聚合时序读取（1h），降级到 OTel Ring Buffer（≤15min）
func (h *ObserveHandler) SLOTimeSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	clusterID, ok := requireClusterID(r)
	if !ok {
		writeError(w, http.StatusBadRequest, "cluster_id is required")
		return
	}

	serviceName := r.URL.Query().Get("service")
	timeRange := r.URL.Query().Get("time_range")

	minutes := 60 // 默认 1h
	if m, ok := parseTimeRangeMinutes(timeRange); ok && m > 0 {
		minutes = m
	}

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	// 优先: 从预聚合时序读取
	if serviceName != "" && otel.SLOTimeSeries != nil {
		for _, ss := range otel.SLOTimeSeries {
			if ss.ServiceName == serviceName {
				points := filterSLOPointsByMinutes(ss.Points, minutes)
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"message": "获取成功",
					"data": map[string]interface{}{
						"service": serviceName,
						"points":  points,
					},
				})
				return
			}
		}
	}

	// 降级: OTel Ring Buffer（≤15min）
	if minutes <= 15 && serviceName != "" {
		since := time.Now().Add(-time.Duration(minutes) * time.Minute)
		entries, err := h.querySvc.GetOTelTimeline(r.Context(), clusterID, since)
		if err == nil && len(entries) > 0 {
			series := buildSLOTimeSeries(entries, serviceName)
			if points, ok := series["points"].([]sloPoint); ok && len(points) > 0 {
				writeJSON(w, http.StatusOK, map[string]interface{}{
					"message": "获取成功",
					"data":    series,
				})
				return
			}
		}
	}

	writeError(w, http.StatusNotFound, "时序数据未就绪")
}

// parseTimeRangeMinutes 解析时间范围字符串为分钟数
// 支持格式: "15m", "1h", "30m" 等
func parseTimeRangeMinutes(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false
	}
	return int(d.Minutes()), true
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

// SLOSummary GET /api/v2/observe/slo/summary (Dashboard: 快照直读)
func (h *ObserveHandler) SLOSummary(w http.ResponseWriter, r *http.Request) {
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
	if err != nil || otel == nil || otel.SLOSummary == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    otel.SLOSummary,
	})
}

// ================================================================
// APM TimeSeries Handlers
// ================================================================

// APMServiceSeries GET /api/v2/observe/traces/services/{name}/series
// 从预聚合 APMTimeSeries 读取指定服务的趋势数据
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

	otel, err := h.querySvc.GetOTelSnapshot(r.Context(), clusterID)
	if err != nil || otel == nil {
		writeError(w, http.StatusNotFound, "数据尚未就绪")
		return
	}

	// 从预聚合 APM 时序读取
	if otel.APMTimeSeries != nil {
		for _, s := range otel.APMTimeSeries {
			if s.ServiceName == serviceName {
				points := filterAPMPointsByMinutes(s.Points, minutes)
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

	writeError(w, http.StatusNotFound, "服务时序数据未就绪")
}

// atlhyper_master_v2/gateway/handler/observe.go
// 可观测性查询 Handler（Traces / Logs / Metrics / SLO）
// 通过 Command 机制将查询请求转发给 Agent 执行 ClickHouse 查询，结果 JSON 透传给前端
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
// Dashboard 端点（8 个）直读快照，Detail 端点（6 个）走 Command 机制。
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

	if len(parts) == 2 && parts[1] == "series" {
		// GET /api/v2/observe/metrics/nodes/{name}/series
		params := map[string]interface{}{
			"sub_action": "get_series",
			"node_name":  nodeName,
		}
		if v := r.URL.Query().Get("metric"); v != "" {
			params["metric"] = v
		}
		// minutes → since（Agent 期望 duration 字符串，如 "30m"）
		if minutes := r.URL.Query().Get("minutes"); minutes != "" {
			if m, err := strconv.Atoi(minutes); err == nil {
				params["since"] = fmt.Sprintf("%dm", m)
			}
		}
		h.executeQuery(w, r, clusterID, command.ActionQueryMetrics, params, 10*time.Second)
	} else {
		// GET /api/v2/observe/metrics/nodes/{name}
		params := map[string]interface{}{
			"sub_action": "get_node",
			"node_name":  nodeName,
		}
		h.executeQuery(w, r, clusterID, command.ActionQueryMetrics, params, 10*time.Second)
	}
}

// ================================================================
// Logs Handlers
// ================================================================

// LogsQuery POST /api/v2/observe/logs/query
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

	// body 直接作为 params 透传给 Agent
	delete(body, "cluster_id")
	h.executeQuery(w, r, clusterID, command.ActionQueryLogs, body, 0)
}

// ================================================================
// Traces Handlers
// ================================================================

// TracesList GET /api/v2/observe/traces
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

	params := map[string]interface{}{
		"sub_action": "list_traces",
	}
	// 转发查询参数（字符串参数直接透传）
	for _, key := range []string{"service", "operation", "start_time", "end_time"} {
		if v := r.URL.Query().Get(key); v != "" {
			params[key] = v
		}
	}
	// min_duration → min_duration_ms（Agent 期望的参数名）
	if v := r.URL.Query().Get("min_duration"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			params["min_duration_ms"] = f
		}
	}
	// 数值参数解析为 int（Agent getIntParam 期望数值类型）
	if v := r.URL.Query().Get("limit"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			params["limit"] = i
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			params["offset"] = i
		}
	}

	h.executeQuery(w, r, clusterID, command.ActionQueryTraces, params, 5*time.Second)
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

	params := map[string]interface{}{
		"sub_action": "get_time_series",
	}
	// service → name（Agent 期望 "name" 参数）
	if v := r.URL.Query().Get("service"); v != "" {
		params["name"] = v
	}
	// time_range → since（Agent 期望 "since" 参数）
	if v := r.URL.Query().Get("time_range"); v != "" {
		params["since"] = v
	}
	if v := r.URL.Query().Get("interval"); v != "" {
		params["interval"] = v
	}

	h.executeQuery(w, r, clusterID, command.ActionQuerySLO, params, 5*time.Second)
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

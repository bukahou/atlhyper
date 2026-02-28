// atlhyper_master_v2/gateway/handler/observe_slo_query.go
// SLO 信号域 (Observe 面板) Handler 方法
package handler

import (
	"net/http"
	"time"
)

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

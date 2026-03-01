// atlhyper_master_v2/gateway/handler/slo_targets.go
// SLO 目标管理与状态历史 Handler 方法
package slo

import (
	"encoding/json"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// ==================== 目标管理 ====================

// Targets 处理 /api/v2/slo/targets
func (h *SLOHandler) Targets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTargets(w, r)
	case http.MethodPut:
		h.updateTarget(w, r)
	default:
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *SLOHandler) getTargets(w http.ResponseWriter, r *http.Request) {
	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	targets, err := h.sloRepo.GetTargets(r.Context(), clusterID)
	if err != nil {
		sloLog.Error("获取 targets 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]model.SLOTargetResponse, 0, len(targets))
	for _, t := range targets {
		resp = append(resp, model.SLOTargetResponse{
			ID:                 t.ID,
			ClusterID:          t.ClusterID,
			Host:               t.Host,
			TimeRange:          t.TimeRange,
			AvailabilityTarget: t.AvailabilityTarget,
			P95LatencyTarget:   t.P95LatencyTarget,
			CreatedAt:          t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:          t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	handler.WriteJSON(w, http.StatusOK, resp)
}

func (h *SLOHandler) updateTarget(w http.ResponseWriter, r *http.Request) {
	var req model.UpdateSLOTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Host == "" || req.TimeRange == "" {
		handler.WriteError(w, http.StatusBadRequest, "host and time_range required")
		return
	}

	if req.ClusterID == "" {
		req.ClusterID = h.defaultClusterID(r.Context())
	}

	now := time.Now()
	target := &database.SLOTarget{
		ClusterID:          req.ClusterID,
		Host:               req.Host,
		TimeRange:          req.TimeRange,
		AvailabilityTarget: req.AvailabilityTarget,
		P95LatencyTarget:   req.P95LatencyTarget,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := h.sloRepo.UpsertTarget(r.Context(), target); err != nil {
		sloLog.Error("更新 target 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ==================== 状态历史 ====================

// StatusHistory GET /api/v2/slo/status-history
// 状态历史不再写入 SQLite，返回空数组
func (h *SLOHandler) StatusHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	handler.WriteJSON(w, http.StatusOK, []model.SLOStatusHistoryItem{})
}

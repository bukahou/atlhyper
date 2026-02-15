// atlhyper_master_v2/gateway/handler/aiops_risk.go
// AIOps 风险评分 API Handler
package handler

import (
	"net/http"
	"strconv"

	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsRiskHandler AIOps 风险评分 Handler
type AIOpsRiskHandler struct {
	svc service.Query
}

// NewAIOpsRiskHandler 创建 Handler
func NewAIOpsRiskHandler(svc service.Query) *AIOpsRiskHandler {
	return &AIOpsRiskHandler{svc: svc}
}

// ClusterRisk 获取集群风险评分
// GET /api/v2/aiops/risk/cluster?cluster={id}
func (h *AIOpsRiskHandler) ClusterRisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "missing cluster parameter")
		return
	}

	risk, err := h.svc.GetAIOpsClusterRisk(r.Context(), clusterID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, risk)
}

// EntityRisks 获取实体风险列表
// GET /api/v2/aiops/risk/entities?cluster={id}&sort=r_final&limit=20
func (h *AIOpsRiskHandler) EntityRisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "missing cluster parameter")
		return
	}

	sortBy := r.URL.Query().Get("sort")
	if sortBy == "" {
		sortBy = "r_final"
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	risks, err := h.svc.GetAIOpsEntityRisks(r.Context(), clusterID, sortBy, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, risks)
}

// EntityRisk 获取单个实体的风险详情
// GET /api/v2/aiops/risk/entity?cluster={id}&entity={key}
func (h *AIOpsRiskHandler) EntityRisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster")
	if clusterID == "" {
		writeError(w, http.StatusBadRequest, "missing cluster parameter")
		return
	}

	entityKey := r.URL.Query().Get("entity")
	if entityKey == "" {
		writeError(w, http.StatusBadRequest, "missing entity parameter")
		return
	}

	detail, err := h.svc.GetAIOpsEntityRisk(r.Context(), clusterID, entityKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, detail)
}

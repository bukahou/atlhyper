// atlhyper_master_v2/gateway/handler/aiops_baseline.go
// AIOps 基线 API Handler
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsBaselineHandler AIOps 基线 Handler
type AIOpsBaselineHandler struct {
	svc service.Query
}

// NewAIOpsBaselineHandler 创建 Handler
func NewAIOpsBaselineHandler(svc service.Query) *AIOpsBaselineHandler {
	return &AIOpsBaselineHandler{svc: svc}
}

// Baseline 获取实体基线状态
// GET /api/v2/aiops/baseline?cluster={id}&entity={key}
func (h *AIOpsBaselineHandler) Baseline(w http.ResponseWriter, r *http.Request) {
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

	baseline, err := h.svc.GetAIOpsBaseline(r.Context(), clusterID, entityKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, baseline)
}

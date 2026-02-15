// atlhyper_master_v2/gateway/handler/aiops_ai.go
// AIOps AI 增强 API Handler
package handler

import (
	"encoding/json"
	"net/http"

	"AtlHyper/atlhyper_master_v2/service"
)

// AIOpsAIHandler AIOps AI 增强 Handler
type AIOpsAIHandler struct {
	svc service.Query
}

// NewAIOpsAIHandler 创建 AIOps AI Handler
func NewAIOpsAIHandler(svc service.Query) *AIOpsAIHandler {
	return &AIOpsAIHandler{svc: svc}
}

// Summarize POST /api/v2/aiops/ai/summarize
func (h *AIOpsAIHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		IncidentID string `json:"incidentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.IncidentID == "" {
		writeError(w, http.StatusBadRequest, "incidentId is required")
		return
	}

	result, err := h.svc.SummarizeIncident(r.Context(), req.IncidentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "分析完成",
		"data":    result,
	})
}

// Recommend POST /api/v2/aiops/ai/recommend
// 与 Summarize 相同流程，但 Prompt 专注于处置建议
func (h *AIOpsAIHandler) Recommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		IncidentID string `json:"incidentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.IncidentID == "" {
		writeError(w, http.StatusBadRequest, "incidentId is required")
		return
	}

	// 复用 Summarize（Phase 4 不做区分，后续可特化 Prompt）
	result, err := h.svc.SummarizeIncident(r.Context(), req.IncidentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "AI recommendation failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "建议生成完成",
		"data":    result,
	})
}

// atlhyper_master_v2/gateway/handler/aiops_ai.go
// AIOps AI 增强 API Handler
package aiops

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/service"
)

// AnalyzeTrigger 深度分析触发接口
type AnalyzeTrigger interface {
	TriggerAnalysis(incidentID string)
}

// AIOpsAIHandler AIOps AI 增强 Handler
type AIOpsAIHandler struct {
	svc     service.Query
	trigger AnalyzeTrigger
}

// NewAIOpsAIHandler 创建 AIOps AI Handler
func NewAIOpsAIHandler(svc service.Query) *AIOpsAIHandler {
	return &AIOpsAIHandler{svc: svc}
}

// SetAnalyzeTrigger 设置深度分析触发器
func (h *AIOpsAIHandler) SetAnalyzeTrigger(t AnalyzeTrigger) {
	h.trigger = t
}

// Summarize POST /api/v2/aiops/ai/summarize
func (h *AIOpsAIHandler) Summarize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		IncidentID string `json:"incidentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.IncidentID == "" {
		handler.WriteError(w, http.StatusBadRequest, "incidentId is required")
		return
	}

	result, err := h.svc.SummarizeIncident(r.Context(), req.IncidentID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "AI analysis failed: "+err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "分析完成",
		"data":    result,
	})
}

// Recommend POST /api/v2/aiops/ai/recommend
// 与 Summarize 相同流程，但 Prompt 专注于处置建议
func (h *AIOpsAIHandler) Recommend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		IncidentID string `json:"incidentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.IncidentID == "" {
		handler.WriteError(w, http.StatusBadRequest, "incidentId is required")
		return
	}

	// 复用 Summarize（Phase 4 不做区分，后续可特化 Prompt）
	result, err := h.svc.SummarizeIncident(r.Context(), req.IncidentID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "AI recommendation failed: "+err.Error())
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "建议生成完成",
		"data":    result,
	})
}

// ReportsHandler GET /api/v2/aiops/ai/reports?incident_id=xxx
func (h *AIOpsAIHandler) ReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	incidentID := r.URL.Query().Get("incident_id")
	if incidentID == "" {
		handler.WriteError(w, http.StatusBadRequest, "incident_id is required")
		return
	}

	reports, err := h.svc.ListAIReports(r.Context(), incidentID)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to list reports")
		return
	}
	if reports == nil {
		reports = []*database.AIReport{}
	}

	// 列表只返回摘要字段，不含完整分析内容
	type reportItem struct {
		ID           int64  `json:"id"`
		IncidentID   string `json:"incidentId"`
		ClusterID    string `json:"clusterId"`
		Role         string `json:"role"`
		Trigger      string `json:"trigger"`
		Summary      string `json:"summary"`
		ProviderName string `json:"providerName"`
		Model        string `json:"model"`
		InputTokens  int    `json:"inputTokens"`
		OutputTokens int    `json:"outputTokens"`
		DurationMs   int64  `json:"durationMs"`
		CreatedAt    string `json:"createdAt"`
	}

	items := make([]reportItem, len(reports))
	for i, r := range reports {
		items[i] = reportItem{
			ID:           r.ID,
			IncidentID:   r.IncidentID,
			ClusterID:    r.ClusterID,
			Role:         r.Role,
			Trigger:      r.Trigger,
			Summary:      r.Summary,
			ProviderName: r.ProviderName,
			Model:        r.Model,
			InputTokens:  r.InputTokens,
			OutputTokens: r.OutputTokens,
			DurationMs:   r.DurationMs,
			CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		}
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    items,
	})
}

// ReportDetailHandler GET /api/v2/aiops/ai/reports/{id}
func (h *AIOpsAIHandler) ReportDetailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 从路径提取 ID
	idStr := strings.TrimPrefix(r.URL.Path, "/api/v2/aiops/ai/reports/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		handler.WriteError(w, http.StatusBadRequest, "invalid report id")
		return
	}

	report, err := h.svc.GetAIReport(r.Context(), id)
	if err != nil {
		handler.WriteError(w, http.StatusInternalServerError, "failed to get report")
		return
	}
	if report == nil {
		handler.WriteError(w, http.StatusNotFound, "report not found")
		return
	}

	type reportDetail struct {
		ID                 int64  `json:"id"`
		IncidentID         string `json:"incidentId"`
		ClusterID          string `json:"clusterId"`
		Role               string `json:"role"`
		Trigger            string `json:"trigger"`
		Summary            string `json:"summary"`
		RootCauseAnalysis  string `json:"rootCauseAnalysis"`
		Recommendations    string `json:"recommendations"`
		SimilarIncidents   string `json:"similarIncidents"`
		InvestigationSteps string `json:"investigationSteps,omitempty"`
		EvidenceChain      string `json:"evidenceChain,omitempty"`
		ProviderName       string `json:"providerName"`
		Model              string `json:"model"`
		InputTokens        int    `json:"inputTokens"`
		OutputTokens       int    `json:"outputTokens"`
		DurationMs         int64  `json:"durationMs"`
		CreatedAt          string `json:"createdAt"`
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data": reportDetail{
			ID:                 report.ID,
			IncidentID:         report.IncidentID,
			ClusterID:          report.ClusterID,
			Role:               report.Role,
			Trigger:            report.Trigger,
			Summary:            report.Summary,
			RootCauseAnalysis:  report.RootCauseAnalysis,
			Recommendations:    report.Recommendations,
			SimilarIncidents:   report.SimilarIncidents,
			InvestigationSteps: report.InvestigationSteps,
			EvidenceChain:      report.EvidenceChain,
			ProviderName:       report.ProviderName,
			Model:              report.Model,
			InputTokens:        report.InputTokens,
			OutputTokens:       report.OutputTokens,
			DurationMs:         report.DurationMs,
			CreatedAt:          report.CreatedAt.Format(time.RFC3339),
		},
	})
}

// AnalyzeHandler POST /api/v2/aiops/ai/analyze — 手动触发深度分析
func (h *AIOpsAIHandler) AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handler.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if h.trigger == nil {
		handler.WriteError(w, http.StatusServiceUnavailable, "深度分析服务未启用")
		return
	}

	var req struct {
		IncidentID string `json:"incidentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.IncidentID == "" {
		handler.WriteError(w, http.StatusBadRequest, "incidentId is required")
		return
	}

	h.trigger.TriggerAnalysis(req.IncidentID)

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "分析已提交",
	})
}

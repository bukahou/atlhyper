// atlhyper_master_v2/gateway/handler/ai_provider.go
// AI Provider 管理 API Handler
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// supportedProviders 支持的提供商 ID 列表
var supportedProviders = []string{"gemini", "openai", "anthropic"}

// providerNames 提供商名称映射
var providerNames = map[string]string{
	"gemini":    "Google Gemini",
	"openai":    "OpenAI",
	"anthropic": "Anthropic Claude",
}

// AIProviderHandler AI Provider Handler
type AIProviderHandler struct {
	db *database.DB
}

// NewAIProviderHandler 创建 AIProviderHandler
func NewAIProviderHandler(db *database.DB) *AIProviderHandler {
	return &AIProviderHandler{db: db}
}

// ==================== Response Types ====================

// ProviderResponse プロバイダー情報レスポンス
type ProviderResponse struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	Description   string  `json:"description"`
	APIKeyMasked  string  `json:"api_key_masked"`
	APIKeySet     bool    `json:"api_key_set"`
	IsActive      bool    `json:"is_active"`
	Status        string  `json:"status"`
	TotalRequests int64   `json:"total_requests"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
	LastUsedAt    *string `json:"last_used_at,omitempty"`
	LastError     string  `json:"last_error,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// ActiveConfigResponse アクティブ設定レスポンス
type ActiveConfigResponse struct {
	Enabled     bool   `json:"enabled"`
	ProviderID  *int64 `json:"provider_id"`
	ToolTimeout int    `json:"tool_timeout"`
}

// ProviderListResponse プロバイダー一覧レスポンス
type ProviderListResponse struct {
	Providers    []ProviderResponse   `json:"providers"`
	ActiveConfig ActiveConfigResponse `json:"active_config"`
	Models       []ProviderModelInfo  `json:"models"`
}

// ProviderModelInfo モデル情報
type ProviderModelInfo struct {
	Provider    string   `json:"provider"`
	Name        string   `json:"name"`
	Models      []string `json:"models"`
}

// ProviderCreateRequest プロバイダー作成リクエスト
type ProviderCreateRequest struct {
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Description string `json:"description"`
}

// ProviderUpdateRequest プロバイダー更新リクエスト
type ProviderUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	APIKey      *string `json:"api_key,omitempty"`
	Model       *string `json:"model,omitempty"`
	Description *string `json:"description,omitempty"`
}

// ActiveConfigUpdateRequest アクティブ設定更新リクエスト
type ActiveConfigUpdateRequest struct {
	Enabled     *bool  `json:"enabled,omitempty"`
	ProviderID  *int64 `json:"provider_id,omitempty"`
	ToolTimeout *int   `json:"tool_timeout,omitempty"`
}

// ==================== Handlers ====================

// ProvidersHandler プロバイダー一覧・作成
// GET  /api/v2/ai/providers -> 一覧取得
// POST /api/v2/ai/providers -> 新規作成
func (h *AIProviderHandler) ProvidersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listProviders(w, r)
	case http.MethodPost:
		h.createProvider(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ProviderHandler 個別プロバイダー操作
// GET    /api/v2/ai/providers/{id} -> 取得
// PUT    /api/v2/ai/providers/{id} -> 更新
// DELETE /api/v2/ai/providers/{id} -> 削除
func (h *AIProviderHandler) ProviderHandler(w http.ResponseWriter, r *http.Request) {
	// パスから ID 抽出
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/ai/providers/")
	idStr := strings.Split(path, "/")[0]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProvider(w, r, id)
	case http.MethodPut, http.MethodPatch:
		h.updateProvider(w, r, id)
	case http.MethodDelete:
		h.deleteProvider(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ActiveConfigHandler アクティブ設定操作
// GET /api/v2/ai/active -> 取得
// PUT /api/v2/ai/active -> 更新
func (h *AIProviderHandler) ActiveConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getActiveConfig(w, r)
	case http.MethodPut, http.MethodPatch:
		h.updateActiveConfig(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ==================== Implementation ====================

// listProviders プロバイダー一覧取得
func (h *AIProviderHandler) listProviders(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// プロバイダー一覧
	providers, err := h.db.AIProvider.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}

	// アクティブ設定
	active, err := h.db.AIActive.Get(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get active config")
		return
	}
	// 如果 active 为 nil（表中无记录），使用默认值
	if active == nil {
		active = &database.AIActiveConfig{
			Enabled:     false,
			ProviderID:  nil,
			ToolTimeout: 30,
		}
	}

	// モデル一覧
	models := h.loadModelsGrouped(ctx)

	// レスポンス構築
	resp := ProviderListResponse{
		Providers: make([]ProviderResponse, 0, len(providers)),
		ActiveConfig: ActiveConfigResponse{
			Enabled:     active.Enabled,
			ProviderID:  active.ProviderID,
			ToolTimeout: active.ToolTimeout,
		},
		Models: models,
	}

	for _, p := range providers {
		resp.Providers = append(resp.Providers, h.toProviderResponse(p, active.ProviderID))
	}

	writeJSON(w, http.StatusOK, resp)
}

// createProvider プロバイダー作成
func (h *AIProviderHandler) createProvider(w http.ResponseWriter, r *http.Request) {
	var req ProviderCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// バリデーション
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "api_key is required")
		return
	}
	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "model is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	now := time.Now()
	provider := &database.AIProvider{
		Name:        req.Name,
		Provider:    req.Provider,
		APIKey:      req.APIKey,
		Model:       req.Model,
		Description: req.Description,
		Status:      "unknown",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.db.AIProvider.Create(ctx, provider); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create provider")
		return
	}

	active, _ := h.db.AIActive.Get(ctx)
	writeJSON(w, http.StatusCreated, h.toProviderResponse(provider, active.ProviderID))
}

// getProvider プロバイダー取得
func (h *AIProviderHandler) getProvider(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	provider, err := h.db.AIProvider.GetByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}
	if provider == nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	active, _ := h.db.AIActive.Get(ctx)
	writeJSON(w, http.StatusOK, h.toProviderResponse(provider, active.ProviderID))
}

// updateProvider プロバイダー更新
func (h *AIProviderHandler) updateProvider(w http.ResponseWriter, r *http.Request, id int64) {
	var req ProviderUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	provider, err := h.db.AIProvider.GetByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get provider")
		return
	}
	if provider == nil {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	// 更新
	if req.Name != nil {
		provider.Name = *req.Name
	}
	if req.Provider != nil {
		provider.Provider = *req.Provider
	}
	if req.APIKey != nil && *req.APIKey != "" {
		provider.APIKey = *req.APIKey
	}
	if req.Model != nil {
		provider.Model = *req.Model
	}
	if req.Description != nil {
		provider.Description = *req.Description
	}
	provider.UpdatedAt = time.Now()

	if err := h.db.AIProvider.Update(ctx, provider); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update provider")
		return
	}

	active, _ := h.db.AIActive.Get(ctx)
	writeJSON(w, http.StatusOK, h.toProviderResponse(provider, active.ProviderID))
}

// deleteProvider プロバイダー削除
func (h *AIProviderHandler) deleteProvider(w http.ResponseWriter, r *http.Request, id int64) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// アクティブなプロバイダーは削除不可
	active, _ := h.db.AIActive.Get(ctx)
	if active != nil && active.ProviderID != nil && *active.ProviderID == id {
		writeError(w, http.StatusBadRequest, "cannot delete active provider")
		return
	}

	if err := h.db.AIProvider.Delete(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete provider")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// getActiveConfig アクティブ設定取得
func (h *AIProviderHandler) getActiveConfig(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	active, err := h.db.AIActive.Get(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get active config")
		return
	}
	// 如果 active 为 nil，使用默认值
	if active == nil {
		active = &database.AIActiveConfig{
			Enabled:     false,
			ProviderID:  nil,
			ToolTimeout: 30,
		}
	}

	writeJSON(w, http.StatusOK, ActiveConfigResponse{
		Enabled:     active.Enabled,
		ProviderID:  active.ProviderID,
		ToolTimeout: active.ToolTimeout,
	})
}

// updateActiveConfig アクティブ設定更新
func (h *AIProviderHandler) updateActiveConfig(w http.ResponseWriter, r *http.Request) {
	var req ActiveConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	active, err := h.db.AIActive.Get(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get active config")
		return
	}
	// 如果 active 为 nil，使用默认值
	if active == nil {
		active = &database.AIActiveConfig{
			Enabled:     false,
			ProviderID:  nil,
			ToolTimeout: 30,
		}
	}

	// 更新
	if req.Enabled != nil {
		active.Enabled = *req.Enabled
	}
	if req.ProviderID != nil {
		// プロバイダー存在チェック
		provider, _ := h.db.AIProvider.GetByID(ctx, *req.ProviderID)
		if provider == nil {
			writeError(w, http.StatusBadRequest, "provider not found")
			return
		}
		active.ProviderID = req.ProviderID
	}
	if req.ToolTimeout != nil {
		active.ToolTimeout = *req.ToolTimeout
	}
	active.UpdatedAt = time.Now()

	if err := h.db.AIActive.Update(ctx, active); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update active config")
		return
	}

	writeJSON(w, http.StatusOK, ActiveConfigResponse{
		Enabled:     active.Enabled,
		ProviderID:  active.ProviderID,
		ToolTimeout: active.ToolTimeout,
	})
}

// ==================== Helpers ====================

// toProviderResponse DB モデルをレスポンスに変換
func (h *AIProviderHandler) toProviderResponse(p *database.AIProvider, activeID *int64) ProviderResponse {
	resp := ProviderResponse{
		ID:            p.ID,
		Name:          p.Name,
		Provider:      p.Provider,
		Model:         p.Model,
		Description:   p.Description,
		APIKeyMasked:  maskAPIKey(p.APIKey),
		APIKeySet:     p.APIKey != "",
		IsActive:      activeID != nil && *activeID == p.ID,
		Status:        p.Status,
		TotalRequests: p.TotalRequests,
		TotalTokens:   p.TotalTokens,
		TotalCost:     p.TotalCost,
		LastError:     p.LastError,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.Format(time.RFC3339),
	}
	if p.LastUsedAt != nil {
		s := p.LastUsedAt.Format(time.RFC3339)
		resp.LastUsedAt = &s
	}
	return resp
}

// loadModelsGrouped プロバイダー別モデル一覧取得
func (h *AIProviderHandler) loadModelsGrouped(ctx context.Context) []ProviderModelInfo {
	models, err := h.db.AIModel.ListAll(ctx)
	if err != nil {
		return []ProviderModelInfo{}
	}

	// グループ化
	grouped := make(map[string][]string)
	for _, m := range models {
		grouped[m.Provider] = append(grouped[m.Provider], m.Model)
	}

	// 結果構築
	result := []ProviderModelInfo{}
	for _, pid := range supportedProviders {
		if ms, ok := grouped[pid]; ok {
			result = append(result, ProviderModelInfo{
				Provider: pid,
				Name:     providerNames[pid],
				Models:   ms,
			})
		}
	}

	return result
}

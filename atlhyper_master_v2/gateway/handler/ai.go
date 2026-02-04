// atlhyper_master_v2/gateway/handler/ai.go
// AI 对话 API Handler
// 提供会话 CRUD + SSE 流式 Chat
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	aiPkg "AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/common/logger"
)

var aiLog = logger.Module("AI-Handler")

// AIHandler AI Handler
type AIHandler struct {
	aiService aiPkg.AIService
}

// NewAIHandler 创建 AIHandler
func NewAIHandler(aiService aiPkg.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// ==================== 对话管理 ====================

// createConversationRequest 创建对话请求
type createConversationRequest struct {
	ClusterID string `json:"cluster_id"`
	Title     string `json:"title"`
}

// Conversations 处理 /api/v2/ai/conversations
// POST: 创建对话, GET: 列表
func (h *AIHandler) Conversations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createConversation(w, r)
	case http.MethodGet:
		h.listConversations(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// createConversation 创建对话
func (h *AIHandler) createConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req createConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ClusterID == "" {
		writeError(w, http.StatusBadRequest, "cluster_id required")
		return
	}

	title := req.Title
	if title == "" {
		title = "新对话"
	}

	conv, err := h.aiService.CreateConversation(r.Context(), userID, req.ClusterID, title)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, conv)
}

// listConversations 获取对话列表
func (h *AIHandler) listConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	convs, err := h.aiService.GetConversations(r.Context(), userID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, convs)
}

// ConversationByID 处理 /api/v2/ai/conversations/{id}...
// GET .../messages: 获取消息, DELETE: 删除对话
func (h *AIHandler) ConversationByID(w http.ResponseWriter, r *http.Request) {
	// 解析路径: /api/v2/ai/conversations/{id} 或 /api/v2/ai/conversations/{id}/messages
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/ai/conversations/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "conversation id required")
		return
	}

	convID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid conversation id")
		return
	}

	// /api/v2/ai/conversations/{id}/messages
	if len(parts) == 2 && parts[1] == "messages" {
		h.getMessages(w, r, convID)
		return
	}

	// /api/v2/ai/conversations/{id}
	switch r.Method {
	case http.MethodDelete:
		h.deleteConversation(w, r, convID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getMessages 获取对话消息
func (h *AIHandler) getMessages(w http.ResponseWriter, r *http.Request, convID int64) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	msgs, err := h.aiService.GetMessages(r.Context(), convID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, msgs)
}

// deleteConversation 删除对话（需要 Operator 权限）
func (h *AIHandler) deleteConversation(w http.ResponseWriter, r *http.Request, convID int64) {
	// 检查权限：需要 Operator (Role >= 2)
	role, ok := middleware.GetRole(r.Context())
	if !ok || role < middleware.RoleOperator {
		writeError(w, http.StatusForbidden, "需要 Operator 权限")
		return
	}

	if err := h.aiService.DeleteConversation(r.Context(), convID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ==================== Chat (SSE) ====================

// chatRequest Chat 请求
type chatRequest struct {
	ConversationID int64  `json:"conversation_id"`
	ClusterID      string `json:"cluster_id"`
	Message        string `json:"message"`
}

// Chat 处理 /api/v2/ai/chat (SSE 流式响应)
func (h *AIHandler) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "未获取到用户信息")
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ConversationID == 0 || req.Message == "" {
		writeError(w, http.StatusBadRequest, "conversation_id and message required")
		return
	}

	// 调用 AIService.Chat
	ch, err := h.aiService.Chat(r.Context(), &aiPkg.ChatRequest{
		ConversationID: req.ConversationID,
		ClusterID:      req.ClusterID,
		UserID:         userID,
		Message:        req.Message,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 设置 SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx 禁用缓冲

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// 流式输出
	chunkCount := 0
	for chunk := range ch {
		chunkCount++
		data, _ := json.Marshal(chunk)
		// 只记录错误类型的消息
		if chunk.Type == "error" {
			aiLog.Warn("SSE 发送错误", "chunk", chunkCount, "type", chunk.Type)
		}
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
		flusher.Flush()
	}

	aiLog.Debug("SSE 流结束", "conv", req.ConversationID, "user", userID, "chunks", chunkCount)
}

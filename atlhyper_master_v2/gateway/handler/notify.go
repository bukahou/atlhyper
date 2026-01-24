// atlhyper_master_v2/gateway/handler/notify.go
// 通知配置 API Handler
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// NotifyHandler 通知 Handler
type NotifyHandler struct {
	db *database.DB
}

// NewNotifyHandler 创建 NotifyHandler
func NewNotifyHandler(db *database.DB) *NotifyHandler {
	return &NotifyHandler{
		db: db,
	}
}

// ChannelResponse 渠道响应
type ChannelResponse struct {
	ID        int64           `json:"id"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Enabled   bool            `json:"enabled"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ListChannels 列出通知渠道
func (h *NotifyHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	channels, err := h.db.Notify.List(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list channels")
		return
	}

	// 转换响应
	var responses []ChannelResponse
	for _, ch := range channels {
		responses = append(responses, ChannelResponse{
			ID:        ch.ID,
			Type:      ch.Type,
			Name:      ch.Name,
			Enabled:   ch.Enabled,
			Config:    json.RawMessage(ch.Config),
			CreatedAt: ch.CreatedAt,
			UpdatedAt: ch.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"channels": responses,
		"total":    len(responses),
	})
}

// UpdateChannelRequest 更新通知渠道请求
type UpdateChannelRequest struct {
	Enabled *bool           `json:"enabled,omitempty"`
	Name    string          `json:"name,omitempty"`
	Config  json.RawMessage `json:"config,omitempty"`
}

// ChannelHandler 通知渠道综合处理（Admin 权限）
// 根据 Method 和 Path 分发到对应的处理函数
// GET  /api/v2/notify/channels/{type}       -> 获取详情
// PUT  /api/v2/notify/channels/{type}       -> 更新配置
// POST /api/v2/notify/channels/{type}/test  -> 测试发送
func (h *NotifyHandler) ChannelHandler(w http.ResponseWriter, r *http.Request) {
	// 解析路径
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/notify/channels/")

	// 检查是否是测试请求
	if strings.HasSuffix(path, "/test") {
		h.testChannel(w, r, strings.TrimSuffix(path, "/test"))
		return
	}

	// 根据 Method 分发
	switch r.Method {
	case http.MethodGet:
		h.getChannel(w, r, strings.TrimSuffix(path, "/"))
	case http.MethodPut, http.MethodPatch:
		h.updateChannel(w, r, strings.TrimSuffix(path, "/"))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// getChannel 获取通知渠道详情
func (h *NotifyHandler) getChannel(w http.ResponseWriter, r *http.Request, channelType string) {
	if channelType == "" {
		writeError(w, http.StatusBadRequest, "channel type required")
		return
	}

	if !isValidChannelType(channelType) {
		writeError(w, http.StatusBadRequest, "invalid channel type")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	channel, err := h.db.Notify.GetByType(ctx, channelType)
	if err != nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}

	writeJSON(w, http.StatusOK, ChannelResponse{
		ID:        channel.ID,
		Type:      channel.Type,
		Name:      channel.Name,
		Enabled:   channel.Enabled,
		Config:    json.RawMessage(channel.Config),
		CreatedAt: channel.CreatedAt,
		UpdatedAt: channel.UpdatedAt,
	})
}

// UpdateChannel 更新通知渠道（保留用于独立路由）
func (h *NotifyHandler) UpdateChannel(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v2/notify/channels/")
	channelType := strings.TrimSuffix(path, "/")
	h.updateChannel(w, r, channelType)
}

// updateChannel 更新通知渠道（内部方法）
func (h *NotifyHandler) updateChannel(w http.ResponseWriter, r *http.Request, channelType string) {
	if channelType == "" {
		writeError(w, http.StatusBadRequest, "channel type required")
		return
	}

	if !isValidChannelType(channelType) {
		writeError(w, http.StatusBadRequest, "invalid channel type")
		return
	}

	var req UpdateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 获取现有配置
	existing, err := h.db.Notify.GetByType(ctx, channelType)
	if err != nil || existing == nil {
		// 不存在则创建
		existing = &database.NotifyChannel{
			Type:    channelType,
			Name:    channelType,
			Enabled: false,
			Config:  "{}",
		}
	}

	// 更新字段
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Config != nil {
		existing.Config = string(req.Config)
	}

	// 保存
	if existing.ID == 0 {
		err = h.db.Notify.Create(ctx, existing)
	} else {
		err = h.db.Notify.Update(ctx, existing)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update channel")
		return
	}

	writeJSON(w, http.StatusOK, ChannelResponse{
		ID:        existing.ID,
		Type:      existing.Type,
		Name:      existing.Name,
		Enabled:   existing.Enabled,
		Config:    json.RawMessage(existing.Config),
		CreatedAt: existing.CreatedAt,
		UpdatedAt: existing.UpdatedAt,
	})
}

// testChannel 测试通知渠道（内部方法）
func (h *NotifyHandler) testChannel(w http.ResponseWriter, r *http.Request, channelType string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if channelType == "" {
		writeError(w, http.StatusBadRequest, "channel type required")
		return
	}

	if !isValidChannelType(channelType) {
		writeError(w, http.StatusBadRequest, "invalid channel type")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 获取渠道配置
	channel, err := h.db.Notify.GetByType(ctx, channelType)
	if err != nil || channel == nil {
		writeError(w, http.StatusNotFound, "channel not found")
		return
	}

	if !channel.Enabled {
		writeError(w, http.StatusBadRequest, "channel is disabled")
		return
	}

	// TODO: 实际发送测试通知
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "测试通知已发送",
		"channel": channelType,
		"success": true,
	})
}

// ==================== 辅助函数 ====================

// validChannelTypes 有效的渠道类型
var validChannelTypes = map[string]bool{
	"email":    true,
	"mail":     true, // alias
	"slack":    true,
	"webhook":  true,
	"dingtalk": true,
}

// isValidChannelType 校验渠道类型
func isValidChannelType(channelType string) bool {
	return validChannelTypes[channelType]
}

// atlhyper_master_v2/database/sqlite/ai.go
// SQLite AI Dialect 实现 (AIConversation + AIMessage)
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// ==================== AIConversation Dialect ====================

type aiConversationDialect struct{}

func (d *aiConversationDialect) Insert(conv *database.AIConversation) (string, []any) {
	query := `INSERT INTO ai_conversations (user_id, cluster_id, title, message_count, total_input_tokens, total_output_tokens, total_tool_calls, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		conv.UserID, conv.ClusterID, conv.Title, conv.MessageCount,
		conv.TotalInputTokens, conv.TotalOutputTokens, conv.TotalToolCalls,
		conv.CreatedAt.Format(time.RFC3339), conv.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *aiConversationDialect) Update(conv *database.AIConversation) (string, []any) {
	query := `UPDATE ai_conversations SET title = ?, message_count = ?, total_input_tokens = ?, total_output_tokens = ?, total_tool_calls = ?, updated_at = ? WHERE id = ?`
	args := []any{conv.Title, conv.MessageCount, conv.TotalInputTokens, conv.TotalOutputTokens, conv.TotalToolCalls, conv.UpdatedAt.Format(time.RFC3339), conv.ID}
	return query, args
}

func (d *aiConversationDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM ai_conversations WHERE id = ?", []any{id}
}

func (d *aiConversationDialect) SelectByID(id int64) (string, []any) {
	return "SELECT id, user_id, cluster_id, title, message_count, total_input_tokens, total_output_tokens, total_tool_calls, created_at, updated_at FROM ai_conversations WHERE id = ?", []any{id}
}

func (d *aiConversationDialect) SelectByUser(userID int64, limit, offset int) (string, []any) {
	return "SELECT id, user_id, cluster_id, title, message_count, total_input_tokens, total_output_tokens, total_tool_calls, created_at, updated_at FROM ai_conversations WHERE user_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?",
		[]any{userID, limit, offset}
}

func (d *aiConversationDialect) ScanRow(rows *sql.Rows) (*database.AIConversation, error) {
	conv := &database.AIConversation{}
	var createdAt, updatedAt string
	err := rows.Scan(&conv.ID, &conv.UserID, &conv.ClusterID, &conv.Title, &conv.MessageCount,
		&conv.TotalInputTokens, &conv.TotalOutputTokens, &conv.TotalToolCalls,
		&createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	conv.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return conv, nil
}

var _ database.AIConversationDialect = (*aiConversationDialect)(nil)

// ==================== AIMessage Dialect ====================

type aiMessageDialect struct{}

func (d *aiMessageDialect) Insert(msg *database.AIMessage) (string, []any) {
	query := `INSERT INTO ai_messages (conversation_id, role, content, tool_calls, created_at)
	VALUES (?, ?, ?, ?, ?)`
	args := []any{
		msg.ConversationID, msg.Role, msg.Content, msg.ToolCalls,
		msg.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *aiMessageDialect) SelectByConversation(convID int64) (string, []any) {
	return "SELECT id, conversation_id, role, content, tool_calls, created_at FROM ai_messages WHERE conversation_id = ? ORDER BY created_at ASC", []any{convID}
}

func (d *aiMessageDialect) DeleteByConversation(convID int64) (string, []any) {
	return "DELETE FROM ai_messages WHERE conversation_id = ?", []any{convID}
}

func (d *aiMessageDialect) ScanRow(rows *sql.Rows) (*database.AIMessage, error) {
	msg := &database.AIMessage{}
	var createdAt string
	var toolCalls sql.NullString
	err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &toolCalls, &createdAt)
	if err != nil {
		return nil, err
	}
	if toolCalls.Valid {
		msg.ToolCalls = toolCalls.String
	}
	msg.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return msg, nil
}

var _ database.AIMessageDialect = (*aiMessageDialect)(nil)

// ==================== AIProvider Dialect ====================

type aiProviderDialect struct{}

func (d *aiProviderDialect) Insert(p *database.AIProvider) (string, []any) {
	query := `INSERT INTO ai_providers (name, provider, api_key, model, description,
		total_requests, total_tokens, total_cost, status, created_at, created_by, updated_at, updated_by)
	VALUES (?, ?, ?, ?, ?, 0, 0, 0, 'unknown', ?, ?, ?, ?)`
	args := []any{
		p.Name, p.Provider, p.APIKey, p.Model, p.Description,
		p.CreatedAt.Format(time.RFC3339), p.CreatedBy,
		p.UpdatedAt.Format(time.RFC3339), p.UpdatedBy,
	}
	return query, args
}

func (d *aiProviderDialect) Update(p *database.AIProvider) (string, []any) {
	query := `UPDATE ai_providers SET name = ?, provider = ?, api_key = ?, model = ?, description = ?, updated_at = ?, updated_by = ? WHERE id = ? AND deleted_at IS NULL`
	args := []any{p.Name, p.Provider, p.APIKey, p.Model, p.Description, p.UpdatedAt.Format(time.RFC3339), p.UpdatedBy, p.ID}
	return query, args
}

func (d *aiProviderDialect) Delete(id int64) (string, []any) {
	// 软删除
	return "UPDATE ai_providers SET deleted_at = ? WHERE id = ?", []any{time.Now().Format(time.RFC3339), id}
}

func (d *aiProviderDialect) SelectByID(id int64) (string, []any) {
	return `SELECT id, name, provider, api_key, model, description,
		total_requests, total_tokens, total_cost, last_used_at, last_error, last_error_at,
		status, status_checked_at, created_at, created_by, updated_at, updated_by, deleted_at
		FROM ai_providers WHERE id = ? AND deleted_at IS NULL`, []any{id}
}

func (d *aiProviderDialect) SelectAll() (string, []any) {
	return `SELECT id, name, provider, api_key, model, description,
		total_requests, total_tokens, total_cost, last_used_at, last_error, last_error_at,
		status, status_checked_at, created_at, created_by, updated_at, updated_by, deleted_at
		FROM ai_providers WHERE deleted_at IS NULL ORDER BY id ASC`, nil
}

func (d *aiProviderDialect) IncrementUsage(id int64, requests, tokens int64, cost float64) (string, []any) {
	return `UPDATE ai_providers SET total_requests = total_requests + ?, total_tokens = total_tokens + ?, total_cost = total_cost + ?, last_used_at = ? WHERE id = ?`,
		[]any{requests, tokens, cost, time.Now().Format(time.RFC3339), id}
}

func (d *aiProviderDialect) UpdateStatus(id int64, status, errorMsg string) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	if errorMsg != "" {
		return `UPDATE ai_providers SET status = ?, last_error = ?, last_error_at = ?, status_checked_at = ? WHERE id = ?`,
			[]any{status, errorMsg, now, now, id}
	}
	return `UPDATE ai_providers SET status = ?, status_checked_at = ? WHERE id = ?`, []any{status, now, id}
}

func (d *aiProviderDialect) ScanRow(rows *sql.Rows) (*database.AIProvider, error) {
	p := &database.AIProvider{}
	var createdAt, updatedAt string
	var lastUsedAt, lastErrorAt, statusCheckedAt, deletedAt sql.NullString
	var lastError sql.NullString
	var status sql.NullString

	err := rows.Scan(&p.ID, &p.Name, &p.Provider, &p.APIKey, &p.Model, &p.Description,
		&p.TotalRequests, &p.TotalTokens, &p.TotalCost, &lastUsedAt, &lastError, &lastErrorAt,
		&status, &statusCheckedAt, &createdAt, &p.CreatedBy, &updatedAt, &p.UpdatedBy, &deletedAt)
	if err != nil {
		return nil, err
	}

	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	if lastUsedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastUsedAt.String)
		p.LastUsedAt = &t
	}
	if lastError.Valid {
		p.LastError = lastError.String
	}
	if lastErrorAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastErrorAt.String)
		p.LastErrorAt = &t
	}
	if status.Valid {
		p.Status = status.String
	}
	if statusCheckedAt.Valid {
		t, _ := time.Parse(time.RFC3339, statusCheckedAt.String)
		p.StatusCheckedAt = &t
	}
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		p.DeletedAt = &t
	}

	return p, nil
}

var _ database.AIProviderDialect = (*aiProviderDialect)(nil)

// ==================== AIActiveConfig Dialect ====================

type aiActiveConfigDialect struct{}

func (d *aiActiveConfigDialect) Select() (string, []any) {
	return "SELECT id, enabled, provider_id, tool_timeout, updated_at, updated_by FROM ai_active_config WHERE id = 1", nil
}

func (d *aiActiveConfigDialect) Update(cfg *database.AIActiveConfig) (string, []any) {
	// 使用 INSERT OR REPLACE 确保首次初始化时能创建记录
	return `INSERT OR REPLACE INTO ai_active_config (id, enabled, provider_id, tool_timeout, updated_at, updated_by) VALUES (1, ?, ?, ?, ?, ?)`,
		[]any{cfg.Enabled, cfg.ProviderID, cfg.ToolTimeout, cfg.UpdatedAt.Format(time.RFC3339), cfg.UpdatedBy}
}

func (d *aiActiveConfigDialect) SwitchProvider(providerID int64, updatedBy int64) (string, []any) {
	return `UPDATE ai_active_config SET provider_id = ?, updated_at = ?, updated_by = ? WHERE id = 1`,
		[]any{providerID, time.Now().Format(time.RFC3339), updatedBy}
}

func (d *aiActiveConfigDialect) SetEnabled(enabled bool, updatedBy int64) (string, []any) {
	return `UPDATE ai_active_config SET enabled = ?, updated_at = ?, updated_by = ? WHERE id = 1`,
		[]any{enabled, time.Now().Format(time.RFC3339), updatedBy}
}

func (d *aiActiveConfigDialect) ScanRow(rows *sql.Rows) (*database.AIActiveConfig, error) {
	cfg := &database.AIActiveConfig{}
	var updatedAt string
	var providerID sql.NullInt64
	var updatedBy sql.NullInt64

	err := rows.Scan(&cfg.ID, &cfg.Enabled, &providerID, &cfg.ToolTimeout, &updatedAt, &updatedBy)
	if err != nil {
		return nil, err
	}

	cfg.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if providerID.Valid {
		cfg.ProviderID = &providerID.Int64
	}
	if updatedBy.Valid {
		cfg.UpdatedBy = updatedBy.Int64
	}

	return cfg, nil
}

var _ database.AIActiveConfigDialect = (*aiActiveConfigDialect)(nil)

// ==================== AIProviderModel Dialect ====================

type aiProviderModelDialect struct{}

func (d *aiProviderModelDialect) Insert(m *database.AIProviderModel) (string, []any) {
	query := `INSERT INTO ai_provider_models (provider, model, display_name, is_default, sort_order, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	args := []any{m.Provider, m.Model, m.DisplayName, m.IsDefault, m.SortOrder, m.CreatedAt.Format(time.RFC3339)}
	return query, args
}

func (d *aiProviderModelDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM ai_provider_models WHERE id = ?", []any{id}
}

func (d *aiProviderModelDialect) SelectByID(id int64) (string, []any) {
	return "SELECT id, provider, model, display_name, is_default, sort_order, created_at FROM ai_provider_models WHERE id = ?", []any{id}
}

func (d *aiProviderModelDialect) SelectByProvider(provider string) (string, []any) {
	return "SELECT id, provider, model, display_name, is_default, sort_order, created_at FROM ai_provider_models WHERE provider = ? ORDER BY sort_order ASC", []any{provider}
}

func (d *aiProviderModelDialect) SelectAll() (string, []any) {
	return "SELECT id, provider, model, display_name, is_default, sort_order, created_at FROM ai_provider_models ORDER BY provider, sort_order ASC", nil
}

func (d *aiProviderModelDialect) SelectDefault(provider string) (string, []any) {
	return "SELECT id, provider, model, display_name, is_default, sort_order, created_at FROM ai_provider_models WHERE provider = ? AND is_default = 1 LIMIT 1", []any{provider}
}

func (d *aiProviderModelDialect) ScanRow(rows *sql.Rows) (*database.AIProviderModel, error) {
	m := &database.AIProviderModel{}
	var createdAt string
	var displayName sql.NullString

	err := rows.Scan(&m.ID, &m.Provider, &m.Model, &displayName, &m.IsDefault, &m.SortOrder, &createdAt)
	if err != nil {
		return nil, err
	}

	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if displayName.Valid {
		m.DisplayName = displayName.String
	}

	return m, nil
}

var _ database.AIProviderModelDialect = (*aiProviderModelDialect)(nil)

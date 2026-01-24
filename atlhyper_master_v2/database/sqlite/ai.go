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
	query := `INSERT INTO ai_conversations (user_id, cluster_id, title, message_count, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)`
	args := []any{
		conv.UserID, conv.ClusterID, conv.Title, conv.MessageCount,
		conv.CreatedAt.Format(time.RFC3339), conv.UpdatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *aiConversationDialect) Update(conv *database.AIConversation) (string, []any) {
	query := `UPDATE ai_conversations SET title = ?, message_count = ?, updated_at = ? WHERE id = ?`
	args := []any{conv.Title, conv.MessageCount, conv.UpdatedAt.Format(time.RFC3339), conv.ID}
	return query, args
}

func (d *aiConversationDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM ai_conversations WHERE id = ?", []any{id}
}

func (d *aiConversationDialect) SelectByID(id int64) (string, []any) {
	return "SELECT id, user_id, cluster_id, title, message_count, created_at, updated_at FROM ai_conversations WHERE id = ?", []any{id}
}

func (d *aiConversationDialect) SelectByUser(userID int64, limit, offset int) (string, []any) {
	return "SELECT id, user_id, cluster_id, title, message_count, created_at, updated_at FROM ai_conversations WHERE user_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?",
		[]any{userID, limit, offset}
}

func (d *aiConversationDialect) ScanRow(rows *sql.Rows) (*database.AIConversation, error) {
	conv := &database.AIConversation{}
	var createdAt, updatedAt string
	err := rows.Scan(&conv.ID, &conv.UserID, &conv.ClusterID, &conv.Title, &conv.MessageCount, &createdAt, &updatedAt)
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

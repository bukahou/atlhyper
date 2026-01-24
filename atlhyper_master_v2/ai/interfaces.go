// atlhyper_master_v2/ai/interfaces.go
// AIService 对外接口 + 类型定义
package ai

import (
	"context"
	"time"
)

// AIService AI 服务接口
type AIService interface {
	// CreateConversation 创建对话
	CreateConversation(ctx context.Context, userID int64, clusterID, title string) (*Conversation, error)

	// Chat 发送消息并获取流式响应
	// 返回 ChatChunk channel，通过 SSE 推送给前端
	Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error)

	// GetConversations 获取用户对话列表
	GetConversations(ctx context.Context, userID int64, limit, offset int) ([]*Conversation, error)

	// GetMessages 获取对话历史消息
	GetMessages(ctx context.Context, conversationID int64) ([]*Message, error)

	// DeleteConversation 删除对话及其所有消息
	DeleteConversation(ctx context.Context, conversationID int64) error
}

// ChatRequest 发送消息请求
type ChatRequest struct {
	ConversationID int64  // 对话 ID
	ClusterID      string // 目标集群
	UserID         int64  // 用户 ID
	Message        string // 用户消息
}

// ChatChunk SSE 流式响应块
type ChatChunk struct {
	Type    string `json:"type"`              // text / tool_call / tool_result / done / error
	Content string `json:"content,omitempty"` // 文本内容
	Tool    string `json:"tool,omitempty"`    // tool 名称
	Params  string `json:"params,omitempty"`  // tool 参数 JSON
}

// Conversation 对话
type Conversation struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	ClusterID    string    `json:"cluster_id"`
	Title        string    `json:"title"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Message 消息
type Message struct {
	ID             int64     `json:"id"`
	ConversationID int64     `json:"conversation_id"`
	Role           string    `json:"role"`                 // user / assistant / tool
	Content        string    `json:"content"`
	ToolCalls      string    `json:"tool_calls,omitempty"` // JSON
	CreatedAt      time.Time `json:"created_at"`
}

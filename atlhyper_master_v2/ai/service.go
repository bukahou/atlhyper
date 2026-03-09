// atlhyper_master_v2/ai/service.go
// AIService 实现
// 提供会话 CRUD + Chat 入口
package ai

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

// ServiceConfig AI 服务配置（仅用于 Tool 超时等非敏感配置）
type ServiceConfig struct {
	ToolTimeout time.Duration // Tool 执行超时，默认 30s
}

// aiServiceImpl AIService 实现
// 注意：不再缓存 llmClient，每次 Chat 时从 DB 获取最新配置并创建客户端
type aiServiceImpl struct {
	providerRepo database.AIProviderRepository
	settingsRepo database.AISettingsRepository
	modelRepo    database.AIProviderModelRepository
	budgetRepo   database.AIRoleBudgetRepository
	executor     *toolExecutor
	convRepo     database.AIConversationRepository
	msgRepo      database.AIMessageRepository
}

// CreateConversation 创建对话
func (s *aiServiceImpl) CreateConversation(ctx context.Context, userID int64, clusterID, title string) (*Conversation, error) {
	now := time.Now()
	conv := &database.AIConversation{
		UserID:    userID,
		ClusterID: clusterID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}
	return toConversation(conv), nil
}

// GetConversations 获取用户对话列表
func (s *aiServiceImpl) GetConversations(ctx context.Context, userID int64, limit, offset int) ([]*Conversation, error) {
	if limit <= 0 {
		limit = 20
	}
	convs, err := s.convRepo.ListByUser(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]*Conversation, len(convs))
	for i, c := range convs {
		result[i] = toConversation(c)
	}
	return result, nil
}

// GetMessages 获取对话历史消息
func (s *aiServiceImpl) GetMessages(ctx context.Context, conversationID int64) ([]*Message, error) {
	msgs, err := s.msgRepo.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	result := make([]*Message, len(msgs))
	for i, m := range msgs {
		result[i] = toMessage(m)
	}
	return result, nil
}

// DeleteConversation 删除对话及其所有消息
func (s *aiServiceImpl) DeleteConversation(ctx context.Context, conversationID int64) error {
	// 先删消息再删对话
	if err := s.msgRepo.DeleteByConversation(ctx, conversationID); err != nil {
		return err
	}
	return s.convRepo.Delete(ctx, conversationID)
}

// RegisterTool 注册自定义 Tool
func (s *aiServiceImpl) RegisterTool(name string, handler ToolHandler) {
	s.executor.RegisterTool(name, handler)
}

// GetToolExecuteFunc 获取 Tool 执行函数
func (s *aiServiceImpl) GetToolExecuteFunc() func(ctx context.Context, clusterID string, tc *llm.ToolCall) (string, error) {
	return s.executor.Execute
}

// GetToolDefs 获取 Tool 定义列表
func (s *aiServiceImpl) GetToolDefs() []llm.ToolDefinition {
	return GetToolDefinitions()
}

// toConversation 转换 DB 模型为 API 类型
func toConversation(c *database.AIConversation) *Conversation {
	return &Conversation{
		ID:                c.ID,
		UserID:            c.UserID,
		ClusterID:         c.ClusterID,
		Title:             c.Title,
		MessageCount:      c.MessageCount,
		TotalInputTokens:  c.TotalInputTokens,
		TotalOutputTokens: c.TotalOutputTokens,
		TotalToolCalls:    c.TotalToolCalls,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

// toMessage 转换 DB 模型为 API 类型
func toMessage(m *database.AIMessage) *Message {
	return &Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           m.Role,
		Content:        m.Content,
		ToolCalls:      m.ToolCalls,
		CreatedAt:      m.CreatedAt,
	}
}

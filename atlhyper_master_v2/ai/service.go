// atlhyper_master_v2/ai/service.go
// AIService 实现
// 提供会话 CRUD + Chat 入口
package ai

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	_ "AtlHyper/atlhyper_master_v2/ai/llm/gemini" // 注册 gemini provider
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/service/operations"
)

// ServiceConfig AI 服务配置
type ServiceConfig struct {
	Provider    string // gemini
	APIKey      string
	Model       string
	ToolTimeout time.Duration // Tool 执行超时，默认 30s
}

// aiServiceImpl AIService 实现
type aiServiceImpl struct {
	llmClient llm.LLMClient
	executor  *toolExecutor
	convRepo  database.AIConversationRepository
	msgRepo   database.AIMessageRepository
}

// NewService 创建 AIService
func NewService(
	cfg ServiceConfig,
	ops *operations.CommandService,
	bus mq.Producer,
	convRepo database.AIConversationRepository,
	msgRepo database.AIMessageRepository,
) (AIService, error) {
	// 通过工厂创建 LLM Client（由 provider 注册机制决定具体实现）
	client, err := llm.New(llm.Config{
		Provider: cfg.Provider,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
	})
	if err != nil {
		return nil, err
	}

	// Tool 超时默认 30s
	toolTimeout := cfg.ToolTimeout
	if toolTimeout == 0 {
		toolTimeout = 30 * time.Second
	}

	return &aiServiceImpl{
		llmClient: client,
		executor:  newToolExecutor(ops, bus, toolTimeout),
		convRepo:  convRepo,
		msgRepo:   msgRepo,
	}, nil
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

// toConversation 转换 DB 模型为 API 类型
func toConversation(c *database.AIConversation) *Conversation {
	return &Conversation{
		ID:           c.ID,
		UserID:       c.UserID,
		ClusterID:    c.ClusterID,
		Title:        c.Title,
		MessageCount: c.MessageCount,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
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

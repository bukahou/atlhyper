// atlhyper_master_v2/ai/service.go
// AIService 实现
// 提供会话 CRUD + Chat 入口
package ai

import (
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/budget"
	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

// ServiceConfig AI 服务配置（仅用于 Tool 超时等非敏感配置）
type ServiceConfig struct {
	BedrockRegion string // Bedrock 区域（Master 配置 MASTER_AI_BEDROCK_REGION；所有 bedrock provider 共用）
}

// aiServiceImpl AIService 实现
// 注意：不再缓存 llmClient，每次 Chat 时从 DB 获取最新配置并创建客户端
type aiServiceImpl struct {
	providerRepo database.AIProviderRepository
	modelRepo    database.AIProviderModelRepository
	budgetRepo   database.AIRoleBudgetRepository
	budgets      *budget.Ledger // nil = 不做预算控制（未接预算仓库时）
	tools        *toolkit.Registry
	convRepo     database.AIConversationRepository
	msgRepo      database.AIMessageRepository

	bedrockRegion string // 来自 Master 配置，不来自数据库

	// newClient 按配置创建模型客户端（默认 llm.NewLLMClient；测试替换）
	newClient func(llm.Config) (llm.LLMClient, error)
}

// createClient 按配置创建模型客户端
func (s *aiServiceImpl) createClient(cfg llm.Config) (llm.LLMClient, error) {
	if s.newClient != nil {
		return s.newClient(cfg)
	}
	return llm.NewLLMClient(cfg)
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

// ownedConversation 取会话并确认属于 userID；不存在与不属于同样返回 ErrConversationNotFound
func (s *aiServiceImpl) ownedConversation(ctx context.Context, userID, conversationID int64) (*database.AIConversation, error) {
	conv, err := s.convRepo.GetByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("获取对话失败: %w", err)
	}
	if conv == nil || conv.UserID != userID {
		return nil, ErrConversationNotFound
	}
	return conv, nil
}

// GetMessages 获取对话历史消息
func (s *aiServiceImpl) GetMessages(ctx context.Context, userID, conversationID int64) ([]*Message, error) {
	if _, err := s.ownedConversation(ctx, userID, conversationID); err != nil {
		return nil, err
	}
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
func (s *aiServiceImpl) DeleteConversation(ctx context.Context, userID, conversationID int64) error {
	if _, err := s.ownedConversation(ctx, userID, conversationID); err != nil {
		return err
	}
	// 先删消息再删对话
	if err := s.msgRepo.DeleteByConversation(ctx, conversationID); err != nil {
		return err
	}
	return s.convRepo.Delete(ctx, conversationID)
}

// Complete 单轮 LLM 调用（无 Tool，用于 background 摘要等）
func (s *aiServiceImpl) Complete(ctx context.Context, req *CompleteRequest) (*CompleteResult, error) {
	start := time.Now()
	role := req.Role
	if role == "" {
		role = RoleBackground
	}

	roleCfg, err := s.loadAIConfigForRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("AI 配置错误: %w", err)
	}

	llmClient, err := s.createClient(roleCfg.Config)
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, CodeConfig, err)
		return nil, fmt.Errorf("创建 LLM 客户端失败: %w", err)
	}
	defer llmClient.Close()

	stream, err := llmClient.ChatStream(ctx, &llm.Request{
		SystemPrompt: req.SystemPrompt,
		Messages:     []llm.Message{{Role: "user", Content: req.UserPrompt}},
	})
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, ErrorCode(err), err)
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 收集完整响应（流出错或没按约定收尾 → 报错，不把半截文本当结果）
	res, err := llm.ReadStream(ctx, stream, nil)
	var inputTokens, outputTokens int
	if res.Usage != nil {
		inputTokens = res.Usage.InputTokens
		outputTokens = res.Usage.OutputTokens
	}
	// 调过模型就记账（失败也记）；用不随请求取消的 ctx
	s.RecordUsage(context.WithoutCancel(ctx), role, roleCfg.ProviderID, inputTokens, outputTokens)
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, ErrorCode(err), err)
		return nil, fmt.Errorf("LLM 流错误: %w", err)
	}
	text := res.Text

	// 成功调用，清除之前的错误状态
	s.clearProviderError(ctx, roleCfg.ProviderID)

	return &CompleteResult{
		Response:     text,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		ProviderID:   roleCfg.ProviderID,
		ProviderName: roleCfg.ProviderName,
		Model:        roleCfg.Config.Model,
		Duration:     time.Since(start),
	}, nil
}

// Tools 工具注册表
func (s *aiServiceImpl) Tools() *toolkit.Registry {
	return s.tools
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
	decoded := decodeStoredMessage(m)
	return &Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Role:           m.Role,
		Content:        decoded.Content,
		ToolCalls:      decoded.ToolCalls,
		ToolResult:     decoded.ToolResult,
		CreatedAt:      m.CreatedAt,
	}
}

// recordProviderError 记日志并把 Provider 标成出错（原始错误只进日志）
func (s *aiServiceImpl) recordProviderError(ctx context.Context, providerID int64, code string, err error) {
	log.Warn("模型提供商调用失败", "provider", providerID, "code", code, "err", err)
	s.markProviderError(ctx, providerID, code)
}

// markProviderError 把 Provider 标成出错（前端 AI 设置页可见），调用处已记过日志时用它。
// 只记错误码对应的通用说明，⛔ 不暴露内部 URL / IP / 响应体。
// 用不随请求取消的 ctx：请求被取消、超时之后状态照样要写进库
func (s *aiServiceImpl) markProviderError(ctx context.Context, providerID int64, code string) {
	if err := s.providerRepo.UpdateStatus(context.WithoutCancel(ctx), providerID, "error", errorSummary(code)); err != nil {
		log.Warn("更新 Provider 错误状态失败", "provider", providerID, "err", err)
	}
}

// clearProviderError 成功调用后清除错误状态（连同上一次的错误说明与时间）
func (s *aiServiceImpl) clearProviderError(ctx context.Context, providerID int64) {
	if err := s.providerRepo.UpdateStatus(context.WithoutCancel(ctx), providerID, "active", ""); err != nil {
		log.Warn("清除 Provider 错误状态失败", "provider", providerID, "err", err)
	}
}

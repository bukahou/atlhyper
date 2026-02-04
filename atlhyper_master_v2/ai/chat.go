// atlhyper_master_v2/ai/chat.go
// Chat 核心逻辑
// 多轮 Tool Calling 循环 + SSE channel
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AI-Chat")

// loadAIConfig 从数据库加载 AI 配置
// 使用 2 表设计: ai_active_config (当前配置) + ai_providers (提供商配置)
func (s *aiServiceImpl) loadAIConfig(ctx context.Context) (*llm.Config, error) {
	// 1. 获取当前配置
	active, err := s.activeRepo.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 AI 配置失败: %w", err)
	}
	if active == nil {
		return nil, fmt.Errorf("AI 配置未初始化")
	}

	// 2. 检查是否启用
	if !active.Enabled {
		return nil, fmt.Errorf("AI 功能未启用")
	}

	// 3. 检查是否设置了提供商
	if active.ProviderID == nil {
		return nil, fmt.Errorf("未设置 AI 提供商")
	}

	// 4. 获取提供商配置
	provider, err := s.providerRepo.GetByID(ctx, *active.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("获取 AI 提供商失败: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("AI 提供商不存在: %d", *active.ProviderID)
	}

	// 5. 构建 LLM 配置
	return &llm.Config{
		Provider: provider.Provider,
		APIKey:   provider.APIKey,
		Model:    provider.Model,
	}, nil
}

const maxToolRounds = 5             // 最大 Tool 调用轮数
const maxToolCallsPerRound = 5      // 每轮最多 Tool Call 数
const chatTimeout = 3 * time.Minute // Chat 全局超时
const maxHistoryMessages = 20       // 最大历史消息加载数

// Chat 发送消息并获取流式响应
func (s *aiServiceImpl) Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// 1. 获取对话信息
	conv, err := s.convRepo.GetByID(ctx, req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("获取对话失败: %w", err)
	}
	if conv == nil {
		return nil, fmt.Errorf("对话不存在: %d", req.ConversationID)
	}

	// 2. 加载历史消息
	dbMsgs, err := s.msgRepo.ListByConversation(ctx, req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("加载历史消息失败: %w", err)
	}

	// 3. 限制历史消息数量，只保留最近 N 条
	if len(dbMsgs) > maxHistoryMessages {
		dbMsgs = dbMsgs[len(dbMsgs)-maxHistoryMessages:]
	}

	// 4. 转换为 LLM 消息格式
	messages := buildLLMMessages(dbMsgs)

	// 5. 追加用户消息
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: req.Message,
	})

	// 6. 持久化用户消息
	userMsg := &database.AIMessage{
		ConversationID: req.ConversationID,
		Role:           "user",
		Content:        req.Message,
		CreatedAt:      time.Now(),
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		log.Warn("持久化用户消息失败", "err", err)
	}

	// 7. 启动异步 Chat 循环（带全局超时）
	chatCtx, cancel := context.WithTimeout(ctx, chatTimeout)
	ch := make(chan *ChatChunk, 64)
	go func() {
		defer close(ch)
		defer cancel()
		s.chatLoop(chatCtx, req.ClusterID, req.ConversationID, messages, ch)
	}()

	return ch, nil
}

// chatLoop 多轮 Tool Calling 循环
func (s *aiServiceImpl) chatLoop(ctx context.Context, clusterID string, convID int64, messages []llm.Message, ch chan<- *ChatChunk) {
	startTime := time.Now()

	// 每次 Chat 从 DB 获取最新配置并创建 LLM Client（支持热更新）
	llmCfg, err := s.loadAIConfig(ctx)
	if err != nil {
		sendError(ch, fmt.Sprintf("AI 配置错误: %v", err))
		return
	}

	llmClient, err := llm.New(*llmCfg)
	if err != nil {
		sendError(ch, fmt.Sprintf("创建 LLM 客户端失败: %v", err))
		return
	}
	defer llmClient.Close()

	log.Info("对话开始",
		"conv", convID,
		"provider", llmCfg.Provider,
		"model", llmCfg.Model,
	)

	systemPrompt := BuildSystemPrompt()
	tools := GetToolDefinitions()

	var assistantContent string
	var totalToolCalls int    // 统计总指令数
	var toolRounds int        // 统计有 Tool 调用的轮次（用户关心的）
	var totalInputTokens int  // 累计输入 Token
	var totalOutputTokens int // 累计输出 Token

	for round := 0; round < maxToolRounds; round++ {
		remaining := maxToolRounds - round

		// 注入剩余次数提示（让 AI 合理规划 Tool 调用）
		roundHint := fmt.Sprintf(
			"\n\n[系统提示] 你还有 %d 次 Tool 调用机会（每次可并行调用多个 query_cluster）。请合理规划，在机会用完前完成分析。",
			remaining,
		)

		// 调用 LLM
		llmReq := &llm.Request{
			SystemPrompt: systemPrompt + roundHint,
			Messages:     messages,
			Tools:        tools,
		}

		stream, err := llmClient.ChatStream(ctx, llmReq)
		if err != nil {
			sendError(ch, fmt.Sprintf("LLM 调用失败: %v", err))
			return
		}

		// 读取流式响应
		text, toolCalls, usage, err := s.readLLMStream(ctx, stream, ch)
		if err != nil {
			log.Error("LLM 流读取错误", "conv", convID, "err", err)
			sendError(ch, fmt.Sprintf("AI 服务错误: %v", err))
			return
		}

		// 累计 token 使用量
		if usage != nil {
			totalInputTokens += usage.InputTokens
			totalOutputTokens += usage.OutputTokens
		}

		assistantContent += text

		// 没有 Tool Call → 对话结束
		if len(toolCalls) == 0 {
			// 持久化最终的 assistant 消息（纯文本，tool_use 已在前面保存）
			s.persistFinalAssistantMessage(ctx, convID, assistantContent)
			// 累加统计到对话
			s.accumulateConversationStats(ctx, convID, totalInputTokens, totalOutputTokens, totalToolCalls)
			// 发送 done 并附带统计信息
			ch <- &ChatChunk{
				Type: "done",
				Stats: &ChatStats{
					Rounds:         toolRounds, // 只统计有 Tool 调用的轮次
					TotalToolCalls: totalToolCalls,
					InputTokens:    totalInputTokens,
					OutputTokens:   totalOutputTokens,
				},
			}
			log.Info("对话完成",
				"conv", convID,
				"rounds", toolRounds,
				"tools", totalToolCalls,
				"tokens", totalInputTokens+totalOutputTokens,
				"duration", logger.Duration(time.Since(startTime)),
			)
			return
		}

		// 有 Tool Call → 执行并继续
		toolRounds++ // 只有有 Tool 调用时才算一轮

		// 限制每轮 Tool Call 数量
		if len(toolCalls) > maxToolCallsPerRound {
			toolCalls = toolCalls[:maxToolCallsPerRound]
			ch <- &ChatChunk{Type: "text", Content: "\n[系统: 本轮 Tool 调用过多，已截断为前5个]\n"}
		}
		totalToolCalls += len(toolCalls) // 累计指令数

		// 添加 assistant 消息（含 tool calls）到历史
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   text,
			ToolCalls: toolCalls,
		})

		// 【重要】先に assistant メッセージを保存（tool_result より前に必要）
		// Anthropic API は tool_use の後に tool_result が来ることを要求する
		s.persistAssistantWithToolCalls(ctx, convID, text, toolCalls)

		// 执行每个 Tool Call
		for _, tc := range toolCalls {
			// 通知前端正在执行 tool
			ch <- &ChatChunk{
				Type:   "tool_call",
				Tool:   tc.Name,
				Params: tc.Params,
			}

			// 执行（传递 ctx 以支持全局超时取消）
			result, err := s.executor.Execute(ctx, clusterID, &tc)
			if err != nil {
				result = fmt.Sprintf("Tool 执行失败: %v", err)
			}

			// 通知前端 tool 结果
			ch <- &ChatChunk{
				Type:    "tool_result",
				Tool:    tc.Name,
				Content: truncate(result, 4000),
			}

			// 添加 tool 结果到历史（截断防 token 爆炸）
			toolResult := &llm.ToolResult{
				CallID:  tc.ID,
				Name:    tc.Name,
				Content: truncate(result, 32000),
			}
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolResult: toolResult,
			})

			// 持久化 tool 结果消息（重要：Anthropic 要求 tool_use 后必须有 tool_result）
			s.persistToolResultMessage(ctx, convID, toolResult)
		}

		// 重置文本累积（下一轮 LLM 输出新文本）
		assistantContent += "\n"
	}

	// 超过最大轮数，要求 AI 基于已有信息给出结论
	// 最后一次调用不提供 tools，强制 AI 输出文本结论（不算 toolRounds）
	llmReq := &llm.Request{
		SystemPrompt: systemPrompt + "\n\n[系统提示] Tool 调用次数已用完。请根据已获取的信息给出分析结论。不要再调用 Tool。",
		Messages:     messages,
		Tools:        nil, // 不提供 tools，强制文本输出
	}
	stream, err := llmClient.ChatStream(ctx, llmReq)
	if err != nil {
		sendError(ch, fmt.Sprintf("LLM 调用失败: %v", err))
		return
	}
	text, _, usage, _ := s.readLLMStream(ctx, stream, ch)
	if usage != nil {
		totalInputTokens += usage.InputTokens
		totalOutputTokens += usage.OutputTokens
	}
	assistantContent += text
	s.persistFinalAssistantMessage(ctx, convID, assistantContent)
	// 累加统计到对话
	s.accumulateConversationStats(ctx, convID, totalInputTokens, totalOutputTokens, totalToolCalls)
	// 发送 done 并附带统计信息
	ch <- &ChatChunk{
		Type: "done",
		Stats: &ChatStats{
			Rounds:         toolRounds, // 只统计有 Tool 调用的轮次
			TotalToolCalls: totalToolCalls,
			InputTokens:    totalInputTokens,
			OutputTokens:   totalOutputTokens,
		},
	}
	log.Info("对话完成",
		"conv", convID,
		"rounds", toolRounds,
		"tools", totalToolCalls,
		"tokens", totalInputTokens+totalOutputTokens,
		"duration", logger.Duration(time.Since(startTime)),
	)
}

// readLLMStream 读取 LLM 流式响应
// 返回: 累积文本, ToolCall 列表, Usage, 错误
func (s *aiServiceImpl) readLLMStream(ctx context.Context, stream <-chan *llm.Chunk, ch chan<- *ChatChunk) (string, []llm.ToolCall, *llm.Usage, error) {
	var text string
	var toolCalls []llm.ToolCall
	var usage *llm.Usage

	for chunk := range stream {
		select {
		case <-ctx.Done():
			return text, toolCalls, usage, ctx.Err()
		default:
		}

		switch chunk.Type {
		case llm.ChunkText:
			text += chunk.Content
			ch <- &ChatChunk{Type: "text", Content: chunk.Content}

		case llm.ChunkToolCall:
			if chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}

		case llm.ChunkDone:
			usage = chunk.Usage
			return text, toolCalls, usage, nil

		case llm.ChunkError:
			log.Debug("收到 ChunkError", "err", chunk.Error)
			return text, toolCalls, usage, chunk.Error
		}
	}

	return text, toolCalls, usage, nil
}

// persistFinalAssistantMessage 持久化最终的 assistant 消息（纯文本回答）
// tool_use 相关的 assistant 消息已在 persistAssistantWithToolCalls 中保存
func (s *aiServiceImpl) persistFinalAssistantMessage(ctx context.Context, convID int64, content string) {
	// 只有当有内容时才保存（避免空消息）
	if content != "" {
		msg := &database.AIMessage{
			ConversationID: convID,
			Role:           "assistant",
			Content:        content,
			CreatedAt:      time.Now(),
		}

		if err := s.msgRepo.Create(ctx, msg); err != nil {
			log.Warn("持久化 assistant 消息失败", "err", err)
		}
	}

	// 更新对话 message_count 和 updated_at
	s.updateConversationStats(ctx, convID)
}

// updateConversationStats 更新对话统计信息
func (s *aiServiceImpl) updateConversationStats(ctx context.Context, convID int64) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err == nil && conv != nil {
		// 从实际消息数计算
		msgs, _ := s.msgRepo.ListByConversation(ctx, convID)
		conv.MessageCount = len(msgs)
		conv.UpdatedAt = time.Now()
		s.convRepo.Update(ctx, conv)
	}
}

// accumulateConversationStats 累加对话统计（token、指令数）
func (s *aiServiceImpl) accumulateConversationStats(ctx context.Context, convID int64, inputTokens, outputTokens, toolCalls int) {
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err != nil || conv == nil {
		log.Warn("累加对话统计失败：获取对话失败", "conv", convID, "err", err)
		return
	}

	// 累加统计
	conv.TotalInputTokens += int64(inputTokens)
	conv.TotalOutputTokens += int64(outputTokens)
	conv.TotalToolCalls += toolCalls

	// 更新消息数
	msgs, _ := s.msgRepo.ListByConversation(ctx, convID)
	conv.MessageCount = len(msgs)
	conv.UpdatedAt = time.Now()

	if err := s.convRepo.Update(ctx, conv); err != nil {
		log.Warn("累加对话统计失败：更新对话失败", "conv", convID, "err", err)
	}
}

// persistToolResultMessage 持久化 tool 结果消息
// Anthropic 要求 tool_use 后必须有 tool_result，所以必须保存以便历史加载
func (s *aiServiceImpl) persistToolResultMessage(ctx context.Context, convID int64, toolResult *llm.ToolResult) {
	// 用 JSON 存储 tool_result 结构
	trJSON, _ := json.Marshal(toolResult)
	msg := &database.AIMessage{
		ConversationID: convID,
		Role:           "tool",
		Content:        string(trJSON), // tool_result JSON
		CreatedAt:      time.Now(),
	}

	if err := s.msgRepo.Create(ctx, msg); err != nil {
		log.Warn("持久化 tool 结果消息失败", "err", err)
	}
}

// persistAssistantWithToolCalls 持久化带 tool_use 的 assistant 消息
// 必须在 tool_result 之前保存，以满足 Anthropic API 的消息顺序要求
func (s *aiServiceImpl) persistAssistantWithToolCalls(ctx context.Context, convID int64, content string, toolCalls []llm.ToolCall) {
	msg := &database.AIMessage{
		ConversationID: convID,
		Role:           "assistant",
		Content:        content,
		CreatedAt:      time.Now(),
	}

	if len(toolCalls) > 0 {
		tcJSON, _ := json.Marshal(toolCalls)
		msg.ToolCalls = string(tcJSON)
	}

	if err := s.msgRepo.Create(ctx, msg); err != nil {
		log.Warn("持久化 assistant(tool_use) 消息失败", "err", err)
	}
}

// buildLLMMessages 将 DB 消息转换为 LLM 消息格式
func buildLLMMessages(dbMsgs []*database.AIMessage) []llm.Message {
	var messages []llm.Message
	for _, m := range dbMsgs {
		msg := llm.Message{
			Role:    m.Role,
			Content: m.Content,
		}

		// 解析 tool_calls (assistant 消息)
		if m.Role == "assistant" && m.ToolCalls != "" {
			var tcs []llm.ToolCall
			if err := json.Unmarshal([]byte(m.ToolCalls), &tcs); err == nil {
				msg.ToolCalls = tcs
			}
		}

		// 解析 tool_result (tool 消息)
		// Anthropic 要求 tool_use 后必须有 tool_result
		if m.Role == "tool" && m.Content != "" {
			var tr llm.ToolResult
			if err := json.Unmarshal([]byte(m.Content), &tr); err == nil {
				msg.ToolResult = &tr
				msg.Content = "" // tool 消息内容在 ToolResult 中
			}
		}

		messages = append(messages, msg)
	}
	return messages
}

// sendError 发送错误到 channel
func sendError(ch chan<- *ChatChunk, msg string) {
	log.Warn("发送错误", "msg", msg)
	ch <- &ChatChunk{Type: "error", Content: msg}
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "...(已截断)"
}

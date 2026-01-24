// atlhyper_master_v2/ai/chat.go
// Chat 核心逻辑
// 多轮 Tool Calling 循环 + SSE channel
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

const maxToolRounds = 5 // 最大 Tool 调用轮数

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

	// 3. 转换为 LLM 消息格式
	messages := buildLLMMessages(dbMsgs)

	// 4. 追加用户消息
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: req.Message,
	})

	// 5. 持久化用户消息
	userMsg := &database.AIMessage{
		ConversationID: req.ConversationID,
		Role:           "user",
		Content:        req.Message,
		CreatedAt:      time.Now(),
	}
	if err := s.msgRepo.Create(ctx, userMsg); err != nil {
		log.Printf("[AI-Chat] 持久化用户消息失败: %v", err)
	}

	// 6. 启动异步 Chat 循环
	ch := make(chan *ChatChunk, 64)
	go func() {
		defer close(ch)
		s.chatLoop(ctx, req.ClusterID, req.ConversationID, messages, ch)
	}()

	return ch, nil
}

// chatLoop 多轮 Tool Calling 循环
func (s *aiServiceImpl) chatLoop(ctx context.Context, clusterID string, convID int64, messages []llm.Message, ch chan<- *ChatChunk) {
	systemPrompt := BuildSystemPrompt()
	tools := GetToolDefinitions()

	var assistantContent string
	var allToolCalls []llm.ToolCall

	for round := 0; round < maxToolRounds; round++ {
		// 调用 LLM
		llmReq := &llm.Request{
			SystemPrompt: systemPrompt,
			Messages:     messages,
			Tools:        tools,
		}

		stream, err := s.llmClient.ChatStream(ctx, llmReq)
		if err != nil {
			sendError(ch, fmt.Sprintf("LLM 调用失败: %v", err))
			return
		}

		// 读取流式响应
		text, toolCalls, err := s.readLLMStream(ctx, stream, ch)
		if err != nil {
			sendError(ch, fmt.Sprintf("读取响应失败: %v", err))
			return
		}

		assistantContent += text

		// 没有 Tool Call → 对话结束
		if len(toolCalls) == 0 {
			// 持久化 assistant 消息
			s.persistAssistantMessage(ctx, convID, assistantContent, allToolCalls)
			ch <- &ChatChunk{Type: "done"}
			return
		}

		// 有 Tool Call → 执行并继续
		allToolCalls = append(allToolCalls, toolCalls...)

		// 添加 assistant 消息（含 tool calls）到历史
		messages = append(messages, llm.Message{
			Role:      "assistant",
			Content:   text,
			ToolCalls: toolCalls,
		})

		// 执行每个 Tool Call
		for _, tc := range toolCalls {
			// 通知前端正在执行 tool
			ch <- &ChatChunk{
				Type:   "tool_call",
				Tool:   tc.Name,
				Params: tc.Params,
			}

			// 执行
			result, err := s.executor.Execute(clusterID, &tc)
			if err != nil {
				result = fmt.Sprintf("Tool 执行失败: %v", err)
			}

			// 通知前端 tool 结果
			ch <- &ChatChunk{
				Type:    "tool_result",
				Tool:    tc.Name,
				Content: truncate(result, 2000),
			}

			// 添加 tool 结果到历史
			messages = append(messages, llm.Message{
				Role: "tool",
				ToolResult: &llm.ToolResult{
					CallID:  tc.ID,
					Name:    tc.Name,
					Content: result,
				},
			})
		}

		// 重置文本累积（下一轮 LLM 输出新文本）
		assistantContent += "\n"
	}

	// 超过最大轮数
	sendError(ch, "分析轮数已达上限，请简化问题重试")
}

// readLLMStream 读取 LLM 流式响应
// 返回: 累积文本, ToolCall 列表, 错误
func (s *aiServiceImpl) readLLMStream(ctx context.Context, stream <-chan *llm.Chunk, ch chan<- *ChatChunk) (string, []llm.ToolCall, error) {
	var text string
	var toolCalls []llm.ToolCall

	for chunk := range stream {
		select {
		case <-ctx.Done():
			return text, toolCalls, ctx.Err()
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
			return text, toolCalls, nil

		case llm.ChunkError:
			return text, toolCalls, chunk.Error
		}
	}

	return text, toolCalls, nil
}

// persistAssistantMessage 持久化 assistant 消息
func (s *aiServiceImpl) persistAssistantMessage(ctx context.Context, convID int64, content string, toolCalls []llm.ToolCall) {
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
		log.Printf("[AI-Chat] 持久化 assistant 消息失败: %v", err)
	}

	// 更新对话 message_count
	conv, err := s.convRepo.GetByID(ctx, convID)
	if err == nil && conv != nil {
		conv.MessageCount += 2 // user + assistant
		conv.UpdatedAt = time.Now()
		s.convRepo.Update(ctx, conv)
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

		// 解析 tool_calls
		if m.Role == "assistant" && m.ToolCalls != "" {
			var tcs []llm.ToolCall
			if err := json.Unmarshal([]byte(m.ToolCalls), &tcs); err == nil {
				msg.ToolCalls = tcs
			}
		}

		messages = append(messages, msg)
	}
	return messages
}

// sendError 发送错误到 channel
func sendError(ch chan<- *ChatChunk, msg string) {
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

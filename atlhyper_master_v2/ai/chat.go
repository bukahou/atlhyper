// atlhyper_master_v2/ai/chat.go
// Chat 核心逻辑
// 多轮 Tool Calling 循环 + SSE channel
package ai

import (
	"AtlHyper/common/truncate"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/prompts"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AI-Chat")

const maxToolRounds = 5             // 最大 Tool 调用轮数
const chatTimeout = 3 * time.Minute // Chat 全局超时
const maxHistoryMessages = 20       // 最大历史消息加载数

// Chat 发送消息并获取流式响应
func (s *aiServiceImpl) Chat(ctx context.Context, req *ChatRequest) (<-chan *ChatChunk, error) {
	// 1. 获取对话信息（只能续聊自己的会话）
	if _, err := s.ownedConversation(ctx, req.UserID, req.ConversationID); err != nil {
		return nil, err
	}

	// 告警分析：用户消息由后端按界面语言拼
	if len(req.Alerts) > 0 {
		req.Message = prompts.BuildAlertAnalysisMessage(req.Alerts, req.Language)
	}
	if req.Message == "" {
		return nil, ErrEmptyMessage
	}

	// 2. 加载历史消息
	dbMsgs, err := s.msgRepo.ListByConversation(ctx, req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("加载历史消息失败: %w", err)
	}

	// 3. 限制历史消息数量：只保留最近的若干整轮（不从一轮中间切，否则工具结果会失去它的调用）
	if len(dbMsgs) > maxHistoryMessages {
		dbMsgs = dbMsgs[recentTurnsStart(len(dbMsgs), maxHistoryMessages, func(i int) string { return dbMsgs[i].Role }):]
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
		s.chatLoop(chatCtx, req.ClusterID, req.ConversationID, req.Allow, messages, ch)
	}()

	return ch, nil
}

// chatLoop 多轮 Tool Calling 循环
func (s *aiServiceImpl) chatLoop(ctx context.Context, clusterID string, convID int64, allow PermissionCheck, messages []llm.Message, ch chan<- *ChatChunk) {
	startTime := time.Now()

	// 每次 Chat 从 DB 获取最新配置并创建 LLM Client（支持热更新）
	roleCfg, err := s.loadAIConfigForRole(ctx, RoleChat)
	if err != nil {
		log.Warn("对话没有可用的 AI 配置", "err", err)
		sendError(ch, configErrorCode(err))
		return
	}

	llmClient, err := s.createClient(roleCfg.Config)
	if err != nil {
		s.recordProviderError(ctx, roleCfg.ProviderID, CodeConfig, err)
		sendError(ch, CodeConfig)
		return
	}
	defer llmClient.Close()

	// 根据 Provider 上下文窗口创建 ContextManager
	ctxMgr := NewContextManager(roleCfg.ContextWindow)
	maxToolResult := toolResultMaxLen(roleCfg.ContextWindow)

	log.Info("对话开始",
		"conv", convID,
		"provider", roleCfg.Provider,
		"model", roleCfg.Model,
		"contextWindow", roleCfg.ContextWindow,
	)

	// 系统提示词在整个对话里逐字不变（缓存断点打在它上面）；轮次预算附在每轮最后一条工具结果上
	systemPrompt := prompts.BuildChatPrompt(MaxToolCallsPerRound)
	tools := s.tools.Definitions(allow) // 已注册 ∩ 调用者有权限

	var finalText string      // 最终回答：最后一次调模型的正文（前几轮的正文已随各自的工具调用落库）
	var totalToolCalls int    // 统计总指令数
	var toolRounds int        // 统计有 Tool 调用的轮次（用户关心的）
	var totalInputTokens int  // 累计输入 Token
	var totalOutputTokens int // 累计输出 Token
	callRound := 0            // 第几次调模型（流里的 round）

	// 用量记账：只要调过模型，不论成功、出错还是被中途停止都记一次（预算与对话统计）。
	// 成功时在发 done 之前记，让前端收到 done 后重取的统计是新的；其余出口由 defer 兜底。
	// 用不随请求取消的 ctx：用户点了停止，已经花掉的用量照样要写进库
	modelCalled := false
	settleUsage := sync.OnceFunc(func() {
		if !modelCalled {
			return
		}
		acctCtx := context.WithoutCancel(ctx)
		s.accumulateConversationStats(acctCtx, convID, totalInputTokens, totalOutputTokens, totalToolCalls)
		s.RecordUsage(acctCtx, RoleChat, roleCfg.ProviderID, totalInputTokens, totalOutputTokens)
	})
	defer settleUsage()

	// readRound 调一次模型并读完这一轮（文字边收边转给前端）；失败时已发出错误分段
	readRound := func(req *llm.Request) (llm.StreamResult, bool) {
		modelCalled = true
		callRound++
		round := callRound
		stream, err := llmClient.ChatStream(ctx, req)
		if err != nil {
			code := ErrorCode(err)
			log.Error("调用模型失败", "conv", convID, "provider", roleCfg.Provider, "code", code, "err", err)
			sendError(ch, code)
			return llm.StreamResult{}, false
		}
		res, err := llm.ReadStream(ctx, stream, func(text string) {
			ch <- &ChatChunk{Type: "text", Round: round, Content: text}
		})
		if res.Usage != nil {
			totalInputTokens += res.Usage.InputTokens
			totalOutputTokens += res.Usage.OutputTokens
		}
		if err != nil {
			switch {
			case ctx.Err() != nil:
				log.Info("对话被中止", "conv", convID, "err", ctx.Err())
			default:
				code := ErrorCode(err)
				log.Error("模型的流出错", "conv", convID, "provider", roleCfg.Provider, "code", code, "err", err)
				sendError(ch, code)
			}
			return res, false
		}
		return res, true
	}

	// fit 发送前按轮裁剪历史；裁剪了就告诉前端（下一次调模型那一轮）
	fit := func() []llm.Message {
		fitted, truncated := ctxMgr.FitMessages(systemPrompt, messages)
		if truncated {
			log.Warn("上下文超限，已按轮裁剪历史消息",
				"original", len(messages), "fitted", len(fitted),
				"provider", roleCfg.Provider, "contextWindow", roleCfg.ContextWindow)
			ch <- &ChatChunk{Type: "notice", Round: callRound + 1, Notice: NoticeHistoryTrimmed}
		}
		return fitted
	}

	finish := func() {
		s.persistFinalAssistantMessage(ctx, convID, finalText)
		settleUsage()
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

	for round := 0; round < maxToolRounds; round++ {
		res, ok := readRound(&llm.Request{SystemPrompt: systemPrompt, Messages: fit(), Tools: tools})
		if !ok {
			return
		}
		text, toolCalls := res.Text, res.ToolCalls

		// 没有 Tool Call → 这一轮的正文就是最终回答
		if len(toolCalls) == 0 {
			finalText = text
			finish()
			return
		}

		toolRounds++ // 只有有 Tool 调用时才算一轮
		if len(toolCalls) > MaxToolCallsPerRound {
			toolCalls = toolCalls[:MaxToolCallsPerRound]
			ch <- &ChatChunk{Type: "notice", Round: callRound, Notice: NoticeToolCallsCapped}
		}
		totalToolCalls += len(toolCalls)

		messages = append(messages, llm.Message{
			Role:         "assistant",
			Content:      text,
			ToolCalls:    toolCalls,
			ProviderTurn: res.ProviderTurn, // 整轮回答原样带回（如 Claude 的思考块），不落库
		})
		// 先保存带 tool_use 的 assistant 消息：Anthropic 要求 tool_use 之后紧跟 tool_result
		s.persistAssistantWithToolCalls(ctx, convID, text, toolCalls)

		remaining := maxToolRounds - round - 1
		for i := range toolCalls {
			tc := toolCalls[i]
			ch <- &ChatChunk{Type: "tool_call", Round: callRound, CallID: tc.ID, Tool: tc.Name, Params: tc.Params}

			result, err := s.tools.Execute(ctx, clusterID, allow, &tc)
			isError := err != nil
			if isError {
				result = fmt.Sprintf("Tool 执行失败: %v", err)
			}
			ch <- &ChatChunk{
				Type: "tool_result", Round: callRound, CallID: tc.ID, Tool: tc.Name,
				Content: clipToolResult(result, 4000), IsError: isError,
			}

			// 落库的是干净的结果；本次请求的记录里，本轮最后一条结果附上轮次预算（之后几轮原样保留，前缀不变）
			toolResult := &llm.ToolResult{CallID: tc.ID, Name: tc.Name, Content: clipToolResult(result, maxToolResult), IsError: isError}
			s.persistToolResultMessage(ctx, convID, toolResult)
			inRequest := *toolResult
			if i == len(toolCalls)-1 {
				inRequest.Content += roundBudgetHint(remaining)
			}
			messages = append(messages, llm.Message{Role: "tool", ToolResult: &inRequest})
		}
	}

	// 轮次用尽：最后一次调模型不给工具，强制给出文字结论（最后一条工具结果上已附「已用完」）
	res, ok := readRound(&llm.Request{SystemPrompt: systemPrompt, Messages: fit(), Tools: nil})
	if !ok {
		return
	}
	finalText = res.Text
	finish()
}

// roundBudgetHint 附在一轮最后一条工具结果上的轮次预算（系统提示词保持逐字不变）
func roundBudgetHint(remaining int) string {
	if remaining <= 0 {
		return "\n\n[系统提示] 工具调用次数已用完。请根据已获取的信息给出分析结论，不要再调用工具。"
	}
	return fmt.Sprintf("\n\n[系统提示] 你还有 %d 次工具调用机会（每次可并行调用多个工具），请在用完前完成分析。", remaining)
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
	messages := make([]llm.Message, 0, len(dbMsgs))
	for _, m := range dbMsgs {
		messages = append(messages, decodeStoredMessage(m))
	}
	return messages
}

// decodeStoredMessage 库里的一条消息 → 结构化形态：assistant 的工具调用、tool 消息的结果在库里以 JSON 存放。
// 读旧数据时字段名大小写不敏感（旧行的键是 Go 字段名）
func decodeStoredMessage(m *database.AIMessage) llm.Message {
	msg := llm.Message{Role: m.Role, Content: m.Content}
	if m.Role == "assistant" && m.ToolCalls != "" {
		var tcs []llm.ToolCall
		if err := json.Unmarshal([]byte(m.ToolCalls), &tcs); err == nil {
			msg.ToolCalls = tcs
		} else {
			log.Warn("库里的工具调用解析失败", "message", m.ID, "err", err)
		}
	}
	if m.Role == "tool" && m.Content != "" {
		var tr llm.ToolResult
		if err := json.Unmarshal([]byte(m.Content), &tr); err == nil {
			msg.ToolResult = &tr
			msg.Content = "" // 结果在 ToolResult 里
		} else {
			log.Warn("库里的工具结果解析失败", "message", m.ID, "err", err)
		}
	}
	return msg
}

// sendError 发送错误分段：错误码 + 通用说明（原始错误由调用处记日志，⛔ 不发给前端）
func sendError(ch chan<- *ChatChunk, code string) {
	ch <- &ChatChunk{Type: "error", Code: code, Content: errorSummary(code)}
}

// clipToolResult 工具结果按字符截断后交给模型
func clipToolResult(s string, maxLen int) string {
	return truncate.Runes(s, maxLen, "...(已截断)")
}

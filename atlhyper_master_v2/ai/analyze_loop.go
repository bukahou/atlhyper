// atlhyper_master_v2/ai/analyze_loop.go
// 分析用的多轮 Tool Calling 循环（与 Provider / 预算 / 存储解耦，便于测试）
package ai

import (
	"context"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// evidenceReminderMarker 取证提示的固定前缀（测试据此识别）
const evidenceReminderMarker = "[系统提示] 你还没有调用任何工具取证"

// evidenceReminder 第 1 轮零 tool call 时追加的提示（用户 2026-10-12 裁定 ③）
// 只提醒、不禁止：模型给出理由后仍可直接收尾
const evidenceReminder = evidenceReminderMarker + "。请先调用工具收集证据（事件、Pod / 节点状态、日志、指标等），" +
	"再基于证据给出结论。若确实无需取证，请说明原因后直接输出最终报告。"

// toolExecFunc 执行一次工具调用
type toolExecFunc func(ctx context.Context, tc *llm.ToolCall) (string, error)

// loopParams 循环参数
type loopParams struct {
	SystemPrompt  string
	UserPrompt    string
	MaxRounds     int
	Tools         []llm.ToolDefinition
	MaxToolResult int
}

// loopOutcome 循环结果
type loopOutcome struct {
	Response     string // 最终报告文本；轮次用尽时为空
	Rounds       int
	ToolCalls    int
	InputTokens  int
	OutputTokens int
	Steps        []AnalyzeStep
}

// runToolLoop 多轮 Tool Calling：
//   - 某轮零 tool call → 视为最终报告收尾
//   - 例外：第 1 轮零 tool call 且仍有余轮 → 追加取证提示重试一次（至多一次）
//   - 最后一轮不提供 tools，强制输出
//   - LLM 流出错 → 返回错误（⛔ 不把半截文本当报告）
func runToolLoop(ctx context.Context, client llm.LLMClient, exec toolExecFunc, p loopParams) (*loopOutcome, error) {
	out := &loopOutcome{}
	messages := []llm.Message{{Role: "user", Content: p.UserPrompt}}
	reminded := false

	for round := 1; round <= p.MaxRounds; round++ {
		out.Rounds = round
		remaining := p.MaxRounds - round + 1
		req := &llm.Request{
			SystemPrompt: p.SystemPrompt + fmt.Sprintf(
				"\n\n[系统提示] 剩余调查轮次: %d/%d。如果信息已足够，请返回最终分析报告，不要调用 Tool。",
				remaining, p.MaxRounds),
			Messages: messages,
			Tools:    p.Tools,
		}
		if round == p.MaxRounds {
			req.SystemPrompt = p.SystemPrompt + "\n\n[系统提示] 这是最后一轮。请基于已有信息输出最终分析报告。不要再调用 Tool。"
			req.Tools = nil
		}

		stream, err := client.ChatStream(ctx, req)
		if err != nil {
			return out, fmt.Errorf("第 %d 轮 LLM 调用失败: %w", round, err)
		}
		text, toolCalls, usage, err := collectStreamResponse(stream)
		if usage != nil {
			out.InputTokens += usage.InputTokens
			out.OutputTokens += usage.OutputTokens
		}
		if err != nil {
			return out, fmt.Errorf("第 %d 轮 LLM 流错误: %w", round, err)
		}

		if len(toolCalls) == 0 {
			// ③ 第 1 轮就直接下结论 → 提醒先取证，重试一次
			if round == 1 && !reminded && round < p.MaxRounds && len(p.Tools) > 0 {
				reminded = true
				if text != "" {
					messages = append(messages, llm.Message{Role: "assistant", Content: text})
				}
				messages = append(messages, llm.Message{Role: "user", Content: evidenceReminder})
				continue
			}
			out.Response = text
			return out, nil
		}

		if len(toolCalls) > maxToolCallsPerAnalyze {
			toolCalls = toolCalls[:maxToolCallsPerAnalyze]
		}
		out.ToolCalls += len(toolCalls)
		step := AnalyzeStep{Round: round, Thinking: text}
		messages = append(messages, llm.Message{Role: "assistant", Content: text, ToolCalls: toolCalls})

		for i := range toolCalls {
			tc := toolCalls[i]
			result, err := exec(ctx, &tc)
			if err != nil {
				result = fmt.Sprintf("执行失败: %v", err)
			}
			step.ToolCalls = append(step.ToolCalls, ToolCallRecord{
				Tool: tc.Name, Params: tc.Params, ResultSummary: truncate(result, 500),
			})
			messages = append(messages, llm.Message{
				Role:       "tool",
				ToolResult: &llm.ToolResult{CallID: tc.ID, Name: tc.Name, Content: truncate(result, p.MaxToolResult)},
			})
		}
		out.Steps = append(out.Steps, step)
	}
	return out, nil // 轮次用尽：Response 为空，由调用方判定
}

// collectStreamResponse 收集流式响应（不推送 SSE，静默收集）
// 流以 error 结束时返回该错误，调用方据此区分「出错」与「正常收尾」
func collectStreamResponse(stream <-chan *llm.Chunk) (string, []llm.ToolCall, *llm.Usage, error) {
	var text strings.Builder
	var toolCalls []llm.ToolCall

	for chunk := range stream {
		switch chunk.Type {
		case llm.ChunkText:
			text.WriteString(chunk.Content)
		case llm.ChunkToolCall:
			if chunk.ToolCall != nil {
				toolCalls = append(toolCalls, *chunk.ToolCall)
			}
		case llm.ChunkDone:
			return text.String(), toolCalls, chunk.Usage, nil
		case llm.ChunkError:
			err := chunk.Error
			if err == nil {
				err = fmt.Errorf("LLM 流错误")
			}
			return text.String(), toolCalls, nil, err
		}
	}
	return text.String(), toolCalls, nil, nil
}

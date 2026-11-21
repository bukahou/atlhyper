// atlhyper_master_v2/ai/llm/anthropic/stream.go
// Claude 流式响应 → llm.Chunk
package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go/v2"
)

// messageEventStream SDK 流式响应中用到的部分
type messageEventStream interface {
	Next() bool
	Current() anthropicsdk.MessageStreamEventUnion
	Err() error
}

// RefusalError Claude 的安全审核拒绝回答（HTTP 200，stop_reason = refusal；被拒的请求照样计费）
type RefusalError struct {
	Category    string
	Explanation string
}

func (e *RefusalError) Error() string {
	return fmt.Sprintf("Claude 拒绝回答（%s）：%s", e.Category, e.Explanation)
}

// LLMErrorKind 失败类别
func (e *RefusalError) LLMErrorKind() llm.ErrorKind { return llm.ErrKindRefused }

// apiError SDK 的 HTTP 错误换成通用的 HTTPError（状态码在 SDK 自己的错误类型里）
func apiError(err error) error {
	var sdkErr *anthropicsdk.Error
	if errors.As(err, &sdkErr) {
		return &llm.HTTPError{Provider: "anthropic", StatusCode: sdkErr.StatusCode, Err: err}
	}
	return err
}

// pumpStream 读完整个流：文字增量即时发出；流结束后发出工具调用，最后是带用量与整轮回答（ProviderTurn）的 Done。
// 工具调用等完整回答拼好再发：输出到上限时参数可能不完整，不能执行。
// 没收到 message_stop 就结束的流报未完成：SDK 读到响应体结束就停，不管有没有收尾事件。
func pumpStream(ctx context.Context, stream messageEventStream, ch chan<- *llm.Chunk) {
	var message anthropicsdk.Message
	stopped := false
	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			emit(ctx, ch, &llm.Chunk{Type: llm.ChunkError, Error: fmt.Errorf("解析 Claude 的流式响应: %w", err)})
			return
		}
		if _, ok := event.AsAny().(anthropicsdk.MessageStopEvent); ok {
			stopped = true
			continue
		}
		delta, ok := event.AsAny().(anthropicsdk.ContentBlockDeltaEvent)
		if !ok {
			continue
		}
		if text, ok := delta.Delta.AsAny().(anthropicsdk.TextDelta); ok && text.Text != "" {
			if !emit(ctx, ch, &llm.Chunk{Type: llm.ChunkText, Content: text.Text}) {
				return
			}
		}
	}
	if err := stream.Err(); err != nil {
		emit(ctx, ch, &llm.Chunk{Type: llm.ChunkError, Error: fmt.Errorf("请求 Claude: %w", apiError(err))})
		return
	}

	usage := usageOf(message.Usage)
	if !stopped {
		emit(ctx, ch, &llm.Chunk{Type: llm.ChunkError, Usage: usage, Error: llm.ErrStreamIncomplete})
		return
	}
	switch message.StopReason {
	case anthropicsdk.StopReasonRefusal:
		log.Warn("Claude 拒绝回答", "category", message.StopDetails.Category, "inputTokens", usage.InputTokens)
		emit(ctx, ch, &llm.Chunk{Type: llm.ChunkError, Usage: usage, Error: &RefusalError{
			Category:    string(message.StopDetails.Category),
			Explanation: message.StopDetails.Explanation,
		}})
		return
	case anthropicsdk.StopReasonMaxTokens:
		if hasToolUse(message.Content) {
			emit(ctx, ch, &llm.Chunk{Type: llm.ChunkError, Usage: usage,
				Error: fmt.Errorf("Claude 的回答达到输出上限（%d token），工具调用参数不完整", maxOutputTokens)})
			return
		}
	}

	calls, err := toolCallsOf(message.Content)
	if err != nil {
		emit(ctx, ch, &llm.Chunk{Type: llm.ChunkError, Usage: usage, Error: err})
		return
	}
	for _, call := range calls {
		if !emit(ctx, ch, &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: call}) {
			return
		}
	}
	emit(ctx, ch, &llm.Chunk{Type: llm.ChunkDone, Usage: usage, ProviderTurn: message.ToParam()})
}

// emit 发出一块；调用方取消时放弃，并在缓冲区有空位时留下取消原因
func emit(ctx context.Context, ch chan<- *llm.Chunk, chunk *llm.Chunk) bool {
	select {
	case ch <- chunk:
		return true
	case <-ctx.Done():
		select {
		case ch <- &llm.Chunk{Type: llm.ChunkError, Error: ctx.Err()}:
		default:
		}
		return false
	}
}

// toolCallsOf 取出回答里的工具调用（参数是 JSON 对象文本）
func toolCallsOf(content []anthropicsdk.ContentBlockUnion) ([]*llm.ToolCall, error) {
	var calls []*llm.ToolCall
	for _, block := range content {
		use, ok := block.AsAny().(anthropicsdk.ToolUseBlock)
		if !ok {
			continue
		}
		params := strings.TrimSpace(string(use.Input))
		if params == "" || params == "null" {
			params = "{}"
		}
		if !json.Valid([]byte(params)) {
			return nil, fmt.Errorf("工具 %s 的参数不是合法 JSON", use.Name)
		}
		calls = append(calls, &llm.ToolCall{ID: use.ID, Name: use.Name, Params: params})
	}
	return calls, nil
}

func hasToolUse(content []anthropicsdk.ContentBlockUnion) bool {
	for _, block := range content {
		if _, ok := block.AsAny().(anthropicsdk.ToolUseBlock); ok {
			return true
		}
	}
	return false
}

// usageOf 输入 = 未命中缓存 + 缓存读 + 缓存写（三者单价不同，这里合计；记多不记少）
func usageOf(usage anthropicsdk.Usage) *llm.Usage {
	return &llm.Usage{
		InputTokens:  int(usage.InputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens),
		OutputTokens: int(usage.OutputTokens),
	}
}

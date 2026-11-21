package anthropic_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/llm/anthropic"
	"AtlHyper/atlhyper_master_v2/ai/llm/llmtest"
)

func TestContract(t *testing.T) {
	llmtest.RunContract(t, llmtest.Adapter{
		NewClient: func(t *testing.T, upstream llmtest.Endpoint) llm.LLMClient {
			client, err := anthropic.NewAnthropicClient(llm.Config{
				APIKey: "test-key-placeholder", Model: "claude-contract", BaseURL: upstream.BaseURL,
			})
			if err != nil {
				t.Fatal(err)
			}
			return client
		},
		Wire:             anthropicWire{},
		Path:             "/v1/messages",
		ToolCallIDOnWire: true,
	})
}

// anthropicWire Messages 流式响应：SSE，event 名与 data 里的 type 相同。
// message_start → 每个内容块 content_block_start / delta… / stop → message_delta（停止原因 + 输出用量）→ message_stop
type anthropicWire struct{}

func (anthropicWire) Stream(reply llmtest.Reply) llmtest.Response {
	var frames [][]byte
	event := func(eventType string, body map[string]any) {
		body["type"] = eventType
		payload, _ := json.Marshal(body)
		frame := append([]byte("event: "+eventType+"\ndata: "), payload...)
		frames = append(frames, append(frame, '\n', '\n'))
	}

	event("message_start", map[string]any{"message": map[string]any{
		"id": "msg_contract", "type": "message", "role": "assistant", "model": "claude-contract",
		"content": []any{}, "stop_reason": nil, "stop_sequence": nil,
		"usage": map[string]any{"input_tokens": reply.Usage.InputTokens, "output_tokens": 1},
	}})
	index := 0
	if len(reply.Texts) > 0 {
		event("content_block_start", map[string]any{"index": index, "content_block": map[string]any{"type": "text", "text": ""}})
		for _, text := range reply.Texts {
			event("content_block_delta", map[string]any{"index": index, "delta": map[string]any{"type": "text_delta", "text": text}})
		}
		if !reply.Finished && reply.ToolCall == nil {
			return anthropicStream(frames)
		}
		event("content_block_stop", map[string]any{"index": index})
		index++
	}
	if call := reply.ToolCall; call != nil {
		event("content_block_start", map[string]any{"index": index, "content_block": map[string]any{
			"type": "tool_use", "id": call.ID, "name": call.Name, "input": map[string]any{},
		}})
		for _, fragment := range call.ArgFragments {
			event("content_block_delta", map[string]any{"index": index, "delta": map[string]any{
				"type": "input_json_delta", "partial_json": fragment,
			}})
		}
		event("content_block_stop", map[string]any{"index": index})
	}
	if reply.Finished {
		stopReason := "end_turn"
		if reply.ToolCall != nil {
			stopReason = "tool_use"
		}
		event("message_delta", map[string]any{
			"delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil},
			"usage": map[string]any{"output_tokens": reply.Usage.OutputTokens},
		})
		event("message_stop", map[string]any{})
	}
	return anthropicStream(frames)
}

func anthropicStream(frames [][]byte) llmtest.Response {
	return llmtest.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Frames: frames}
}

// Error 429 带 retry-after: 0，SDK 照常重试（默认 2 次）而不用等，最后报出的仍是 429
func (anthropicWire) Error(status int) llmtest.Response {
	errType := "authentication_error"
	header := http.Header{"Content-Type": {"application/json"}}
	if status == http.StatusTooManyRequests {
		errType = "rate_limit_error"
		header.Set("Retry-After", "0")
	}
	body, _ := json.Marshal(map[string]any{"type": "error", "error": map[string]any{
		"type": errType, "message": "contract test: " + http.StatusText(status),
	}})
	return llmtest.Response{Status: status, Header: header, Frames: [][]byte{body}}
}

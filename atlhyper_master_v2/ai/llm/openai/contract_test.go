package openai_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/llm/llmtest"
	"AtlHyper/atlhyper_master_v2/ai/llm/openai"
)

func TestContract(t *testing.T) {
	llmtest.RunContract(t, llmtest.Adapter{
		NewClient: func(t *testing.T, upstream llmtest.Endpoint) llm.LLMClient {
			client, err := openai.NewOpenAIClient("test-key-placeholder", "gpt-contract", upstream.BaseURL)
			if err != nil {
				t.Fatal(err)
			}
			return client
		},
		Wire:             openAIWire{},
		Path:             "/v1/chat/completions",
		ToolCallIDOnWire: true,
	})
}

// openAIWire Chat Completions 流式响应：SSE，每个 data 是一个 chat.completion.chunk，最后是 data: [DONE]。
// 请求带 stream_options.include_usage，用量在 [DONE] 之前单独一个 choices 为空的分段里。
type openAIWire struct{}

func (openAIWire) Stream(reply llmtest.Reply) llmtest.Response {
	var frames [][]byte
	chunk := func(choices []any, usage any) {
		body := map[string]any{
			"id": "chatcmpl-contract", "object": "chat.completion.chunk", "created": 1700000000,
			"model": "gpt-contract", "choices": choices,
		}
		if usage != nil {
			body["usage"] = usage
		}
		frames = append(frames, sseData(body))
	}
	delta := func(d map[string]any, finishReason any) []any {
		return []any{map[string]any{"index": 0, "delta": d, "finish_reason": finishReason}}
	}

	chunk(delta(map[string]any{"role": "assistant", "content": ""}, nil), nil)
	for _, text := range reply.Texts {
		chunk(delta(map[string]any{"content": text}, nil), nil)
	}
	if call := reply.ToolCall; call != nil {
		// 首个分段带 id 与函数名，之后每段只带一截参数
		chunk(delta(map[string]any{"tool_calls": []any{map[string]any{
			"index": 0, "id": call.ID, "type": "function",
			"function": map[string]any{"name": call.Name, "arguments": ""},
		}}}, nil), nil)
		for _, fragment := range call.ArgFragments {
			chunk(delta(map[string]any{"tool_calls": []any{map[string]any{
				"index": 0, "function": map[string]any{"arguments": fragment},
			}}}, nil), nil)
		}
	}
	if reply.Finished {
		finishReason := "stop"
		if reply.ToolCall != nil {
			finishReason = "tool_calls"
		}
		chunk(delta(map[string]any{}, finishReason), nil)
		chunk([]any{}, map[string]any{
			"prompt_tokens":     reply.Usage.InputTokens,
			"completion_tokens": reply.Usage.OutputTokens,
			"total_tokens":      reply.Usage.InputTokens + reply.Usage.OutputTokens,
		})
		frames = append(frames, []byte("data: [DONE]\n\n"))
	}
	return llmtest.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Frames: frames}
}

func (openAIWire) Error(status int) llmtest.Response {
	errType, code := "invalid_request_error", "invalid_api_key"
	if status == http.StatusTooManyRequests {
		errType, code = "requests", "rate_limit_exceeded"
	}
	body, _ := json.Marshal(map[string]any{"error": map[string]any{
		"message": "contract test: " + http.StatusText(status), "type": errType, "param": nil, "code": code,
	}})
	return llmtest.Response{
		Status: status,
		Header: http.Header{"Content-Type": {"application/json"}},
		Frames: [][]byte{body},
	}
}

func sseData(v any) []byte {
	payload, _ := json.Marshal(v)
	return append(append([]byte("data: "), payload...), '\n', '\n')
}

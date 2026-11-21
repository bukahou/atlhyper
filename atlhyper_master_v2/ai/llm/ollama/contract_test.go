package ollama_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/llm/llmtest"
	"AtlHyper/atlhyper_master_v2/ai/llm/ollama"
)

func TestContract(t *testing.T) {
	llmtest.RunContract(t, llmtest.Adapter{
		NewClient: func(t *testing.T, upstream llmtest.Endpoint) llm.LLMClient {
			client, err := ollama.NewClient(upstream.BaseURL, "qwen-contract")
			if err != nil {
				t.Fatal(err)
			}
			return client
		},
		Wire:               ollamaWire{},
		Path:               "/api/chat",
		ToolCallIDOnWire:   false, // 原生 /api/chat 的 tool_calls 只有 function（名称 + 参数对象），没有 id
		ToolArgsInOnePiece: "原生 /api/chat 的 tool_calls[].function.arguments 是完整的 JSON 对象，随一行 NDJSON 一次给出，不分段",
	})
}

// ollamaWire 原生 /api/chat 流式响应：NDJSON，每行一个对象；最后一行 done=true，带 done_reason 与用量
type ollamaWire struct{}

func (ollamaWire) Stream(reply llmtest.Reply) llmtest.Response {
	var frames [][]byte
	line := func(message map[string]any, done bool, extra map[string]any) {
		body := map[string]any{
			"model": "qwen-contract", "created_at": "2026-01-01T00:00:00Z", "message": message, "done": done,
		}
		for k, v := range extra {
			body[k] = v
		}
		payload, _ := json.Marshal(body)
		frames = append(frames, append(payload, '\n'))
	}

	for _, text := range reply.Texts {
		line(map[string]any{"role": "assistant", "content": text}, false, nil)
	}
	if call := reply.ToolCall; call != nil {
		line(map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{
			"function": map[string]any{"name": call.Name, "arguments": json.RawMessage(call.Args())},
		}}}, false, nil)
	}
	if reply.Finished {
		line(map[string]any{"role": "assistant", "content": ""}, true, map[string]any{
			"done_reason":       "stop",
			"total_duration":    1000000,
			"prompt_eval_count": reply.Usage.InputTokens,
			"eval_count":        reply.Usage.OutputTokens,
		})
	}
	return llmtest.Response{Header: http.Header{"Content-Type": {"application/x-ndjson"}}, Frames: frames}
}

// Error Ollama 出错时回 {"error": "..."}（401 / 429 多来自前面的鉴权代理或云端托管）
func (ollamaWire) Error(status int) llmtest.Response {
	body, _ := json.Marshal(map[string]any{"error": "contract test: " + http.StatusText(status)})
	return llmtest.Response{
		Status: status,
		Header: http.Header{"Content-Type": {"application/json; charset=utf-8"}},
		Frames: [][]byte{body},
	}
}

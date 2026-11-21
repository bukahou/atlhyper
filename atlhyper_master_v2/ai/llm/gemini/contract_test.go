package gemini_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"google.golang.org/api/option"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/llm/gemini"
	"AtlHyper/atlhyper_master_v2/ai/llm/llmtest"
)

func TestContract(t *testing.T) {
	llmtest.RunContract(t, llmtest.Adapter{
		NewClient: func(t *testing.T, upstream llmtest.Endpoint) llm.LLMClient {
			client, err := gemini.NewGeminiClient("test-key-placeholder", "gemini-contract", option.WithEndpoint(upstream.BaseURL))
			if err != nil {
				t.Fatal(err)
			}
			return client
		},
		Wire:               geminiWire{},
		Path:               "/v1beta/models/gemini-contract:streamGenerateContent",
		ToolCallIDOnWire:   false, // SDK（generative-ai-go）的 FunctionCall 只有名称与参数，没有 id
		ToolArgsInOnePiece: "functionCall.args 是完整的 JSON 对象，随一个 GenerateContentResponse 一次给出，不分段",
	})
}

// geminiWire SDK 走 REST：POST {model}:streamGenerateContent?$alt=json;enum-encoding=int，
// 响应体是逐个写出的 JSON 数组 [resp, resp, …]，枚举按数字。最后一个 resp 的候选带 finishReason 与 usageMetadata。
type geminiWire struct{}

const finishReasonStop = 1 // FinishReason.STOP

func (geminiWire) Stream(reply llmtest.Reply) llmtest.Response {
	var frames [][]byte
	element := func(parts []any, finished bool) {
		candidate := map[string]any{"content": map[string]any{"role": "model", "parts": parts}, "index": 0}
		body := map[string]any{"candidates": []any{candidate}, "modelVersion": "gemini-contract"}
		if finished {
			candidate["finishReason"] = finishReasonStop
			body["usageMetadata"] = map[string]any{
				"promptTokenCount":     reply.Usage.InputTokens,
				"candidatesTokenCount": reply.Usage.OutputTokens,
				"totalTokenCount":      reply.Usage.InputTokens + reply.Usage.OutputTokens,
			}
		}
		payload, _ := json.Marshal(body)
		separator := []byte(",\r\n")
		if len(frames) == 0 {
			separator = []byte("[")
		}
		frames = append(frames, append(separator, payload...))
	}

	for _, text := range reply.Texts {
		element([]any{map[string]any{"text": text}}, false)
	}
	if call := reply.ToolCall; call != nil {
		element([]any{map[string]any{"functionCall": map[string]any{
			"name": call.Name, "args": json.RawMessage(call.Args()),
		}}}, false)
	}
	if reply.Finished {
		element([]any{map[string]any{"text": ""}}, true)
		frames = append(frames, []byte("]"))
	}
	return llmtest.Response{Header: http.Header{"Content-Type": {"application/json; charset=UTF-8"}}, Frames: frames}
}

func (geminiWire) Error(status int) llmtest.Response {
	statusName := "UNAUTHENTICATED"
	if status == http.StatusTooManyRequests {
		statusName = "RESOURCE_EXHAUSTED"
	}
	body, _ := json.Marshal(map[string]any{"error": map[string]any{
		"code": status, "message": "contract test: " + http.StatusText(status), "status": statusName,
	}})
	return llmtest.Response{
		Status: status,
		Header: http.Header{"Content-Type": {"application/json; charset=UTF-8"}},
		Frames: [][]byte{body},
	}
}

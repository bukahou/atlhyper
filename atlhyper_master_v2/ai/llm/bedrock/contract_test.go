package bedrock_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/llm/bedrock"
	"AtlHyper/atlhyper_master_v2/ai/llm/llmtest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

func TestContract(t *testing.T) {
	llmtest.RunContract(t, llmtest.Adapter{
		NewClient: func(t *testing.T, upstream llmtest.Endpoint) llm.LLMClient {
			client, err := bedrock.NewBedrockClient(llm.Config{
				APIKey: "test-key-placeholder", Model: "test.contract-model-v1:0", Region: "us-east-1",
			}, func(o *bedrockruntime.Options) {
				o.BaseEndpoint = aws.String(upstream.BaseURL)
				o.HTTPClient = upstream.HTTPClient
				// 429 由 SDK 自带的重试退避处理，契约只核对最终分类：关掉重试，免得用例等退避
				o.RetryMaxAttempts = 1
			})
			if err != nil {
				t.Fatal(err)
			}
			return client
		},
		Wire:             bedrockWire{},
		TLS:              true, // SDK 只肯经 HTTPS 发 Bearer 令牌
		Path:             "/model/test.contract-model-v1%3A0/converse-stream",
		ToolCallIDOnWire: true,
	})
}

// bedrockWire ConverseStream 响应：AWS event-stream 二进制分帧，每帧一个事件（:event-type 头 + JSON 载荷）。
// messageStart → contentBlockStart / Delta… / Stop → messageStop（停止原因）→ metadata（用量）
type bedrockWire struct{}

func (bedrockWire) Stream(reply llmtest.Reply) llmtest.Response {
	var frames [][]byte
	event := func(eventType string, payload map[string]any) {
		frames = append(frames, eventFrame(eventType, payload))
	}

	event("messageStart", map[string]any{"role": "assistant"})
	index := 0
	if len(reply.Texts) > 0 {
		for _, text := range reply.Texts {
			event("contentBlockDelta", map[string]any{"contentBlockIndex": index, "delta": map[string]any{"text": text}})
		}
		if !reply.Finished && reply.ToolCall == nil {
			return bedrockStream(frames)
		}
		event("contentBlockStop", map[string]any{"contentBlockIndex": index})
		index++
	}
	if call := reply.ToolCall; call != nil {
		event("contentBlockStart", map[string]any{"contentBlockIndex": index, "start": map[string]any{
			"toolUse": map[string]any{"toolUseId": call.ID, "name": call.Name},
		}})
		for _, fragment := range call.ArgFragments {
			event("contentBlockDelta", map[string]any{"contentBlockIndex": index, "delta": map[string]any{
				"toolUse": map[string]any{"input": fragment},
			}})
		}
		event("contentBlockStop", map[string]any{"contentBlockIndex": index})
	}
	if reply.Finished {
		stopReason := "end_turn"
		if reply.ToolCall != nil {
			stopReason = "tool_use"
		}
		event("messageStop", map[string]any{"stopReason": stopReason})
		event("metadata", map[string]any{
			"usage": map[string]any{
				"inputTokens":  reply.Usage.InputTokens,
				"outputTokens": reply.Usage.OutputTokens,
				"totalTokens":  reply.Usage.InputTokens + reply.Usage.OutputTokens,
			},
			"metrics": map[string]any{"latencyMs": 1},
		})
	}
	return bedrockStream(frames)
}

func bedrockStream(frames [][]byte) llmtest.Response {
	return llmtest.Response{Header: http.Header{"Content-Type": {"application/vnd.amazon.eventstream"}}, Frames: frames}
}

// Error REST-JSON 的错误响应：错误类型在 X-Amzn-ErrorType 头里，载荷只有 message。
// 401 不带具体类型（Bedrock 自己的鉴权失败多为 403，401 常见于前置代理），这里核对的是状态码的分类。
func (bedrockWire) Error(status int) llmtest.Response {
	header := http.Header{"Content-Type": {"application/json"}}
	if status == http.StatusTooManyRequests {
		header.Set("X-Amzn-ErrorType", "ThrottlingException")
	}
	body, _ := json.Marshal(map[string]any{"message": "contract test: " + http.StatusText(status)})
	return llmtest.Response{Status: status, Header: header, Frames: [][]byte{body}}
}

// eventFrame 编一帧 event-stream 消息
func eventFrame(eventType string, payload map[string]any) []byte {
	body, _ := json.Marshal(payload)
	var headers eventstream.Headers
	headers.Set(":message-type", eventstream.StringValue("event"))
	headers.Set(":event-type", eventstream.StringValue(eventType))
	headers.Set(":content-type", eventstream.StringValue("application/json"))
	var buf bytes.Buffer
	if err := eventstream.NewEncoder().Encode(&buf, eventstream.Message{Headers: headers, Payload: body}); err != nil {
		return nil
	}
	return buf.Bytes()
}

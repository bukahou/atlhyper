package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go/v2"
)

// fakeMessagesAPI 假的 Messages 接口：记下请求体，按预设返回 SSE 事件或错误。
type fakeMessagesAPI struct {
	mu       sync.Mutex
	bodies   []map[string]any
	headers  []http.Header
	status   int
	errBody  string
	events   []string // 每个元素是一个事件的 data JSON（事件名取其中的 type）
	requests int
}

func (f *fakeMessagesAPI) handler(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	f.mu.Lock()
	f.requests++
	f.bodies = append(f.bodies, body)
	f.headers = append(f.headers, r.Header.Clone())
	f.mu.Unlock()

	if f.status != 0 && f.status != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(f.status)
		_, _ = io.WriteString(w, f.errBody)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	for _, data := range f.events {
		var head struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal([]byte(data), &head)
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", head.Type, data)
	}
}

func newTestClient(t *testing.T, api *fakeMessagesAPI) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(api.handler))
	t.Cleanup(server.Close)
	client, err := NewAnthropicClient(llm.Config{APIKey: "sk-test", Model: "claude-haiku-5-5", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func drain(t *testing.T, ch <-chan *llm.Chunk) []*llm.Chunk {
	t.Helper()
	var chunks []*llm.Chunk
	timeout := time.After(5 * time.Second)
	for {
		select {
		case c, ok := <-ch:
			if !ok {
				return chunks
			}
			chunks = append(chunks, c)
		case <-timeout:
			t.Fatal("流没有结束")
		}
	}
}

func messageStart(input, cacheRead, cacheWrite int) string {
	return fmt.Sprintf(`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-haiku-5-5","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d,"output_tokens":1}}}`, input, cacheRead, cacheWrite)
}

func messageDelta(stopReason string, output int, stopDetails string) string {
	if stopDetails == "" {
		stopDetails = "null"
	}
	return fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":%q,"stop_sequence":null,"stop_details":%s},"usage":{"output_tokens":%d}}`, stopReason, stopDetails, output)
}

const messageStop = `{"type":"message_stop"}`

func TestChatStream_TextToolCallUsageAndProviderTurn(t *testing.T) {
	api := &fakeMessagesAPI{events: []string{
		messageStart(100, 50, 10),
		`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":"","signature":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"sig-abc"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"我查"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"一下"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"content_block_start","index":2,"content_block":{"type":"tool_use","id":"toolu_01","name":"get_pods","input":{}}}`,
		`{"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"{\"namespace\":"}}`,
		`{"type":"content_block_delta","index":2,"delta":{"type":"input_json_delta","partial_json":"\"default\"}"}}`,
		`{"type":"content_block_stop","index":2}`,
		messageDelta("tool_use", 42, ""),
		messageStop,
	}}
	client := newTestClient(t, api)

	ch, err := client.ChatStream(context.Background(), &llm.Request{
		SystemPrompt: "sys",
		Messages:     []llm.Message{{Role: "user", Content: "看看 Pod"}},
		Tools:        []llm.ToolDefinition{{Name: "get_pods"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	chunks := drain(t, ch)

	var text strings.Builder
	var calls []*llm.ToolCall
	var done *llm.Chunk
	for _, c := range chunks {
		switch c.Type {
		case llm.ChunkText:
			text.WriteString(c.Content)
		case llm.ChunkToolCall:
			calls = append(calls, c.ToolCall)
		case llm.ChunkDone:
			done = c
		case llm.ChunkError:
			t.Fatalf("unexpected error chunk: %v", c.Error)
		}
	}
	if text.String() != "我查一下" {
		t.Errorf("text = %q", text.String())
	}
	if len(calls) != 1 || calls[0].ID != "toolu_01" || calls[0].Name != "get_pods" {
		t.Fatalf("tool calls = %+v", calls)
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(calls[0].Params), &params); err != nil || params["namespace"] != "default" {
		t.Errorf("tool params = %q (%v)", calls[0].Params, err)
	}
	if done == nil || done.Usage == nil {
		t.Fatal("缺少 Done 或用量")
	}
	if done.Usage.InputTokens != 160 || done.Usage.OutputTokens != 42 {
		t.Errorf("usage = %+v，输入应为 未命中 + 缓存读 + 缓存写 = 160", done.Usage)
	}
	turn, ok := done.ProviderTurn.(anthropicsdk.MessageParam)
	if !ok || turn.OfAssistant == nil || len(turn.OfAssistant.Content) != 3 ||
		turn.OfAssistant.Content[0].OfThinking == nil || turn.OfAssistant.Content[0].OfThinking.Signature != "sig-abc" {
		t.Errorf("Done 应带上整轮回答（含思考块与签名）: %#v", done.ProviderTurn)
	}

	// SDK v2 的 WithAPIKey 以 Authorization: Bearer 发送 key
	if len(api.headers) != 1 || api.headers[0].Get("Authorization") != "Bearer sk-test" {
		t.Errorf("配了 key 时应带上这把 key: %v", api.headers)
	}
	if api.bodies[0]["stream"] != true {
		t.Errorf("应以流式请求: %v", api.bodies[0]["stream"])
	}
}

func TestChatStream_RefusalIsError(t *testing.T) {
	api := &fakeMessagesAPI{events: []string{
		messageStart(10, 0, 0),
		messageDelta("refusal", 3, `{"type":"refusal","category":"cyber","explanation":"policy"}`),
		messageStop,
	}}
	ch, err := newTestClient(t, api).ChatStream(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "q"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	chunks := drain(t, ch)
	last := chunks[len(chunks)-1]
	if last.Type != llm.ChunkError || last.Error == nil || !strings.Contains(last.Error.Error(), "cyber") {
		t.Errorf("拒答应以错误结束并带上类别: %+v", last)
	}
}

func TestChatStream_TruncatedToolUseIsError(t *testing.T) {
	api := &fakeMessagesAPI{events: []string{
		messageStart(10, 0, 0),
		`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_01","name":"get_pods","input":{}}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"namespace\":\"def"}}`,
		`{"type":"content_block_stop","index":0}`,
		messageDelta("max_tokens", 32000, ""),
		messageStop,
	}}
	ch, err := newTestClient(t, api).ChatStream(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "q"}},
		Tools:    []llm.ToolDefinition{{Name: "get_pods"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	chunks := drain(t, ch)
	for _, c := range chunks {
		if c.Type == llm.ChunkToolCall {
			t.Errorf("截断的工具调用参数不完整，不应执行: %+v", c.ToolCall)
		}
	}
	if last := chunks[len(chunks)-1]; last.Type != llm.ChunkError {
		t.Errorf("输出到上限且含工具调用时应报错: %+v", last)
	}
}

func TestChatStream_APIErrorIsReported(t *testing.T) {
	api := &fakeMessagesAPI{
		status:  http.StatusUnauthorized,
		errBody: `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`,
	}
	ch, err := newTestClient(t, api).ChatStream(context.Background(), &llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "q"}},
	})
	if err != nil {
		if !strings.Contains(err.Error(), "401") {
			t.Errorf("err = %v", err)
		}
		return
	}
	chunks := drain(t, ch)
	if last := chunks[len(chunks)-1]; last.Type != llm.ChunkError || !strings.Contains(last.Error.Error(), "401") {
		t.Errorf("接口错误应以错误结束并带状态码: %+v", last)
	}
}

// SDK 的 HTTP 错误带出状态码、拒答按类别：都能被通用层分类
func TestAPIError_Classified(t *testing.T) {
	if got := llm.Classify(fmt.Errorf("请求 Claude: %w", apiError(&anthropicsdk.Error{StatusCode: 429}))); got != llm.ErrKindRateLimited {
		t.Errorf("429 → %q", got)
	}
	if got := llm.Classify(&RefusalError{Category: "x"}); got != llm.ErrKindRefused {
		t.Errorf("拒答 → %q", got)
	}
}

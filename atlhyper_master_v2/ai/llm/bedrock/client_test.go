package bedrock

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// 假 Bedrock：校验路径与鉴权头，以真实的 event-stream 线格式回一段「文本 + 一次工具调用 + usage」
type fakeBedrock struct {
	gotAuth string
	gotPath string
}

func (f *fakeBedrock) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f.gotAuth = r.Header.Get("Authorization")
		f.gotPath = r.URL.EscapedPath()

		var buf bytes.Buffer
		enc := eventstream.NewEncoder()
		send := func(eventType, payload string) {
			var h eventstream.Headers
			h.Set(":message-type", eventstream.StringValue("event"))
			h.Set(":event-type", eventstream.StringValue(eventType))
			h.Set(":content-type", eventstream.StringValue("application/json"))
			if err := enc.Encode(&buf, eventstream.Message{Headers: h, Payload: []byte(payload)}); err != nil {
				t.Fatal(err)
			}
		}
		send("messageStart", `{"role":"assistant"}`)
		send("contentBlockDelta", `{"contentBlockIndex":0,"delta":{"text":"checking"}}`)
		send("contentBlockStop", `{"contentBlockIndex":0}`)
		send("contentBlockStart", `{"contentBlockIndex":1,"start":{"toolUse":{"toolUseId":"tu-1","name":"get_pods"}}}`)
		send("contentBlockDelta", `{"contentBlockIndex":1,"delta":{"toolUse":{"input":"{\"ns\":\"default\"}"}}}`)
		send("contentBlockStop", `{"contentBlockIndex":1}`)
		send("messageStop", `{"stopReason":"tool_use"}`)
		send("metadata", `{"usage":{"inputTokens":20,"outputTokens":7,"totalTokens":27},"metrics":{"latencyMs":5}}`)

		w.Header().Set("Content-Type", "application/vnd.amazon.eventstream")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

func newTestClient(t *testing.T, srv *httptest.Server, cfg llm.Config) *Client {
	t.Helper()
	c, err := NewBedrockClient(cfg, func(o *bedrockruntime.Options) {
		o.BaseEndpoint = aws.String(srv.URL)
		o.HTTPClient = srv.Client()
		o.RetryMaxAttempts = 1
	})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestChatStream_EndToEndAgainstFakeBedrock(t *testing.T) {
	// ⚠️ SDK 的 New() 会读这个环境变量并覆盖 Options 里的 token —— 我方 key 必须仍然生效
	t.Setenv("AWS_BEARER_TOKEN_BEDROCK", "env-key-must-not-be-used")

	fake := &fakeBedrock{}
	srv := httptest.NewTLSServer(fake.handler(t))
	defer srv.Close()

	c := newTestClient(t, srv, llm.Config{Provider: "bedrock", APIKey: "web-key", Model: "qwen.qwen3-235b-a22b-2507-v1:0", Region: "ap-northeast-1"})
	ch, err := c.ChatStream(context.Background(), &llm.Request{Messages: []llm.Message{{Role: "user", Content: "why"}}})
	if err != nil {
		t.Fatal(err)
	}

	var text string
	var calls []*llm.ToolCall
	var done *llm.Chunk
	for chunk := range ch {
		switch chunk.Type {
		case llm.ChunkText:
			text += chunk.Content
		case llm.ChunkToolCall:
			calls = append(calls, chunk.ToolCall)
		case llm.ChunkDone:
			done = chunk
		case llm.ChunkError:
			t.Fatalf("stream error: %v", chunk.Error)
		}
	}

	if fake.gotAuth != "Bearer web-key" {
		t.Errorf("Authorization = %q（应为 Web 填写的 key，不得被环境变量覆盖）", fake.gotAuth)
	}
	if want := "/model/qwen.qwen3-235b-a22b-2507-v1%3A0/converse-stream"; fake.gotPath != want {
		t.Errorf("path = %q, want %q", fake.gotPath, want)
	}
	if text != "checking" {
		t.Errorf("text = %q", text)
	}
	if len(calls) != 1 || calls[0].ID != "tu-1" || calls[0].Name != "get_pods" || calls[0].Params != `{"ns":"default"}` {
		t.Errorf("calls = %+v", calls)
	}
	if done == nil || done.Usage == nil || done.Usage.InputTokens != 20 || done.Usage.OutputTokens != 7 {
		t.Errorf("done = %+v", done)
	}
}

func TestNewBedrockClient_Validation(t *testing.T) {
	cases := []struct {
		name string
		cfg  llm.Config
		want string
	}{
		{"缺 key", llm.Config{Model: "m", Region: "ap-northeast-1"}, "api key"},
		{"缺模型", llm.Config{APIKey: "k", Region: "ap-northeast-1"}, "model"},
		{"缺 region", llm.Config{APIKey: "k", Model: "m"}, "region"},
	}
	for _, c := range cases {
		_, err := NewBedrockClient(c.cfg)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v", c.name, err)
		}
	}
}

func TestRegisteredAsBedrock(t *testing.T) {
	c, err := llm.NewLLMClient(llm.Config{Provider: "bedrock", APIKey: "k", Model: "m", Region: "ap-northeast-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.(*Client); !ok {
		t.Errorf("got %T", c)
	}
}

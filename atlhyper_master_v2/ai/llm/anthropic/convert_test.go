package anthropic

import (
	"encoding/json"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	anthropicsdk "github.com/anthropics/anthropic-sdk-go/v2"
)

// wire 把请求参数序列化后解析成通用结构，按线上 JSON 断言（不依赖 SDK 联合类型的内部字段）。
func wire(t *testing.T, params anthropicsdk.MessageNewParams) map[string]any {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	return out
}

func messagesOf(t *testing.T, params anthropicsdk.MessageNewParams) []map[string]any {
	t.Helper()
	list, _ := wire(t, params)["messages"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		out = append(out, m.(map[string]any))
	}
	return out
}

func blocksOf(msg map[string]any) []map[string]any {
	list, _ := msg["content"].([]any)
	out := make([]map[string]any, 0, len(list))
	for _, b := range list {
		out = append(out, b.(map[string]any))
	}
	return out
}

func mustBuild(t *testing.T, req *llm.Request) anthropicsdk.MessageNewParams {
	t.Helper()
	params, err := buildParams("claude-haiku-5-5", req)
	if err != nil {
		t.Fatalf("buildParams: %v", err)
	}
	return params
}

func TestBuildParams_RequestShape(t *testing.T) {
	params := mustBuild(t, &llm.Request{
		SystemPrompt: "你是 SRE 助手",
		Messages:     []llm.Message{{Role: "user", Content: "看看 default 的 Pod"}},
		Tools: []llm.ToolDefinition{
			{Name: "get_pods", Description: "列出 Pod", Parameters: json.RawMessage(`{"type":"object","properties":{"namespace":{"type":"string"}},"required":["namespace"]}`)},
			{Name: "now"},
		},
	})
	got := wire(t, params)

	if got["model"] != "claude-haiku-5-5" {
		t.Errorf("model = %v", got["model"])
	}
	if got["max_tokens"] != float64(maxOutputTokens) {
		t.Errorf("max_tokens = %v, want %d（思考也计入，4096 不够）", got["max_tokens"], maxOutputTokens)
	}
	system, _ := got["system"].([]any)
	if len(system) != 1 || system[0].(map[string]any)["text"] != "你是 SRE 助手" ||
		system[0].(map[string]any)["cache_control"] == nil {
		t.Errorf("system 应为一段带缓存断点的文字: %v", got["system"])
	}
	if got["cache_control"] == nil {
		t.Error("应开启请求级自动缓存（工具循环里每轮复用前缀）")
	}
	for _, key := range []string{"temperature", "top_p", "top_k", "thinking"} {
		if _, ok := got[key]; ok {
			t.Errorf("不应发送 %s（新型号拒收采样参数；思考按模型默认）", key)
		}
	}

	tools, _ := got["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("tools = %v", got["tools"])
	}
	first := tools[0].(map[string]any)
	schema := first["input_schema"].(map[string]any)
	if first["description"] != "列出 Pod" || schema["properties"].(map[string]any)["namespace"] == nil {
		t.Errorf("工具定义转换不对: %v", first)
	}
	if req, _ := schema["required"].([]any); len(req) != 1 || req[0] != "namespace" {
		t.Errorf("required = %v", schema["required"])
	}
	noParams := tools[1].(map[string]any)["input_schema"].(map[string]any)
	if props, ok := noParams["properties"].(map[string]any); !ok || len(props) != 0 {
		t.Errorf("无参数工具也要带空 properties（否则 SDK 省掉 input_schema，Claude 回 400）: %v", noParams)
	}
}

func TestBuildParams_RejectsInvalidToolSchema(t *testing.T) {
	_, err := buildParams("claude-haiku-5-5", &llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "hi"}},
		Tools:    []llm.ToolDefinition{{Name: "broken", Parameters: json.RawMessage(`{not json`)}},
	})
	if err == nil {
		t.Fatal("工具参数定义不是合法 JSON 时应报错")
	}
}

func TestConvertMessages_GroupsToolResultsIntoOneUserMessage(t *testing.T) {
	msgs := messagesOf(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "查一下"},
		{Role: "assistant", Content: "好的", ToolCalls: []llm.ToolCall{
			{ID: "toolu_a", Name: "get_pods", Params: `{"namespace":"default"}`},
			{ID: "toolu_b", Name: "get_events", Params: `{}`},
		}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "toolu_a", Name: "get_pods", Content: "3 pods"}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "toolu_b", Name: "get_events", Content: "no events"}},
	}}))

	if len(msgs) != 3 {
		t.Fatalf("应为 user / assistant / user 三条，实际 %d: %v", len(msgs), msgs)
	}
	assistant := blocksOf(msgs[1])
	if msgs[1]["role"] != "assistant" || len(assistant) != 3 ||
		assistant[1]["type"] != "tool_use" || assistant[1]["id"] != "toolu_a" ||
		assistant[1]["input"].(map[string]any)["namespace"] != "default" {
		t.Errorf("assistant = %v", msgs[1])
	}
	results := blocksOf(msgs[2])
	if msgs[2]["role"] != "user" || len(results) != 2 ||
		results[0]["type"] != "tool_result" || results[0]["tool_use_id"] != "toolu_a" ||
		results[1]["tool_use_id"] != "toolu_b" {
		t.Errorf("同一轮的工具结果应并进一条用户消息: %v", msgs[2])
	}
}

// 历史按条数截断后，开头可能是孤立的工具结果或 assistant 消息，中间也可能有没等到结果的调用：Claude 都会直接 400。
func TestConvertMessages_RepairsTruncatedHistory(t *testing.T) {
	msgs := messagesOf(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "assistant", Content: "上一段回答的结尾"},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "toolu_gone", Name: "get_pods", Content: "3 pods"}},
		{Role: "user", Content: "继续"},
		{Role: "assistant", Content: "我先查", ToolCalls: []llm.ToolCall{{ID: "toolu_x", Name: "get_events", Params: `{}`}}},
		{Role: "user", Content: "算了，直接说结论"},
	}}))

	if len(msgs) != 3 || msgs[0]["role"] != "user" {
		t.Fatalf("应为 user / assistant / user 三条且以 user 开头，实际: %v", msgs)
	}
	first := blocksOf(msgs[0])
	if len(first) != 2 || first[0]["type"] != "text" || first[1]["text"] != "继续" {
		t.Errorf("孤立的工具结果应改成文字并与后面的用户消息合并: %v", first)
	}
	for _, b := range blocksOf(msgs[1]) {
		if b["type"] == "tool_use" {
			t.Errorf("没有结果的工具调用应去掉: %v", msgs[1])
		}
	}
}

func TestConvertMessages_RewritesInvalidToolCallIDs(t *testing.T) {
	msgs := messagesOf(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "call:1.2", Name: "get_pods", Params: `{}`}}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "call:1.2", Name: "get_pods", Content: "ok"}},
	}}))

	use := blocksOf(msgs[1])[0]
	result := blocksOf(msgs[2])[0]
	id, _ := use["id"].(string)
	if id == "call:1.2" || id == "" || result["tool_use_id"] != id {
		t.Errorf("不合法的 ID 应重写，且调用与结果一致: use=%v result=%v", use["id"], result["tool_use_id"])
	}
}

// 本次请求内的工具循环：上一轮的完整回答（含思考块与签名）必须原样发回。
func TestConvertMessages_ReplaysProviderTurnVerbatim(t *testing.T) {
	turn := anthropicsdk.NewAssistantMessage(
		anthropicsdk.NewThinkingBlock("sig-abc", ""),
		anthropicsdk.NewTextBlock("我查一下"),
		anthropicsdk.NewToolUseBlock("toolu_01", map[string]any{"namespace": "default"}, "get_pods"),
	)
	msgs := messagesOf(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", Content: "我查一下", ProviderTurn: turn,
			ToolCalls: []llm.ToolCall{{ID: "toolu_01", Name: "get_pods", Params: `{"namespace":"default"}`}}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "toolu_01", Name: "get_pods", Content: "3 pods"}},
	}}))

	assistant := blocksOf(msgs[1])
	if len(assistant) != 3 || assistant[0]["type"] != "thinking" || assistant[0]["signature"] != "sig-abc" ||
		assistant[2]["id"] != "toolu_01" {
		t.Errorf("应原样回放上一轮（思考块在最前、带签名）: %v", assistant)
	}
	if r := blocksOf(msgs[2]); len(r) != 1 || r[0]["tool_use_id"] != "toolu_01" {
		t.Errorf("tool_result = %v", msgs[2])
	}
}

// 工具循环会截掉一轮里过多的调用：原样回放的那一轮仍含被截掉的 tool_use，必须补上「未执行」的结果，否则 400。
func TestConvertMessages_ReplayedTurnGetsResultsForTruncatedCalls(t *testing.T) {
	turn := anthropicsdk.NewAssistantMessage(
		anthropicsdk.NewThinkingBlock("sig", ""),
		anthropicsdk.NewToolUseBlock("toolu_1", map[string]any{}, "get_pods"),
		anthropicsdk.NewToolUseBlock("toolu_2", map[string]any{}, "get_events"),
	)
	msgs := messagesOf(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", ProviderTurn: turn, ToolCalls: []llm.ToolCall{{ID: "toolu_1", Name: "get_pods", Params: `{}`}}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "toolu_1", Name: "get_pods", Content: "ok"}},
	}}))

	if len(blocksOf(msgs[1])) != 3 {
		t.Errorf("回放的那一轮不能删改: %v", msgs[1])
	}
	results := blocksOf(msgs[2])
	if len(results) != 2 || results[0]["tool_use_id"] != "toolu_1" || results[1]["tool_use_id"] != "toolu_2" {
		t.Fatalf("每个 tool_use 都要有结果: %v", results)
	}
	if results[1]["is_error"] != true {
		t.Errorf("补上的结果应标为未执行（is_error）: %v", results[1])
	}
}

// 别的 provider 留下的 ProviderTurn（类型对不上）按普通消息转换。
func TestConvertMessages_IgnoresForeignProviderTurn(t *testing.T) {
	msgs := messagesOf(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", Content: "answer", ProviderTurn: "from-another-provider"},
		{Role: "user", Content: "q2"},
	}}))
	if b := blocksOf(msgs[1]); len(b) != 1 || b[0]["text"] != "answer" {
		t.Errorf("assistant = %v", msgs[1])
	}
}

// 收尾轮不给工具（要求直接作答），但历史里有工具往来：Claude 要求带上这些工具的定义，并声明这一轮不许调用
func TestBuildParams_NoToolsButHistoryHasToolUse(t *testing.T) {
	got := wire(t, mustBuild(t, &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "q"},
		{Role: "assistant", ToolCalls: []llm.ToolCall{{ID: "toolu_1", Name: "get_pods", Params: `{}`}}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "toolu_1", Name: "get_pods", Content: "ok"}},
	}}))
	tools, _ := got["tools"].([]any)
	if len(tools) != 1 || tools[0].(map[string]any)["name"] != "get_pods" {
		t.Errorf("应按历史补上工具定义: %v", got["tools"])
	}
	if choice, _ := got["tool_choice"].(map[string]any); choice["type"] != "none" {
		t.Errorf("tool_choice 应为 none: %v", got["tool_choice"])
	}
}

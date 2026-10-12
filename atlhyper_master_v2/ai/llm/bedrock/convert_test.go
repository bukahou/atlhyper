package bedrock

import (
	"encoding/json"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

func textOf(t *testing.T, b types.ContentBlock) string {
	t.Helper()
	v, ok := b.(*types.ContentBlockMemberText)
	if !ok {
		t.Fatalf("want text block, got %T", b)
	}
	return v.Value
}

func docJSON(t *testing.T, d interface{ MarshalSmithyDocument() ([]byte, error) }) string {
	t.Helper()
	b, err := d.MarshalSmithyDocument()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildInput_SystemModelAndTools(t *testing.T) {
	in, err := buildInput("qwen.test-v1:0", &llm.Request{
		SystemPrompt: "you are sre",
		Messages:     []llm.Message{{Role: "user", Content: "hi"}},
		Tools: []llm.ToolDefinition{
			{Name: "get_pods", Description: "list pods", Parameters: json.RawMessage(`{"type":"object","properties":{"ns":{"type":"string"}}}`)},
			{Name: "ping", Description: "no params"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if *in.ModelId != "qwen.test-v1:0" {
		t.Errorf("model = %s", *in.ModelId)
	}
	if len(in.System) != 1 || in.System[0].(*types.SystemContentBlockMemberText).Value != "you are sre" {
		t.Errorf("system = %+v", in.System)
	}
	if in.ToolConfig == nil || len(in.ToolConfig.Tools) != 2 {
		t.Fatalf("tools = %+v", in.ToolConfig)
	}
	spec := in.ToolConfig.Tools[0].(*types.ToolMemberToolSpec).Value
	if *spec.Name != "get_pods" || *spec.Description != "list pods" {
		t.Errorf("spec = %+v", spec)
	}
	schema := docJSON(t, spec.InputSchema.(*types.ToolInputSchemaMemberJson).Value)
	if schema != `{"properties":{"ns":{"type":"string"}},"type":"object"}` {
		t.Errorf("schema = %s", schema)
	}
	// 没给参数 schema 的工具补一个空 object（Bedrock 要求 inputSchema 必填）
	empty := in.ToolConfig.Tools[1].(*types.ToolMemberToolSpec).Value
	if got := docJSON(t, empty.InputSchema.(*types.ToolInputSchemaMemberJson).Value); got != `{"properties":{},"type":"object"}` {
		t.Errorf("empty schema = %s", got)
	}
}

func TestBuildInput_NoToolsNoToolConfig(t *testing.T) {
	in, err := buildInput("m", &llm.Request{Messages: []llm.Message{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	if in.ToolConfig != nil || len(in.System) != 0 {
		t.Errorf("无工具 / 无系统提示时不应带对应字段：%+v %+v", in.ToolConfig, in.System)
	}
}

// 一轮多工具：assistant 的多个 toolUse 之后，所有 toolResult 必须合进同一条 user 消息（Bedrock 要求角色交替）
func TestBuildInput_ToolRoundTrip(t *testing.T) {
	in, err := buildInput("m", &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "why restarts?"},
		{Role: "assistant", Content: "checking", ToolCalls: []llm.ToolCall{
			{ID: "t1", Name: "get_pods", Params: `{"ns":"default"}`},
			{ID: "t2", Name: "get_events", Params: ``},
		}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "t1", Name: "get_pods", Content: "pod-a CrashLoop"}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "t2", Name: "get_events", Content: "OOMKilled"}},
		{Role: "assistant", Content: "it is OOM"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Messages) != 4 {
		t.Fatalf("messages = %d, want 4（两条 tool 合并为一条 user）", len(in.Messages))
	}

	asst := in.Messages[1]
	if asst.Role != types.ConversationRoleAssistant || len(asst.Content) != 3 {
		t.Fatalf("assistant = %+v", asst)
	}
	if textOf(t, asst.Content[0]) != "checking" {
		t.Error("assistant 文本应在 toolUse 之前")
	}
	use1 := asst.Content[1].(*types.ContentBlockMemberToolUse).Value
	if *use1.ToolUseId != "t1" || *use1.Name != "get_pods" || docJSON(t, use1.Input) != `{"ns":"default"}` {
		t.Errorf("toolUse1 = %+v / %s", use1, docJSON(t, use1.Input))
	}
	if got := docJSON(t, asst.Content[2].(*types.ContentBlockMemberToolUse).Value.Input); got != `{}` {
		t.Errorf("空参数应为 {}，得到 %s", got)
	}

	results := in.Messages[2]
	if results.Role != types.ConversationRoleUser || len(results.Content) != 2 {
		t.Fatalf("tool results = %+v", results)
	}
	r2 := results.Content[1].(*types.ContentBlockMemberToolResult).Value
	if *r2.ToolUseId != "t2" || r2.Content[0].(*types.ToolResultContentBlockMemberText).Value != "OOMKilled" {
		t.Errorf("toolResult2 = %+v", r2)
	}
}

// 历史被裁剪后可能以 assistant / tool 开头；Bedrock 要求首条为 user —— 丢到第一条 user 为止
func TestBuildInput_DropsLeadingNonUser(t *testing.T) {
	in, err := buildInput("m", &llm.Request{Messages: []llm.Message{
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "x", Content: "stale"}},
		{Role: "assistant", Content: "old"},
		{Role: "user", Content: "new question"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(in.Messages) != 1 || textOf(t, in.Messages[0].Content[0]) != "new question" {
		t.Errorf("messages = %+v", in.Messages)
	}
}

// 空内容块会被 Bedrock 拒绝：无文本也无 tool call 的 assistant 消息直接跳过
func TestBuildInput_SkipsEmptyAssistant(t *testing.T) {
	in, err := buildInput("m", &llm.Request{Messages: []llm.Message{
		{Role: "user", Content: "a"},
		{Role: "assistant"},
		{Role: "user", Content: "b"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// 跳过空 assistant 后两条 user 相邻 → 合并为一条，内容各自成块
	if len(in.Messages) != 1 || len(in.Messages[0].Content) != 2 {
		t.Errorf("messages = %+v", in.Messages)
	}
}

func TestBuildInput_InvalidJSONIsError(t *testing.T) {
	_, err := buildInput("m", &llm.Request{
		Messages: []llm.Message{{Role: "user", Content: "a"}},
		Tools:    []llm.ToolDefinition{{Name: "x", Parameters: json.RawMessage(`{bad`)}},
	})
	if err == nil {
		t.Error("非法 schema 应报错")
	}
}

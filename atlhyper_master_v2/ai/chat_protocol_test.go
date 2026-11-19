package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

// 两轮：第 1 轮并行调两次同名工具（一次成功一次失败），第 2 轮给出结论
func runTwoRoundChat(t *testing.T) ([]*ChatChunk, *scriptedLLM, *memMsgRepo) {
	t.Helper()
	model := &scriptedLLM{rounds: [][]*llm.Chunk{
		{text("先查一下"), call("c1", "get_cluster_risk"), call("c2", "get_cluster_risk"), done(10, 5)},
		{text("结论：正常"), done(20, 5)},
	}}
	exec := newToolExecutor(nil, time.Second)
	calls := 0
	exec.RegisterTool("get_cluster_risk", func(context.Context, string, map[string]interface{}) (string, error) {
		calls++
		if calls == 2 {
			return "", errors.New("集群不在线")
		}
		return `{"risk":10}`, nil
	})
	msgs := &memMsgRepo{}
	s := &aiServiceImpl{
		providerRepo: &usageProviderRepo{},
		msgRepo:      msgs,
		convRepo:     memConvRepo{},
		executor:     exec,
		newClient:    func(llm.Config) (llm.LLMClient, error) { return model, nil },
	}
	out := make(chan *ChatChunk, 64)
	s.chatLoop(context.Background(), "c1", 7, func(string) bool { return true },
		[]llm.Message{{Role: "user", Content: "集群怎么样"}}, out)
	close(out)
	var chunks []*ChatChunk
	for c := range out {
		chunks = append(chunks, c)
	}
	return chunks, model, msgs
}

// 流里的分段带轮次与调用 ID，失败的工具结果带 isError：前端按它们配对与分轮，⛔ 不按名字或文字顺序猜
func TestChatLoop_StructuredProtocol(t *testing.T) {
	chunks, _, _ := runTwoRoundChat(t)

	results := map[string]*ChatChunk{}
	for _, c := range chunks {
		switch c.Type {
		case "tool_call":
			if c.Round != 1 || c.CallID == "" {
				t.Errorf("tool_call 应带 round=1 与 callId: %+v", c)
			}
		case "tool_result":
			results[c.CallID] = c
		case "text":
			if c.Round == 0 {
				t.Errorf("text 分段应带 round: %+v", c)
			}
		}
	}
	if r := results["c1"]; r == nil || r.IsError || r.Round != 1 {
		t.Errorf("c1 的结果 = %+v, want 成功、round 1", r)
	}
	if r := results["c2"]; r == nil || !r.IsError {
		t.Errorf("c2 的结果 = %+v, want isError（执行失败）", r)
	}
	last := chunks[len(chunks)-1]
	if last.Type != "done" || last.Stats == nil || last.Stats.Rounds != 1 || last.Stats.TotalToolCalls != 2 {
		t.Errorf("最后一段 = %+v", last)
	}
}

// 系统提示词在各轮之间逐字不变（缓存断点打在它上面）；轮次预算附在本轮最后一条工具结果上，且不落库
func TestChatLoop_SystemPromptStableAcrossRounds(t *testing.T) {
	_, model, msgs := runTwoRoundChat(t)

	if len(model.requests) != 2 {
		t.Fatalf("调了 %d 次模型, want 2", len(model.requests))
	}
	if model.requests[0].SystemPrompt != model.requests[1].SystemPrompt {
		t.Error("两轮的系统提示词不一样")
	}
	second := model.requests[1].Messages
	lastResult := second[len(second)-1].ToolResult
	if lastResult == nil || !strings.Contains(lastResult.Content, "还有") {
		t.Errorf("本轮最后一条工具结果应附上轮次预算: %+v", lastResult)
	}
	for _, m := range msgs.saved {
		if m.Role == "tool" && strings.Contains(m.Content, "[系统提示]") {
			t.Errorf("落库的工具结果不应带轮次预算: %s", m.Content)
		}
	}
}

// 最终回答只存最后一轮的正文：前几轮的正文已随各自的工具调用落库，再存一遍历史里就重复了
func TestChatLoop_FinalAnswerIsLastRoundOnly(t *testing.T) {
	_, _, msgs := runTwoRoundChat(t)

	var assistants []string
	for _, m := range msgs.saved {
		if m.Role == "assistant" {
			assistants = append(assistants, m.Content)
		}
	}
	if len(assistants) != 2 || assistants[0] != "先查一下" || assistants[1] != "结论：正常" {
		t.Errorf("assistant 消息 = %q, want [先查一下 结论：正常]", assistants)
	}
}

// 按轮裁剪：整轮丢弃，⛔ 不把工具调用与它的结果拆开；最新一轮总是保留
func TestFitMessages_DropsWholeTurns(t *testing.T) {
	big := strings.Repeat("字", 400)
	msgs := []llm.Message{
		{Role: "user", Content: big},
		{Role: "assistant", Content: "查", ToolCalls: []llm.ToolCall{{ID: "a", Name: "t"}}},
		{Role: "tool", ToolResult: &llm.ToolResult{CallID: "a", Content: big}},
		{Role: "assistant", Content: big},
		{Role: "user", Content: "第二问"},
		{Role: "assistant", Content: "答"},
	}
	cm := &ContextManager{contextWindow: 400, outputReserve: 100}
	fitted, truncated := cm.FitMessages("", msgs)
	if !truncated || len(fitted) != 2 || fitted[0].Content != "第二问" {
		t.Errorf("应整轮丢掉第一问，实际 %d 条: %+v", len(fitted), fitted)
	}

	// 只有一轮且超了：整轮保留，交给提供商报错，也不拆
	one := msgs[:3]
	if fitted, _ := cm.FitMessages("", one); len(fitted) != 3 {
		t.Errorf("唯一的一轮应整轮保留，实际 %d 条", len(fitted))
	}
}

func TestRecentTurnsStart(t *testing.T) {
	roles := []string{"user", "assistant", "tool", "assistant", "user", "assistant", "tool", "tool", "assistant"}
	role := func(i int) string { return roles[i] }
	if got := recentTurnsStart(len(roles), 5, role); got != 4 {
		t.Errorf("最多 5 条 → 从第 4 条（第二问）开始，实际 %d", got)
	}
	if got := recentTurnsStart(len(roles), 3, role); got != 4 {
		t.Errorf("最新一轮比上限长时整轮保留 → 4，实际 %d", got)
	}
	if got := recentTurnsStart(len(roles), 20, role); got != 0 {
		t.Errorf("不超上限 → 0，实际 %d", got)
	}
}

// 历史接口下发结构化的工具调用与结果；旧行的 JSON 键是 Go 字段名（CallID / Name），也要读得出来
func TestToMessage_StructuredToolFields(t *testing.T) {
	oldCall := toMessage(&database.AIMessage{Role: "assistant", Content: "查", ToolCalls: `[{"ID":"a","Name":"query_cluster","Params":"{}"}]`})
	if len(oldCall.ToolCalls) != 1 || oldCall.ToolCalls[0].ID != "a" || oldCall.ToolCalls[0].Name != "query_cluster" {
		t.Errorf("旧格式的工具调用 = %+v", oldCall.ToolCalls)
	}
	oldResult := toMessage(&database.AIMessage{Role: "tool", Content: `{"CallID":"a","Name":"query_cluster","Content":"ok"}`})
	if oldResult.ToolResult == nil || oldResult.ToolResult.CallID != "a" || oldResult.Content != "" {
		t.Errorf("旧格式的工具结果 = %+v / content %q", oldResult.ToolResult, oldResult.Content)
	}
	newResult := toMessage(&database.AIMessage{Role: "tool", Content: `{"callId":"b","name":"x","content":"boom","isError":true}`})
	if newResult.ToolResult == nil || !newResult.ToolResult.IsError || newResult.ToolResult.CallID != "b" {
		t.Errorf("新格式的工具结果 = %+v", newResult.ToolResult)
	}
}

package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// scriptedLLM 按轮次返回预设响应，并记下每轮收到的请求
type scriptedLLM struct {
	rounds   [][]*llm.Chunk
	requests []*llm.Request
}

func (s *scriptedLLM) ChatStream(_ context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	cp := *req
	cp.Messages = append([]llm.Message(nil), req.Messages...)
	s.requests = append(s.requests, &cp)
	i := len(s.requests) - 1
	if i >= len(s.rounds) {
		return nil, errors.New("unexpected extra round")
	}
	ch := make(chan *llm.Chunk, len(s.rounds[i]))
	for _, c := range s.rounds[i] {
		ch <- c
	}
	close(ch)
	return ch, nil
}

func (s *scriptedLLM) Close() error { return nil }

func text(s string) *llm.Chunk { return &llm.Chunk{Type: llm.ChunkText, Content: s} }
func call(id, name string) *llm.Chunk {
	return &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: &llm.ToolCall{ID: id, Name: name, Params: "{}"}}
}
func done(in, out int) *llm.Chunk {
	return &llm.Chunk{Type: llm.ChunkDone, Usage: &llm.Usage{InputTokens: in, OutputTokens: out}}
}

func runLoop(t *testing.T, client llm.LLMClient, maxRounds int) (*loopOutcome, error) {
	t.Helper()
	exec := func(_ context.Context, tc *llm.ToolCall) (string, error) { return "result of " + tc.Name, nil }
	return runToolLoop(context.Background(), client, exec, loopParams{
		SystemPrompt: "sys", UserPrompt: "analyze incident", MaxRounds: maxRounds,
		Tools: []llm.ToolDefinition{{Name: "get_events"}}, MaxToolResult: 1000,
	})
}

// ③ 第 1 轮零 tool call → 追加取证提示重试一次；重试轮正常取证后收尾
func TestRunToolLoop_FirstRoundNoToolCallRetriesWithEvidenceHint(t *testing.T) {
	llmc := &scriptedLLM{rounds: [][]*llm.Chunk{
		{text(`{"summary":"guess"}`), done(10, 5)},          // 第 1 轮：直接给结论
		{text("let me check"), call("c1", "get_events"), done(20, 5)}, // 重试：去取证
		{text(`{"summary":"OOM"}`), done(30, 8)},           // 基于证据收尾
	}}
	out, err := runLoop(t, llmc, 8)
	if err != nil {
		t.Fatal(err)
	}
	if out.Response != `{"summary":"OOM"}` || out.ToolCalls != 1 {
		t.Errorf("out = %+v", out)
	}
	if out.InputTokens != 60 || out.OutputTokens != 18 {
		t.Errorf("tokens 应累计全部轮次：%d / %d", out.InputTokens, out.OutputTokens)
	}
	retry := llmc.requests[1].Messages
	last := retry[len(retry)-1]
	if last.Role != "user" || !strings.Contains(last.Content, evidenceReminderMarker) {
		t.Errorf("重试轮应追加取证提示，末条消息 = %+v", last)
	}
}

// 重试后仍零 tool call → 接受该结论（不硬性禁止收尾），且只重试一次
func TestRunToolLoop_RetriesAtMostOnce(t *testing.T) {
	llmc := &scriptedLLM{rounds: [][]*llm.Chunk{
		{text("first"), done(1, 1)},
		{text(`{"summary":"no evidence needed: cluster empty"}`), done(1, 1)},
	}}
	out, err := runLoop(t, llmc, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(llmc.requests) != 2 || out.Response != `{"summary":"no evidence needed: cluster empty"}` {
		t.Errorf("rounds=%d out=%+v", len(llmc.requests), out)
	}
}

// 已经取过证之后的零 tool call 是正常收尾，不触发提示
func TestRunToolLoop_NoReminderAfterEvidence(t *testing.T) {
	llmc := &scriptedLLM{rounds: [][]*llm.Chunk{
		{call("c1", "get_events"), done(1, 1)},
		{text("final"), done(1, 1)},
	}}
	out, err := runLoop(t, llmc, 8)
	if err != nil || out.Response != "final" || len(llmc.requests) != 2 {
		t.Errorf("out=%+v err=%v rounds=%d", out, err, len(llmc.requests))
	}
}

// 只剩一轮时（maxRounds=1）没有重试余地，直接收尾
func TestRunToolLoop_NoRetryWhenNoRoundsLeft(t *testing.T) {
	llmc := &scriptedLLM{rounds: [][]*llm.Chunk{{text("only"), done(1, 1)}}}
	out, err := runLoop(t, llmc, 1)
	if err != nil || out.Response != "only" {
		t.Errorf("out=%+v err=%v", out, err)
	}
}

// 流出错 → 返回错误，而不是把半截文本当成「正常结束」的报告
func TestRunToolLoop_StreamErrorIsNotTreatedAsDone(t *testing.T) {
	llmc := &scriptedLLM{rounds: [][]*llm.Chunk{
		{text(`{"summ`), {Type: llm.ChunkError, Error: errors.New("throttled")}},
	}}
	if _, err := runLoop(t, llmc, 8); err == nil || !strings.Contains(err.Error(), "throttled") {
		t.Errorf("err = %v", err)
	}
}

package llmtest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// 阳性对照：判定对违反契约的分段序列必须报出问题，对符合的必须不报

func TestCompletedProblems(t *testing.T) {
	want := Reply{
		Texts:    []string{"甲", "乙"},
		ToolCall: &ToolCallReply{ID: "call_1", Name: "get_pods", ArgFragments: []string{`{"n":`, `1}`}},
		Finished: true,
		Usage:    llm.Usage{InputTokens: 3, OutputTokens: 2},
	}
	valid := func() []*llm.Chunk {
		return []*llm.Chunk{
			{Type: llm.ChunkText, Content: "甲"},
			{Type: llm.ChunkText, Content: "乙"},
			{Type: llm.ChunkToolCall, ToolCall: &llm.ToolCall{ID: "call_1", Name: "get_pods", Params: `{ "n": 1 }`}},
			{Type: llm.ChunkDone, Usage: &llm.Usage{InputTokens: 3, OutputTokens: 2}},
		}
	}
	if problems := completedProblems(valid(), want, true); len(problems) != 0 {
		t.Fatalf("符合契约的流被判违规：%q", problems)
	}

	cases := []struct {
		name             string
		mutate           func([]*llm.Chunk) []*llm.Chunk
		toolCallIDOnWire bool
	}{
		{"没收尾就关闭", func(c []*llm.Chunk) []*llm.Chunk { return c[:3] }, true},
		{"以错误收尾", func(c []*llm.Chunk) []*llm.Chunk {
			c[3] = &llm.Chunk{Type: llm.ChunkError, Error: llm.ErrStreamIncomplete}
			return c
		}, true},
		{"完成之后还有分段", func(c []*llm.Chunk) []*llm.Chunk {
			return append(c, &llm.Chunk{Type: llm.ChunkText, Content: "丙"})
		}, true},
		{"文字乱序", func(c []*llm.Chunk) []*llm.Chunk { c[0], c[1] = c[1], c[0]; return c }, true},
		{"nil 分段", func(c []*llm.Chunk) []*llm.Chunk { return append([]*llm.Chunk{nil}, c...) }, true},
		{"完成没带用量", func(c []*llm.Chunk) []*llm.Chunk { c[3].Usage = nil; return c }, true},
		{"用量不对", func(c []*llm.Chunk) []*llm.Chunk { c[3].Usage = &llm.Usage{InputTokens: 3}; return c }, true},
		{"没透传工具调用 ID", func(c []*llm.Chunk) []*llm.Chunk { c[2].ToolCall.ID = "other"; return c }, true},
		{"自己生成的 ID 为空", func(c []*llm.Chunk) []*llm.Chunk { c[2].ToolCall.ID = ""; return c }, false},
		{"参数没拼完整", func(c []*llm.Chunk) []*llm.Chunk { c[2].ToolCall.Params = `{"n":`; return c }, true},
		{"工具调用拆成两个", func(c []*llm.Chunk) []*llm.Chunk {
			return append(c[:3:3], &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: &llm.ToolCall{ID: "call_1", Name: "get_pods"}}, c[3])
		}, true},
	}
	for _, c := range cases {
		if problems := completedProblems(c.mutate(valid()), want, c.toolCallIDOnWire); len(problems) == 0 {
			t.Errorf("%s：没判出违规", c.name)
		}
	}
}

func TestFailedProblems(t *testing.T) {
	text := &llm.Chunk{Type: llm.ChunkText, Content: "半截"}
	incomplete := &llm.Chunk{Type: llm.ChunkError, Error: fmt.Errorf("读流: %w", llm.ErrStreamIncomplete)}
	valid := []struct {
		name    string
		chatErr error
		chunks  []*llm.Chunk
	}{
		{"以该类错误收尾", nil, []*llm.Chunk{text, incomplete}},
		{"ChatStream 直接报该类错误", fmt.Errorf("发请求: %w", llm.ErrStreamIncomplete), nil},
	}
	for _, c := range valid {
		if problems := failedProblems(c.chatErr, c.chunks, llm.ErrKindIncomplete); len(problems) != 0 {
			t.Errorf("%s：被判违规 %q", c.name, problems)
		}
	}

	violations := []struct {
		name    string
		chatErr error
		chunks  []*llm.Chunk
	}{
		{"以完成收尾", nil, []*llm.Chunk{text, {Type: llm.ChunkDone, Usage: &llm.Usage{}}}},
		{"没收尾就关闭", nil, []*llm.Chunk{text}},
		{"错误类别不对", nil, []*llm.Chunk{text, {Type: llm.ChunkError, Error: io.ErrUnexpectedEOF}}},
		{"ChatStream 的错误类别不对", errors.New("boom"), nil},
	}
	for _, c := range violations {
		if problems := failedProblems(c.chatErr, c.chunks, llm.ErrKindIncomplete); len(problems) == 0 {
			t.Errorf("%s：没判出违规", c.name)
		}
	}
}

func TestCanceledProblems(t *testing.T) {
	text := &llm.Chunk{Type: llm.ChunkText, Content: "第 1 段"}
	canceled := &llm.Chunk{Type: llm.ChunkError, Error: fmt.Errorf("读流: %w", context.Canceled)}
	valid := []struct {
		name    string
		chatErr error
		chunks  []*llm.Chunk
	}{
		{"没收尾（缓冲区满时只能放弃）", nil, []*llm.Chunk{text}},
		{"以取消收尾", nil, []*llm.Chunk{text, canceled}},
		{"ChatStream 直接报取消", context.Canceled, nil},
	}
	for _, c := range valid {
		if problems := canceledProblems(c.chatErr, c.chunks); len(problems) != 0 {
			t.Errorf("%s：被判违规 %q", c.name, problems)
		}
	}

	violations := []struct {
		name    string
		chatErr error
		chunks  []*llm.Chunk
	}{
		{"以完成收尾", nil, []*llm.Chunk{text, {Type: llm.ChunkDone}}},
		{"报了别的类别", nil, []*llm.Chunk{text, {Type: llm.ChunkError, Error: llm.ErrStreamIncomplete}}},
		{"错误之后还有分段", nil, []*llm.Chunk{canceled, text}},
		{"ChatStream 报了别的类别", llm.ErrStreamIncomplete, nil},
	}
	for _, c := range violations {
		if problems := canceledProblems(c.chatErr, c.chunks); len(problems) == 0 {
			t.Errorf("%s：没判出违规", c.name)
		}
	}
}

// 泄漏检查的阳性对照：卡在发送上的 goroutine 数得到，放开后数不到
func TestGoroutinesIn_SeesBlockedSender(t *testing.T) {
	pkg := reflect.TypeOf(upstream{}).PkgPath()
	baseline := len(goroutinesIn(pkg))

	ch := make(chan int) // 没人读
	done := make(chan struct{})
	go func() {
		defer close(done)
		sendBlocking(ch)
	}()
	deadline := time.Now().Add(time.Second)
	for len(goroutinesIn(pkg)) <= baseline {
		if time.Now().After(deadline) {
			t.Fatal("卡在发送上的 goroutine 没被数到")
		}
		time.Sleep(5 * time.Millisecond)
	}

	<-ch
	<-done
	if stuck := waitGoroutinesIn(pkg, baseline, time.Second); stuck != nil {
		t.Errorf("goroutine 已退出却还被数到：\n%s", stuck[0])
	}
}

func sendBlocking(ch chan<- int) { ch <- 1 }

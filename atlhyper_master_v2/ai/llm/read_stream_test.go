package llm

import (
	"context"
	"errors"
	"testing"
)

func feed(chunks ...*Chunk) <-chan *Chunk {
	ch := make(chan *Chunk, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch
}

func TestReadStream_Done(t *testing.T) {
	var texts []string
	got, err := ReadStream(context.Background(), feed(
		&Chunk{Type: ChunkText, Content: "你"},
		&Chunk{Type: ChunkToolCall, ToolCall: &ToolCall{ID: "1", Name: "query_cluster"}},
		&Chunk{Type: ChunkText, Content: "好"},
		&Chunk{Type: ChunkDone, Usage: &Usage{InputTokens: 3, OutputTokens: 2}, ProviderTurn: "turn"},
	), func(s string) { texts = append(texts, s) })
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "你好" || len(got.ToolCalls) != 1 || got.Usage.OutputTokens != 2 || got.ProviderTurn != "turn" {
		t.Errorf("结果 = %+v", got)
	}
	if len(texts) != 2 {
		t.Errorf("onText 调用 %d 次，want 2", len(texts))
	}
}

// 通道关了却没有完成标记：是中断，不是正常收尾（半截文字不能当作完整回答）
func TestReadStream_ClosedWithoutDoneIsIncomplete(t *testing.T) {
	got, err := ReadStream(context.Background(), feed(&Chunk{Type: ChunkText, Content: "半截"}), nil)
	if !errors.Is(err, ErrStreamIncomplete) {
		t.Fatalf("err = %v, want ErrStreamIncomplete", err)
	}
	if got.Text != "半截" {
		t.Errorf("已收到的部分应照样返回，实际 %q", got.Text)
	}
}

func TestReadStream_Error(t *testing.T) {
	boom := errors.New("rate limited")
	_, err := ReadStream(context.Background(), feed(&Chunk{Type: ChunkError, Error: boom}), nil)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want rate limited", err)
	}
	_, err = ReadStream(context.Background(), feed(&Chunk{Type: ChunkError}), nil)
	if err == nil {
		t.Fatal("没带具体错误的 ChunkError 也要报错")
	}
}

func TestReadStream_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ch := make(chan *Chunk) // 永远不来数据
	_, err := ReadStream(ctx, ch, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

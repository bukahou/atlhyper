package llm

import (
	"context"
	"testing"
	"time"
)

func TestSendChunk_Delivers(t *testing.T) {
	ch := make(chan *Chunk, 1)
	if !SendChunk(context.Background(), ch, &Chunk{Type: ChunkText, Content: "你好"}) {
		t.Fatal("没取消时应发出")
	}
	if got := <-ch; got.Content != "你好" {
		t.Errorf("got %+v", got)
	}
}

// 缓冲区满、调用方已取消且不再读：立即放弃，不能卡住
func TestSendChunk_CanceledWithFullBufferDoesNotBlock(t *testing.T) {
	ch := make(chan *Chunk, 1)
	ch <- &Chunk{Type: ChunkText, Content: "占满"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	returned := make(chan bool)
	go func() { returned <- SendChunk(ctx, ch, &Chunk{Type: ChunkText, Content: "发不出"}) }()
	select {
	case ok := <-returned:
		if ok {
			t.Error("缓冲区满时取消应返回 false")
		}
	case <-time.After(time.Second):
		t.Fatal("取消后仍卡在发送上")
	}
}

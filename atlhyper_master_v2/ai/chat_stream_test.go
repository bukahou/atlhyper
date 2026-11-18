package ai

import (
	"context"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// 对话的工具循环同样要把 Done 带回的整轮回答交给调用方
func TestReadLLMStream_ReturnsProviderTurn(t *testing.T) {
	stream := make(chan *llm.Chunk, 3)
	stream <- &llm.Chunk{Type: llm.ChunkText, Content: "hi"}
	stream <- &llm.Chunk{Type: llm.ChunkToolCall, ToolCall: &llm.ToolCall{ID: "c1", Name: "get_pods", Params: "{}"}}
	stream <- &llm.Chunk{Type: llm.ChunkDone, Usage: &llm.Usage{InputTokens: 1, OutputTokens: 1}, ProviderTurn: "turn-1"}
	close(stream)

	out := make(chan *ChatChunk, 8)
	s := &aiServiceImpl{}
	text, calls, usage, turn, err := s.readLLMStream(context.Background(), stream, out)
	if err != nil {
		t.Fatal(err)
	}
	if text != "hi" || len(calls) != 1 || usage == nil || turn != "turn-1" {
		t.Errorf("text=%q calls=%v usage=%v turn=%v", text, calls, usage, turn)
	}
}

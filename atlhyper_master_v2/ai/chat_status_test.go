package ai

import (
	"context"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
)

// funcLLM 每次调模型都交给 chatStream 处理（按需模拟失败、中途取消）
type funcLLM struct {
	chatStream func(ctx context.Context) (<-chan *llm.Chunk, error)
}

func (f *funcLLM) ChatStream(ctx context.Context, _ *llm.Request) (<-chan *llm.Chunk, error) {
	return f.chatStream(ctx)
}
func (f *funcLLM) Close() error { return nil }

func runChatLoopWith(t *testing.T, ctx context.Context, client llm.LLMClient) ([]*ChatChunk, *usageProviderRepo) {
	t.Helper()
	providers := &usageProviderRepo{}
	s := &aiServiceImpl{
		providerRepo: providers,
		msgRepo:      &memMsgRepo{},
		convRepo:     memConvRepo{},
		tools:        toolkit.NewRegistry(),
		newClient:    func(llm.Config) (llm.LLMClient, error) { return client, nil },
	}
	out := make(chan *ChatChunk, 64)
	s.chatLoop(ctx, "c1", 7, func(string) bool { return true }, nil, out)
	close(out)
	var chunks []*ChatChunk
	for c := range out {
		chunks = append(chunks, c)
	}
	return chunks, providers
}

func equalStatuses(got, want []statusUpdate) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// 调模型就失败（如密钥被拒）：对话报错，AI 设置页的提供商状态也记成对应的错误
func TestChatLoop_ModelCallErrorMarksProviderError(t *testing.T) {
	client := &funcLLM{chatStream: func(context.Context) (<-chan *llm.Chunk, error) {
		return nil, &llm.HTTPError{Provider: "anthropic", StatusCode: 401}
	}}
	chunks, providers := runChatLoopWith(t, context.Background(), client)

	if types := chunkTypes(chunks); types["error"] != 1 {
		t.Fatalf("分段类型 = %v, want 1 个 error", types)
	}
	if want := []statusUpdate{{"error", errorSummary(CodeAuth)}}; !equalStatuses(providers.statuses, want) {
		t.Errorf("提供商状态更新 = %v, want %v", providers.statuses, want)
	}
}

// 用户点了停止（请求被取消）：不是提供商的问题，⛔ 不记成提供商错误，也不当成功清状态
func TestChatLoop_CanceledIsNotAProviderError(t *testing.T) {
	cases := map[string]func(ctx context.Context, cancel context.CancelFunc) (<-chan *llm.Chunk, error){
		"调模型时取消": func(ctx context.Context, cancel context.CancelFunc) (<-chan *llm.Chunk, error) {
			cancel()
			return nil, ctx.Err()
		},
		"读流时取消": func(_ context.Context, cancel context.CancelFunc) (<-chan *llm.Chunk, error) {
			cancel()
			return make(chan *llm.Chunk), nil // 永远不出数据，只能等取消
		},
	}
	for name, stream := range cases {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &funcLLM{chatStream: func(ctx context.Context) (<-chan *llm.Chunk, error) { return stream(ctx, cancel) }}
			chunks, providers := runChatLoopWith(t, ctx, client)

			if len(providers.statuses) != 0 {
				t.Errorf("提供商状态更新 = %v, want 无", providers.statuses)
			}
			if types := chunkTypes(chunks); types["error"] != 0 || types["done"] != 0 {
				t.Errorf("分段类型 = %v, want 既不报错也不发 done（用户自己停的）", types)
			}
		})
	}
}

package ai

import (
	"context"
	"sync"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/database"
)

type usageProviderRepo struct {
	database.AIProviderRepository
	mu     sync.Mutex
	calls  int
	tokens int64
}

func (r *usageProviderRepo) List(context.Context) ([]*database.AIProvider, error) {
	return []*database.AIProvider{{ID: 1, Name: "p", Provider: "openai", Model: "m", Roles: []string{RoleChat}}}, nil
}
func (r *usageProviderRepo) IncrementUsage(_ context.Context, _ int64, requests, tokens int64, _ float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls += int(requests)
	r.tokens += tokens
	return nil
}
func (r *usageProviderRepo) UpdateStatus(context.Context, int64, string, string) error { return nil }

type memMsgRepo struct {
	database.AIMessageRepository
	saved []*database.AIMessage
}

func (r *memMsgRepo) Create(_ context.Context, m *database.AIMessage) error {
	r.saved = append(r.saved, m)
	return nil
}
func (r *memMsgRepo) ListByConversation(context.Context, int64) ([]*database.AIMessage, error) {
	return r.saved, nil
}

type memConvRepo struct {
	database.AIConversationRepository
}

func (memConvRepo) GetByID(_ context.Context, id int64) (*database.AIConversation, error) {
	return &database.AIConversation{ID: id}, nil
}
func (memConvRepo) Update(context.Context, *database.AIConversation) error { return nil }

func runChatLoop(t *testing.T, rounds ...[]*llm.Chunk) ([]*ChatChunk, *usageProviderRepo, *memMsgRepo) {
	t.Helper()
	providers := &usageProviderRepo{}
	msgs := &memMsgRepo{}
	s := &aiServiceImpl{
		providerRepo: providers,
		msgRepo:      msgs,
		convRepo:     memConvRepo{},
		tools:        toolkit.NewRegistry(),
		newClient: func(llm.Config) (llm.LLMClient, error) {
			return &scriptedLLM{rounds: rounds}, nil
		},
	}
	out := make(chan *ChatChunk, 64)
	s.chatLoop(context.Background(), "c1", 7, func(string) bool { return true }, nil, out)
	close(out)
	var chunks []*ChatChunk
	for c := range out {
		chunks = append(chunks, c)
	}
	return chunks, providers, msgs
}

func chunkTypes(chunks []*ChatChunk) map[string]int {
	types := map[string]int{}
	for _, c := range chunks {
		types[c.Type]++
	}
	return types
}

// 模型的流没有完成标记就断了：报错（不当作完整回答、不发 done、不把半截文字存成最终回答），但用量照样记账
func TestChatLoop_IncompleteStreamIsAnErrorButUsageIsRecorded(t *testing.T) {
	chunks, providers, msgs := runChatLoop(t, []*llm.Chunk{text("半截")})

	types := chunkTypes(chunks)
	if types["error"] != 1 || types["done"] != 0 {
		t.Fatalf("分段类型 = %v, want 1 个 error、0 个 done", types)
	}
	if providers.calls != 1 {
		t.Errorf("用量记账 %d 次，want 1（失败的调用也要记）", providers.calls)
	}
	for _, m := range msgs.saved {
		if m.Role == "assistant" {
			t.Errorf("半截回答不应存成最终回答: %q", m.Content)
		}
	}
}

func TestChatLoop_CompleteAnswer(t *testing.T) {
	chunks, providers, msgs := runChatLoop(t, []*llm.Chunk{text("一切正常"), done(10, 5)})

	types := chunkTypes(chunks)
	if types["done"] != 1 || types["error"] != 0 {
		t.Fatalf("分段类型 = %v", types)
	}
	if providers.calls != 1 || providers.tokens != 15 {
		t.Errorf("记账 calls=%d tokens=%d, want 1 / 15", providers.calls, providers.tokens)
	}
	if len(msgs.saved) != 1 || msgs.saved[0].Content != "一切正常" {
		t.Errorf("最终回答应存一条，实际 %+v", msgs.saved)
	}
}

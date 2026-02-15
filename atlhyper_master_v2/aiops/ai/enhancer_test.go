// atlhyper_master_v2/aiops/ai/enhancer_test.go
// AI 增强服务测试
package ai

import (
	"context"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
	"AtlHyper/atlhyper_master_v2/database"
)

// ==================== Mock 实现 ====================

// mockIncidentRepo 模拟事件仓库
type mockIncidentRepo struct {
	incident *database.AIOpsIncident
	entities []*database.AIOpsIncidentEntity
	timeline []*database.AIOpsIncidentTimeline
	byEntity []*database.AIOpsIncident
}

func (m *mockIncidentRepo) CreateIncident(ctx context.Context, inc *database.AIOpsIncident) error {
	return nil
}
func (m *mockIncidentRepo) GetByID(ctx context.Context, id string) (*database.AIOpsIncident, error) {
	return m.incident, nil
}
func (m *mockIncidentRepo) UpdateState(ctx context.Context, id, state, severity string) error {
	return nil
}
func (m *mockIncidentRepo) Resolve(ctx context.Context, id string, resolvedAt time.Time) error {
	return nil
}
func (m *mockIncidentRepo) UpdateRootCause(ctx context.Context, id, rootCause string) error {
	return nil
}
func (m *mockIncidentRepo) UpdatePeakRisk(ctx context.Context, id string, peakRisk float64) error {
	return nil
}
func (m *mockIncidentRepo) IncrementRecurrence(ctx context.Context, id string) error { return nil }
func (m *mockIncidentRepo) List(ctx context.Context, opts database.AIOpsIncidentQueryOpts) ([]*database.AIOpsIncident, error) {
	return nil, nil
}
func (m *mockIncidentRepo) Count(ctx context.Context, opts database.AIOpsIncidentQueryOpts) (int, error) {
	return 0, nil
}
func (m *mockIncidentRepo) AddEntity(ctx context.Context, entity *database.AIOpsIncidentEntity) error {
	return nil
}
func (m *mockIncidentRepo) GetEntities(ctx context.Context, incidentID string) ([]*database.AIOpsIncidentEntity, error) {
	return m.entities, nil
}
func (m *mockIncidentRepo) AddTimeline(ctx context.Context, entry *database.AIOpsIncidentTimeline) error {
	return nil
}
func (m *mockIncidentRepo) GetTimeline(ctx context.Context, incidentID string) ([]*database.AIOpsIncidentTimeline, error) {
	return m.timeline, nil
}
func (m *mockIncidentRepo) GetIncidentStats(ctx context.Context, clusterID string, since time.Time) (*database.AIOpsIncidentStatsRaw, error) {
	return nil, nil
}
func (m *mockIncidentRepo) TopRootCauses(ctx context.Context, clusterID string, since time.Time, limit int) ([]database.AIOpsRootCauseCount, error) {
	return nil, nil
}
func (m *mockIncidentRepo) ListByEntity(ctx context.Context, entityKey string, since time.Time) ([]*database.AIOpsIncident, error) {
	return m.byEntity, nil
}

// mockLLMClient 模拟 LLM 客户端
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) ChatStream(ctx context.Context, req *llm.Request) (<-chan *llm.Chunk, error) {
	if m.err != nil {
		return nil, m.err
	}
	ch := make(chan *llm.Chunk, 2)
	ch <- &llm.Chunk{Type: llm.ChunkText, Content: m.response}
	ch <- &llm.Chunk{Type: llm.ChunkDone}
	close(ch)
	return ch, nil
}

func (m *mockLLMClient) Close() error { return nil }

// ==================== 测试 ====================

func makeTestIncident() *database.AIOpsIncident {
	return &database.AIOpsIncident{
		ID:        "inc-test-001",
		ClusterID: "cluster-1",
		State:     "incident",
		Severity:  "high",
		RootCause: "node/worker-3",
		PeakRisk:  85.0,
		StartedAt: time.Now().Add(-30 * time.Minute),
		DurationS: 1800,
		CreatedAt: time.Now().Add(-30 * time.Minute),
	}
}

func makeTestEntities() []*database.AIOpsIncidentEntity {
	return []*database.AIOpsIncidentEntity{
		{IncidentID: "inc-test-001", EntityKey: "node/worker-3", EntityType: "node", RLocal: 0.9, RFinal: 0.9, Role: "root_cause"},
		{IncidentID: "inc-test-001", EntityKey: "default/pod/api-abc", EntityType: "pod", RLocal: 0.6, RFinal: 0.78, Role: "affected"},
		{IncidentID: "inc-test-001", EntityKey: "default/service/api", EntityType: "service", RLocal: 0.5, RFinal: 0.85, Role: "symptom"},
	}
}

func makeTestTimeline() []*database.AIOpsIncidentTimeline {
	now := time.Now()
	return []*database.AIOpsIncidentTimeline{
		{ID: 1, IncidentID: "inc-test-001", Timestamp: now.Add(-30 * time.Minute), EventType: "anomaly_detected", EntityKey: "node/worker-3", Detail: "内存使用率超过基线 3.2σ"},
		{ID: 2, IncidentID: "inc-test-001", Timestamp: now.Add(-28 * time.Minute), EventType: "state_change", EntityKey: "node/worker-3", Detail: "Healthy → Warning"},
		{ID: 3, IncidentID: "inc-test-001", Timestamp: now.Add(-25 * time.Minute), EventType: "state_escalated", EntityKey: "node/worker-3", Detail: "Warning → Incident"},
	}
}

// TestSummarize_NormalIncident 正常事件的 AI 摘要
func TestSummarize_NormalIncident(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
		byEntity: []*database.AIOpsIncident{
			{ID: "inc-old-001", RootCause: "node/worker-3", StartedAt: time.Now().Add(-7 * 24 * time.Hour), DurationS: 2700},
		},
	}

	llmResponse := `{
		"summary": "worker-3 节点内存压力导致 API 服务降级",
		"rootCauseAnalysis": "根因链: Node memory → Pod OOM → Service errors",
		"recommendations": [
			{"priority": 1, "action": "检查 worker-3 内存使用", "reason": "内存持续高位", "impact": "防止 OOM"}
		],
		"similarPattern": "与 7 天前事件模式一致"
	}`

	mockClient := &mockLLMClient{response: llmResponse}
	factory := func(ctx context.Context) (llm.LLMClient, error) {
		return mockClient, nil
	}

	enhancer := NewEnhancer(repo, factory)
	result, err := enhancer.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}

	if result.IncidentID != "inc-test-001" {
		t.Fatalf("expected incidentId 'inc-test-001', got '%s'", result.IncidentID)
	}
	if result.Summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if result.RootCauseAnalysis == "" {
		t.Fatal("expected non-empty rootCauseAnalysis")
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].Priority != 1 {
		t.Fatalf("expected priority 1, got %d", result.Recommendations[0].Priority)
	}
	if len(result.SimilarIncidents) != 1 {
		t.Fatalf("expected 1 similar incident, got %d", len(result.SimilarIncidents))
	}
	if result.GeneratedAt == 0 {
		t.Fatal("expected non-zero generatedAt")
	}
}

// TestSummarize_LLMParseError LLM 返回非 JSON 格式时降级
func TestSummarize_LLMParseError(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	// LLM 返回纯文本（非 JSON）
	mockClient := &mockLLMClient{response: "这是一个关于内存压力的事件分析结果。"}
	factory := func(ctx context.Context) (llm.LLMClient, error) {
		return mockClient, nil
	}

	enhancer := NewEnhancer(repo, factory)
	result, err := enhancer.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("Summarize should not fail on parse error: %v", err)
	}

	// 降级：整段文本作为 summary
	if result.Summary == "" {
		t.Fatal("expected fallback summary")
	}
	if result.Summary != "这是一个关于内存压力的事件分析结果。" {
		t.Fatalf("expected fallback to raw text, got '%s'", result.Summary)
	}
}

// TestSummarize_LLMUnavailable LLM 不可用时返回错误
func TestSummarize_LLMUnavailable(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	factory := func(ctx context.Context) (llm.LLMClient, error) {
		return nil, fmt.Errorf("LLM 连接超时")
	}

	enhancer := NewEnhancer(repo, factory)
	_, err := enhancer.Summarize(context.Background(), "inc-test-001")
	if err == nil {
		t.Fatal("expected error when LLM unavailable")
	}
}

// TestSummarize_IncidentNotFound 事件不存在
func TestSummarize_IncidentNotFound(t *testing.T) {
	repo := &mockIncidentRepo{incident: nil}
	factory := func(ctx context.Context) (llm.LLMClient, error) {
		return &mockLLMClient{}, nil
	}

	enhancer := NewEnhancer(repo, factory)
	_, err := enhancer.Summarize(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent incident")
	}
}

// TestSummarize_NoHistoricalPatterns 无历史相似事件
func TestSummarize_NoHistoricalPatterns(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
		byEntity: nil,
	}

	llmResponse := `{"summary": "事件摘要", "rootCauseAnalysis": "根因分析", "recommendations": []}`
	mockClient := &mockLLMClient{response: llmResponse}
	factory := func(ctx context.Context) (llm.LLMClient, error) {
		return mockClient, nil
	}

	enhancer := NewEnhancer(repo, factory)
	result, err := enhancer.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if len(result.SimilarIncidents) != 0 {
		t.Fatalf("expected 0 similar incidents, got %d", len(result.SimilarIncidents))
	}
}

// TestSummarize_MarkdownCodeBlock LLM 用 markdown 代码块包裹 JSON
func TestSummarize_MarkdownCodeBlock(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	llmResponse := "```json\n{\"summary\": \"代码块摘要\", \"rootCauseAnalysis\": \"分析\", \"recommendations\": []}\n```"
	mockClient := &mockLLMClient{response: llmResponse}
	factory := func(ctx context.Context) (llm.LLMClient, error) {
		return mockClient, nil
	}

	enhancer := NewEnhancer(repo, factory)
	result, err := enhancer.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("Summarize failed: %v", err)
	}
	if result.Summary != "代码块摘要" {
		t.Fatalf("expected '代码块摘要', got '%s'", result.Summary)
	}
}

// ==================== Context Builder 测试 ====================

// TestBuildIncidentContext 上下文构建正确性
func TestBuildIncidentContext(t *testing.T) {
	incident := makeTestIncident()
	entities := makeTestEntities()
	timeline := makeTestTimeline()
	historical := []*database.AIOpsIncident{
		{ID: "inc-old-001", RootCause: "node/worker-3", StartedAt: time.Now().Add(-7 * 24 * time.Hour), DurationS: 2700},
	}

	ctx := BuildIncidentContext(incident, entities, timeline, historical)

	if ctx.IncidentSummary == "" {
		t.Fatal("expected non-empty IncidentSummary")
	}
	if ctx.RootCauseEntity == "" {
		t.Fatal("expected non-empty RootCauseEntity")
	}
	if ctx.AffectedEntities == "" {
		t.Fatal("expected non-empty AffectedEntities")
	}
	if ctx.TimelineText == "" {
		t.Fatal("expected non-empty TimelineText")
	}
	if ctx.HistoricalContext == "" {
		t.Fatal("expected non-empty HistoricalContext")
	}
}

// TestBuildIncidentContext_NoRootCause 无根因实体
func TestBuildIncidentContext_NoRootCause(t *testing.T) {
	incident := makeTestIncident()
	entities := []*database.AIOpsIncidentEntity{
		{IncidentID: "inc-test-001", EntityKey: "default/pod/api-abc", Role: "affected"},
	}

	ctx := BuildIncidentContext(incident, entities, nil, nil)
	if ctx.RootCauseEntity != "根因实体: 未识别" {
		t.Fatalf("expected '根因实体: 未识别', got '%s'", ctx.RootCauseEntity)
	}
}

// ==================== extractJSON 测试 ====================

func TestExtractJSON_Direct(t *testing.T) {
	input := `{"key": "value"}`
	result := extractJSON(input)
	if result != `{"key": "value"}` {
		t.Fatalf("expected direct JSON, got '%s'", result)
	}
}

func TestExtractJSON_MarkdownBlock(t *testing.T) {
	input := "```json\n{\"key\": \"value\"}\n```"
	result := extractJSON(input)
	if result != `{"key": "value"}` {
		t.Fatalf("expected extracted JSON, got '%s'", result)
	}
}

func TestExtractJSON_WithSurroundingText(t *testing.T) {
	input := "Here is the analysis:\n{\"key\": \"value\"}\nDone."
	result := extractJSON(input)
	if result != `{"key": "value"}` {
		t.Fatalf("expected extracted JSON, got '%s'", result)
	}
}

// ==================== formatDuration 测试 ====================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds  int64
		expected string
	}{
		{0, "进行中"},
		{30, "30 秒"},
		{120, "2 分钟"},
		{3600, "1.0 小时"},
		{7200, "2.0 小时"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDuration(%d) = '%s', expected '%s'", tt.seconds, result, tt.expected)
		}
	}
}

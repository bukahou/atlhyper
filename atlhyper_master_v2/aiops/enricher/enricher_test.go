// atlhyper_master_v2/aiops/enricher/enricher_test.go
// Enricher 测试
package enricher

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
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

// mockAIService 模拟 ai.AIService
type mockAIService struct {
	completeResponse string
	completeErr      error
	callCount        int
}

func (m *mockAIService) CreateConversation(ctx context.Context, userID int64, clusterID, title string) (*ai.Conversation, error) {
	return nil, nil
}
func (m *mockAIService) Chat(ctx context.Context, req *ai.ChatRequest) (<-chan *ai.ChatChunk, error) {
	return nil, nil
}
func (m *mockAIService) GetConversations(ctx context.Context, userID int64, limit, offset int) ([]*ai.Conversation, error) {
	return nil, nil
}
func (m *mockAIService) GetMessages(ctx context.Context, conversationID int64) ([]*ai.Message, error) {
	return nil, nil
}
func (m *mockAIService) DeleteConversation(ctx context.Context, conversationID int64) error {
	return nil
}
func (m *mockAIService) RegisterTool(name string, handler ai.ToolHandler) {}
func (m *mockAIService) Analyze(ctx context.Context, req *ai.AnalyzeRequest) (*ai.AnalyzeResult, error) {
	return nil, nil
}
func (m *mockAIService) Complete(ctx context.Context, req *ai.CompleteRequest) (*ai.CompleteResult, error) {
	m.callCount++
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return &ai.CompleteResult{
		Response:     m.completeResponse,
		InputTokens:  100,
		OutputTokens: 50,
		ProviderID:   1,
		ProviderName: "test",
		Model:        "test-model",
	}, nil
}
func (m *mockAIService) GetToolExecuteFunc() func(ctx context.Context, clusterID string, tc *llm.ToolCall) (string, error) {
	return nil
}
func (m *mockAIService) GetToolDefs() []llm.ToolDefinition { return nil }

// ==================== 测试辅助 ====================

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

func newMockAIService(response string) *mockAIService {
	return &mockAIService{completeResponse: response}
}

func newMockAIServiceErr(err error) *mockAIService {
	return &mockAIService{completeErr: err}
}

// ==================== 测试 ====================

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

	svc := newMockAIService(llmResponse)
	e := NewEnricher(repo, nil, svc)
	result, err := e.Summarize(context.Background(), "inc-test-001")
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

	svc := newMockAIService("这是一个关于内存压力的事件分析结果。")
	e := NewEnricher(repo, nil, svc)
	result, err := e.Summarize(context.Background(), "inc-test-001")
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

	svc := newMockAIServiceErr(fmt.Errorf("LLM 连接超时"))
	e := NewEnricher(repo, nil, svc)
	_, err := e.Summarize(context.Background(), "inc-test-001")
	if err == nil {
		t.Fatal("expected error when LLM unavailable")
	}
}

// TestSummarize_IncidentNotFound 事件不存在
func TestSummarize_IncidentNotFound(t *testing.T) {
	repo := &mockIncidentRepo{incident: nil}
	svc := newMockAIService("")
	e := NewEnricher(repo, nil, svc)
	_, err := e.Summarize(context.Background(), "nonexistent")
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

	svc := newMockAIService(`{"summary": "事件摘要", "rootCauseAnalysis": "根因分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	result, err := e.Summarize(context.Background(), "inc-test-001")
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
	svc := newMockAIService(llmResponse)
	e := NewEnricher(repo, nil, svc)
	result, err := e.Summarize(context.Background(), "inc-test-001")
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

// ==================== Rate Limit 测试 ====================

// TestRateLimit_Cooldown 同一事件在冷却期内被拒绝
func TestRateLimit_Cooldown(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 2 * time.Second // 测试用短冷却

	// 第一次调用应成功
	_, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}
	if svc.callCount != 1 {
		t.Fatalf("expected 1 LLM call, got %d", svc.callCount)
	}

	// 立即再次调用同一事件应被 rate limit 拒绝
	_, err = e.Summarize(context.Background(), "inc-test-001")
	if err == nil {
		t.Fatal("second call should be rate limited")
	}
	if !strings.Contains(err.Error(), "请等待") {
		t.Fatalf("expected rate limit error, got: %v", err)
	}
	if svc.callCount != 1 {
		t.Fatalf("LLM should not be called again, count=%d", svc.callCount)
	}

	// 不同事件不受影响
	_, err = e.Summarize(context.Background(), "inc-test-002")
	if err == nil || !strings.Contains(err.Error(), "事件不存在") {
		// inc-test-002 不存在于 mock repo，但不应该被 rate limit 拒绝
		// 应该先通过 rate limit，然后在查数据阶段失败
		if err != nil && strings.Contains(err.Error(), "请等待") {
			t.Fatal("different incident should not be rate limited")
		}
	}
}

// TestRateLimit_CooldownExpired 冷却期过后可以再次调用
func TestRateLimit_CooldownExpired(t *testing.T) {
	repo := &mockIncidentRepo{
		incident: makeTestIncident(),
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 50 * time.Millisecond // 极短冷却

	// 第一次调用
	_, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// 等待冷却期过
	time.Sleep(100 * time.Millisecond)

	// 第二次调用应成功
	_, err = e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("call after cooldown should succeed: %v", err)
	}
	if svc.callCount != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", svc.callCount)
	}
}

// ==================== 结果缓存测试 ====================

// TestCache_StableIncident 已稳定事件的结果被缓存
func TestCache_StableIncident(t *testing.T) {
	inc := makeTestIncident()
	inc.State = "stable" // 已完结

	repo := &mockIncidentRepo{
		incident: inc,
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 0 // 关闭 rate limit

	// 第一次调用 → LLM
	result1, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if svc.callCount != 1 {
		t.Fatalf("expected 1 LLM call, got %d", svc.callCount)
	}

	// 第二次调用 → 缓存命中，不调 LLM
	result2, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if svc.callCount != 1 {
		t.Fatalf("expected still 1 LLM call (cached), got %d", svc.callCount)
	}

	// 缓存返回同一结果
	if result1.Summary != result2.Summary {
		t.Fatalf("cached result mismatch: '%s' vs '%s'", result1.Summary, result2.Summary)
	}
}

// TestCache_RecoveryIncident recovery 状态也被缓存
func TestCache_RecoveryIncident(t *testing.T) {
	inc := makeTestIncident()
	inc.State = "recovery"

	repo := &mockIncidentRepo{
		incident: inc,
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 0

	_, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	_, err = e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if svc.callCount != 1 {
		t.Fatalf("expected 1 LLM call for recovery state, got %d", svc.callCount)
	}
}

// TestCache_ActiveIncident_NoCache 进行中的事件不缓存
func TestCache_ActiveIncident_NoCache(t *testing.T) {
	inc := makeTestIncident()
	inc.State = "incident" // 进行中

	repo := &mockIncidentRepo{
		incident: inc,
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 0 // 关闭 rate limit

	// 第一次调用
	_, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// 第二次调用 → 不应命中缓存，应该再次调 LLM
	_, err = e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if svc.callCount != 2 {
		t.Fatalf("expected 2 LLM calls (no cache for active), got %d", svc.callCount)
	}
}

// TestCache_WarningState_NoCache warning 状态不缓存
func TestCache_WarningState_NoCache(t *testing.T) {
	inc := makeTestIncident()
	inc.State = "warning"

	repo := &mockIncidentRepo{
		incident: inc,
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 0

	_, _ = e.Summarize(context.Background(), "inc-test-001")
	_, _ = e.Summarize(context.Background(), "inc-test-001")

	if svc.callCount != 2 {
		t.Fatalf("expected 2 LLM calls (no cache for warning), got %d", svc.callCount)
	}
}

// TestCache_SkipsRateLimit 缓存命中时不受 rate limit 限制
func TestCache_SkipsRateLimit(t *testing.T) {
	inc := makeTestIncident()
	inc.State = "stable"

	repo := &mockIncidentRepo{
		incident: inc,
		entities: makeTestEntities(),
		timeline: makeTestTimeline(),
	}

	svc := newMockAIService(`{"summary": "摘要", "rootCauseAnalysis": "分析", "recommendations": []}`)
	e := NewEnricher(repo, nil, svc)
	e.cooldown = 1 * time.Hour // 极长冷却期

	// 第一次调用
	_, err := e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// 立即第二次 → 缓存命中，不受 rate limit 限制
	_, err = e.Summarize(context.Background(), "inc-test-001")
	if err != nil {
		t.Fatalf("cached call should not be rate limited: %v", err)
	}
	if svc.callCount != 1 {
		t.Fatalf("expected 1 LLM call, got %d", svc.callCount)
	}
}

// ==================== Token 预估 / Prompt 截断测试 ====================

// TestPromptTruncation_NormalPrompt 正常长度不截断
func TestPromptTruncation_NormalPrompt(t *testing.T) {
	inc := makeTestIncident()
	entities := makeTestEntities()
	timeline := makeTestTimeline()

	e := NewEnricher(nil, nil, nil)
	prompt := e.buildPromptWithTruncation(inc, entities, timeline, nil)

	totalChars := len(prompt.System) + len(prompt.User)
	if totalChars > MaxPromptChars {
		t.Fatalf("normal prompt should be under limit, got %d", totalChars)
	}
	// 确认包含关键内容
	if !strings.Contains(prompt.User, "node/worker-3") {
		t.Fatal("prompt should contain root cause entity")
	}
}

// TestPromptTruncation_LargePrompt 超长 Prompt 被截断
func TestPromptTruncation_LargePrompt(t *testing.T) {
	inc := makeTestIncident()

	// 构造大量实体
	entities := make([]*database.AIOpsIncidentEntity, 200)
	for i := range entities {
		entities[i] = &database.AIOpsIncidentEntity{
			IncidentID: "inc-test-001",
			EntityKey:  fmt.Sprintf("default/pod/very-long-name-pod-%d-with-extra-padding-text", i),
			EntityType: "pod",
			RLocal:     0.5,
			RFinal:     0.6,
			Role:       "affected",
		}
	}

	// 构造大量时间线
	now := time.Now()
	timeline := make([]*database.AIOpsIncidentTimeline, 100)
	for i := range timeline {
		timeline[i] = &database.AIOpsIncidentTimeline{
			ID:         int64(i + 1),
			IncidentID: "inc-test-001",
			Timestamp:  now.Add(-time.Duration(100-i) * time.Minute),
			EventType:  "metric_anomaly",
			EntityKey:  fmt.Sprintf("default/pod/pod-%d", i),
			Detail:     strings.Repeat("这是一段很长的详情描述文本用于填充", 5),
		}
	}

	// 构造大量历史事件
	historical := make([]*database.AIOpsIncident, 50)
	for i := range historical {
		historical[i] = &database.AIOpsIncident{
			ID:        fmt.Sprintf("inc-hist-%03d", i),
			RootCause: fmt.Sprintf("node/worker-%d", i),
			StartedAt: now.Add(-time.Duration(i+1) * 24 * time.Hour),
			DurationS: int64(3600 + i*100),
		}
	}

	e := NewEnricher(nil, nil, nil)
	prompt := e.buildPromptWithTruncation(inc, entities, timeline, historical)

	totalChars := len(prompt.System) + len(prompt.User)
	if totalChars > MaxPromptChars {
		t.Fatalf("truncated prompt should be under limit, got %d (max %d)", totalChars, MaxPromptChars)
	}
}

// ==================== Context Builder 截断测试 ====================

// TestBuildTimeline_Truncation 时间线超过上限时截断
func TestBuildTimeline_Truncation(t *testing.T) {
	now := time.Now()
	timeline := make([]*database.AIOpsIncidentTimeline, 50)
	for i := range timeline {
		timeline[i] = &database.AIOpsIncidentTimeline{
			ID:        int64(i + 1),
			Timestamp: now.Add(-time.Duration(50-i) * time.Minute),
			EventType: "test",
			EntityKey: fmt.Sprintf("entity-%d", i),
			Detail:    "detail",
		}
	}

	result := buildTimeline(timeline)

	// 应包含省略提示
	if !strings.Contains(result, "省略前") {
		t.Fatal("expected truncation notice for timeline")
	}
	// 应包含最后一条（最新的）
	if !strings.Contains(result, "entity-49") {
		t.Fatal("expected latest entry to be preserved")
	}
}

// TestBuildHistorical_Truncation 历史事件超过上限时截断
func TestBuildHistorical_Truncation(t *testing.T) {
	now := time.Now()
	incidents := make([]*database.AIOpsIncident, 20)
	for i := range incidents {
		incidents[i] = &database.AIOpsIncident{
			ID:        fmt.Sprintf("inc-%03d", i),
			RootCause: "node/worker-1",
			StartedAt: now.Add(-time.Duration(i+1) * 24 * time.Hour),
			DurationS: 3600,
		}
	}

	result := buildHistorical(incidents)

	// 应包含省略提示
	if !strings.Contains(result, "省略") {
		t.Fatal("expected truncation notice for historical")
	}
	// 总数应显示 20
	if !strings.Contains(result, "20 个") {
		t.Fatal("expected total count 20 in header")
	}
}

// TestBuildEntities_Truncation 实体数超过上限时截断
func TestBuildEntities_Truncation(t *testing.T) {
	entities := make([]*database.AIOpsIncidentEntity, 60)
	for i := range entities {
		entities[i] = &database.AIOpsIncidentEntity{
			EntityKey:  fmt.Sprintf("entity-%d", i),
			EntityType: "pod",
			RFinal:     0.5,
			Role:       "affected",
		}
	}

	result := buildEntities(entities)

	// 应包含省略提示
	if !strings.Contains(result, "省略") {
		t.Fatal("expected truncation notice for entities")
	}
	// 总数应显示 60
	if !strings.Contains(result, "60 个") {
		t.Fatal("expected total count 60 in header")
	}
}

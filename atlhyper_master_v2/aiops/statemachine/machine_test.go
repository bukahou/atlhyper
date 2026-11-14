// atlhyper_master_v2/aiops/statemachine/machine_test.go
// 状态机测试: 5 条状态转换路径、持续时间不足不触发、告警抑制、Recovery→Stable
package statemachine

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// mockCallback 模拟 TransitionCallback
type mockCallback struct {
	warningCreated  int
	stateEscalated  int
	recoveryStarted int
	recurrence      int
	stable          int
	lastIncidentID  string
}

func (m *mockCallback) OnWarningCreated(ctx context.Context, clusterID, entityKey string, risk *aiops.EntityRisk, now time.Time) string {
	m.warningCreated++
	m.lastIncidentID = "inc-test"
	return "inc-test"
}

func (m *mockCallback) OnStateEscalated(ctx context.Context, incidentID string, state aiops.EntityState, risk *aiops.EntityRisk, now time.Time) {
	m.stateEscalated++
}

func (m *mockCallback) OnRecoveryStarted(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time) {
	m.recoveryStarted++
}

func (m *mockCallback) OnRecurrence(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time) {
	m.recurrence++
}

func (m *mockCallback) OnStable(ctx context.Context, incidentID string, entityKey string, now time.Time) {
	m.stable++
}

// testCluster 单集群用例统一使用的集群 ID
const testCluster = "cluster-1"

func testRef(entityKey string) aiops.EntityRef {
	return aiops.EntityRef{ClusterID: testCluster, EntityKey: entityKey}
}

func makeEntityRisks(key string, rFinal float64) map[string]*aiops.EntityRisk {
	return map[string]*aiops.EntityRisk{
		key: {
			EntityKey: key,
			RFinal:    rFinal,
		},
	}
}

// TestHealthyToWarning 测试 Healthy → Warning 转换
// 条件: R_final > 0.2 持续 > 2 分钟
func TestHealthyToWarning(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"
	risks := makeEntityRisks(entity, 0.3) // > 0.2

	// 第一次评估：条件满足，开始计时
	sm.Evaluate(ctx, "cluster-1", risks, nil)
	entry := sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateHealthy {
		t.Fatalf("expected Healthy, got %s", entry.CurrentState)
	}
	if cb.warningCreated != 0 {
		t.Fatal("should not create warning yet (duration < 2min)")
	}

	// 模拟 3 分钟后再次评估
	entry.ConditionMetSince = time.Now().Add(-3 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", risks, nil)

	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateWarning {
		t.Fatalf("expected Warning, got %s", entry.CurrentState)
	}
	if cb.warningCreated != 1 {
		t.Fatalf("expected 1 warningCreated, got %d", cb.warningCreated)
	}
	if entry.IncidentID != "inc-test" {
		t.Fatalf("expected incident ID 'inc-test', got '%s'", entry.IncidentID)
	}
}

// TestWarningToIncident 测试 Warning → Incident 转换
// 条件: R_final > 0.5 持续 > 5 分钟
func TestWarningToIncident(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	// 先到 Warning 状态
	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateWarning,
		IncidentID:   "inc-1",
	}
	sm.mu.Unlock()

	risks := makeEntityRisks(entity, 0.6) // > 0.5

	// 第一次：开始计时
	sm.Evaluate(ctx, "cluster-1", risks, nil)
	entry := sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateWarning {
		t.Fatalf("expected Warning (duration < 5min), got %s", entry.CurrentState)
	}

	// 模拟 6 分钟后
	entry.ConditionMetSince = time.Now().Add(-6 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", risks, nil)

	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateIncident {
		t.Fatalf("expected Incident, got %s", entry.CurrentState)
	}
	if cb.stateEscalated != 1 {
		t.Fatalf("expected 1 stateEscalated, got %d", cb.stateEscalated)
	}
}

// TestWarningToIncident_ClusterRiskTrigger 测试 Warning → Incident 由 ClusterRisk > 80 触发
func TestWarningToIncident_ClusterRiskTrigger(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/pod/pod-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateWarning,
		IncidentID:   "inc-2",
	}
	sm.mu.Unlock()

	// R_final 不够 0.5，但 ClusterRisk > 80
	risks := makeEntityRisks(entity, 0.3)
	clusterRisk := &aiops.ClusterRisk{Risk: 85}

	// 第一次：开始计时（条件由 ClusterRisk 满足）
	sm.Evaluate(ctx, "cluster-1", risks, clusterRisk)

	// 模拟 6 分钟后
	entry := sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-6 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", risks, clusterRisk)

	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateIncident {
		t.Fatalf("expected Incident (ClusterRisk trigger), got %s", entry.CurrentState)
	}
}

// TestIncidentToRecovery 测试 Incident → Recovery 转换
// 条件: R_final < 0.15 持续 > 10 分钟
func TestIncidentToRecovery(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateIncident,
		IncidentID:   "inc-3",
	}
	sm.mu.Unlock()

	risks := makeEntityRisks(entity, 0.1) // < 0.15

	// 第一次：开始计时
	sm.Evaluate(ctx, "cluster-1", risks, nil)

	// 模拟 11 分钟后
	entry := sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-11 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", risks, nil)

	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateRecovery {
		t.Fatalf("expected Recovery, got %s", entry.CurrentState)
	}
	if cb.recoveryStarted != 1 {
		t.Fatalf("expected 1 recoveryStarted, got %d", cb.recoveryStarted)
	}
}

// TestRecoveryToWarning_Recurrence 测试 Recovery → Warning（复发）
// 条件: R_final > 0.2（立即触发）
func TestRecoveryToWarning_Recurrence(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateRecovery,
		IncidentID:   "inc-4",
	}
	sm.mu.Unlock()

	risks := makeEntityRisks(entity, 0.3) // > 0.2

	// 立即触发（MinDuration = 0）
	sm.Evaluate(ctx, "cluster-1", risks, nil)

	entry := sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateWarning {
		t.Fatalf("expected Warning (recurrence), got %s", entry.CurrentState)
	}
	if cb.recurrence != 1 {
		t.Fatalf("expected 1 recurrence, got %d", cb.recurrence)
	}
}

// TestDurationNotMet 测试持续时间不足不触发转换
func TestDurationNotMet(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"
	risks := makeEntityRisks(entity, 0.3) // > 0.2

	// 连续评估 3 次（间隔很短），不应触发
	for i := 0; i < 3; i++ {
		sm.Evaluate(ctx, "cluster-1", risks, nil)
	}

	entry := sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateHealthy {
		t.Fatalf("expected Healthy (duration not met), got %s", entry.CurrentState)
	}
	if cb.warningCreated != 0 {
		t.Fatal("should not create warning (duration not met)")
	}
}

// TestConditionResets 测试条件不满足时计时重置
func TestConditionResets(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	// 高风险
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.3), nil)
	entry := sm.GetEntry(testCluster, entity)
	if entry.ConditionMetSince == 0 {
		t.Fatal("ConditionMetSince should be set")
	}

	// 风险降低到阈值以下 → 计时重置
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.1), nil)
	entry = sm.GetEntry(testCluster, entity)
	if entry.ConditionMetSince != 0 {
		t.Fatal("ConditionMetSince should be reset")
	}
}

// TestShouldSuppress 测试告警抑制
func TestShouldSuppress(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)

	entity := "ns/service/svc-a"

	// Healthy → 不抑制
	if sm.ShouldSuppress(testCluster, entity) {
		t.Fatal("Healthy should not suppress")
	}

	// 设置为 Incident → 抑制
	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateIncident,
		IncidentID:   "inc-5",
	}
	sm.mu.Unlock()

	if !sm.ShouldSuppress(testCluster, entity) {
		t.Fatal("Incident should suppress")
	}
	if sm.GetActiveIncidentID(testCluster, entity) != "inc-5" {
		t.Fatal("should return active incident ID")
	}

	// Recovery → 也抑制
	sm.mu.Lock()
	sm.entries[testRef(entity)].CurrentState = aiops.StateRecovery
	sm.mu.Unlock()

	if !sm.ShouldSuppress(testCluster, entity) {
		t.Fatal("Recovery should suppress")
	}
}

// TestCheckRecoveryToStable 测试 Recovery → Stable（48h 无复发）
func TestCheckRecoveryToStable(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:         entity,
		CurrentState:      aiops.StateRecovery,
		IncidentID:        "inc-6",
		RecoveryStartedAt: time.Now().Add(-49 * time.Hour).Unix(), // 超过 48h
	}
	sm.mu.Unlock()

	sm.CheckRecoveryToStable(ctx)

	// entry 应该被删除（已稳定）
	if entry := sm.GetEntry(testCluster, entity); entry != nil {
		t.Fatalf("expected entry to be deleted, got state %s", entry.CurrentState)
	}
	if cb.stable != 1 {
		t.Fatalf("expected 1 stable callback, got %d", cb.stable)
	}
}

// TestCheckRecoveryToStable_NotYet 测试 Recovery 未满 48h 不转换
func TestCheckRecoveryToStable_NotYet(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:         entity,
		CurrentState:      aiops.StateRecovery,
		IncidentID:        "inc-7",
		RecoveryStartedAt: time.Now().Add(-24 * time.Hour).Unix(), // 只有 24h
	}
	sm.mu.Unlock()

	sm.CheckRecoveryToStable(ctx)

	entry := sm.GetEntry(testCluster, entity)
	if entry == nil || entry.CurrentState != aiops.StateRecovery {
		t.Fatal("should still be in Recovery (< 48h)")
	}
	if cb.stable != 0 {
		t.Fatal("should not callback stable")
	}
}

// TestFullLifecycle 完整生命周期: Healthy → Warning → Incident → Recovery → Stable
func TestFullLifecycle(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	// 1. Healthy → Warning (R > 0.2, > 2min)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.3), nil)
	entry := sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-3 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.3), nil)
	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateWarning {
		t.Fatalf("step 1: expected Warning, got %s", entry.CurrentState)
	}

	// 2. Warning → Incident (R > 0.5, > 5min)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.6), nil)
	entry = sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-6 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.6), nil)
	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateIncident {
		t.Fatalf("step 2: expected Incident, got %s", entry.CurrentState)
	}

	// 3. Incident → Recovery (R < 0.15, > 10min)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.1), nil)
	entry = sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-11 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.1), nil)
	entry = sm.GetEntry(testCluster, entity)
	if entry.CurrentState != aiops.StateRecovery {
		t.Fatalf("step 3: expected Recovery, got %s", entry.CurrentState)
	}

	// 4. Recovery → Stable (48h)
	entry.RecoveryStartedAt = time.Now().Add(-49 * time.Hour).Unix()
	sm.CheckRecoveryToStable(ctx)
	if sm.GetEntry(testCluster, entity) != nil {
		t.Fatal("step 4: expected entry deleted (stable)")
	}

	// 验证回调计数
	if cb.warningCreated != 1 {
		t.Fatalf("expected 1 warningCreated, got %d", cb.warningCreated)
	}
	if cb.stateEscalated != 1 {
		t.Fatalf("expected 1 stateEscalated, got %d", cb.stateEscalated)
	}
	if cb.recoveryStarted != 1 {
		t.Fatalf("expected 1 recoveryStarted, got %d", cb.recoveryStarted)
	}
	if cb.stable != 1 {
		t.Fatalf("expected 1 stable, got %d", cb.stable)
	}
}

// ==================== 新增测试: RestoreEntry + Warning→Healthy + 过期清理 ====================

// TestRestoreEntry 测试从外部注入状态机条目（启动恢复用）
func TestRestoreEntry(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)

	entry := &aiops.StateMachineEntry{
		ClusterID:       testCluster,
		EntityKey:       "ns/pod/pod-a",
		CurrentState:    aiops.StateWarning,
		IncidentID:      "inc-restore-1",
		LastEvaluatedAt: time.Now().Unix(),
	}

	sm.RestoreEntry(entry)

	got := sm.GetEntry(testCluster, "ns/pod/pod-a")
	if got == nil {
		t.Fatal("expected entry to exist after RestoreEntry")
	}
	if got.CurrentState != aiops.StateWarning {
		t.Fatalf("expected Warning, got %s", got.CurrentState)
	}
	if got.IncidentID != "inc-restore-1" {
		t.Fatalf("expected inc-restore-1, got %s", got.IncidentID)
	}
}

// TestWarningToHealthy 测试 Warning → Healthy 转换
// 条件: R_final < 0.15 持续 > 5 分钟 → 事件自动关闭
func TestWarningToHealthy(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateWarning,
		IncidentID:   "inc-w2h",
	}
	sm.mu.Unlock()

	risks := makeEntityRisks(entity, 0.1) // < 0.15

	// 第一次：条件满足，开始计时
	sm.Evaluate(ctx, "cluster-1", risks, nil)
	entry := sm.GetEntry(testCluster, entity)
	if entry == nil || entry.CurrentState != aiops.StateWarning {
		t.Fatal("should still be Warning (duration < 5min)")
	}

	// 模拟 6 分钟后
	entry.ConditionMetSince = time.Now().Add(-6 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", risks, nil)

	// entry 应该被删除（Warning → Healthy → 移除）
	if sm.GetEntry(testCluster, entity) != nil {
		t.Fatal("expected entry to be deleted after Warning→Healthy")
	}
	if cb.stable != 1 {
		t.Fatalf("expected 1 stable callback (OnStable), got %d", cb.stable)
	}
}

// TestWarningToHealthy_NoFalsePositive 测试 Warning 状态下风险在中间区域不误触发
// R_final 在 0.2~0.5 之间 → 保持 Warning（两个条件都不满足）
func TestWarningToHealthy_NoFalsePositive(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-a"

	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{
		EntityKey:    entity,
		CurrentState: aiops.StateWarning,
		IncidentID:   "inc-nofp",
	}
	sm.mu.Unlock()

	// R = 0.3 既不满足 <0.15（→Healthy）也不满足 >0.5（→Incident）
	risks := makeEntityRisks(entity, 0.3)

	sm.Evaluate(ctx, "cluster-1", risks, nil)

	entry := sm.GetEntry(testCluster, entity)
	if entry == nil || entry.CurrentState != aiops.StateWarning {
		t.Fatal("should stay Warning when 0.15 <= R <= 0.5")
	}
	if cb.stable != 0 {
		t.Fatal("should not trigger stable callback")
	}
	if cb.stateEscalated != 0 {
		t.Fatal("should not escalate to Incident")
	}
}

// TestCleanupStaleEntries 测试过期条目自动清理
func TestCleanupStaleEntries(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	// 注入一个 35 分钟前最后评估的条目
	sm.mu.Lock()
	sm.entries[testRef("ns/pod/stale-pod")] = &aiops.StateMachineEntry{
		EntityKey:       "ns/pod/stale-pod",
		CurrentState:    aiops.StateWarning,
		IncidentID:      "inc-stale",
		LastEvaluatedAt: time.Now().Add(-35 * time.Minute).Unix(),
	}
	// 注入一个刚刚评估的条目（不应被清理）
	sm.entries[testRef("ns/pod/fresh-pod")] = &aiops.StateMachineEntry{
		EntityKey:       "ns/pod/fresh-pod",
		CurrentState:    aiops.StateWarning,
		IncidentID:      "inc-fresh",
		LastEvaluatedAt: time.Now().Unix(),
	}
	// 注入一个 LastEvaluatedAt=0 的条目（不应被清理）
	sm.entries[testRef("ns/pod/zero-pod")] = &aiops.StateMachineEntry{
		EntityKey:       "ns/pod/zero-pod",
		CurrentState:    aiops.StateIncident,
		IncidentID:      "inc-zero",
		LastEvaluatedAt: 0,
	}
	sm.mu.Unlock()

	sm.CleanupStaleEntries(ctx, 30*time.Minute)

	// stale-pod 应被清理
	if sm.GetEntry(testCluster, "ns/pod/stale-pod") != nil {
		t.Fatal("stale entry should be cleaned up")
	}
	// fresh-pod 应保留
	if sm.GetEntry(testCluster, "ns/pod/fresh-pod") == nil {
		t.Fatal("fresh entry should NOT be cleaned up")
	}
	// zero-pod 应保留（LastEvaluatedAt=0 跳过）
	if sm.GetEntry(testCluster, "ns/pod/zero-pod") == nil {
		t.Fatal("zero-eval entry should NOT be cleaned up")
	}
	if cb.stable != 1 {
		t.Fatalf("expected 1 stable callback for stale entry, got %d", cb.stable)
	}
}

// TestWarningToHealthy_FullCycle 测试 Healthy → Warning → Healthy 完整回退周期
func TestWarningToHealthy_FullCycle(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	ctx := context.Background()

	entity := "ns/service/svc-b"

	// 1. Healthy → Warning (R > 0.2, > 2min)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.3), nil)
	entry := sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-3 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.3), nil)

	entry = sm.GetEntry(testCluster, entity)
	if entry == nil || entry.CurrentState != aiops.StateWarning {
		t.Fatalf("step 1: expected Warning, got %v", entry)
	}
	if cb.warningCreated != 1 {
		t.Fatalf("expected 1 warningCreated, got %d", cb.warningCreated)
	}

	// 2. Warning → Healthy (R < 0.15, > 5min)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)
	entry = sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-6 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)

	if sm.GetEntry(testCluster, entity) != nil {
		t.Fatal("step 2: expected entry deleted after Warning→Healthy")
	}
	if cb.stable != 1 {
		t.Fatalf("expected 1 stable callback, got %d", cb.stable)
	}

	// 3. 再次出现风险 → 应该重新创建（全新 Healthy → Warning 周期）
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.4), nil)
	entry = sm.GetEntry(testCluster, entity)
	entry.ConditionMetSince = time.Now().Add(-3 * time.Minute).Unix()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.4), nil)

	entry = sm.GetEntry(testCluster, entity)
	if entry == nil || entry.CurrentState != aiops.StateWarning {
		t.Fatalf("step 3: expected new Warning, got %v", entry)
	}
	if cb.warningCreated != 2 {
		t.Fatalf("expected 2 warningCreated, got %d", cb.warningCreated)
	}
}

// fakeClock 可手动拨动的时钟
type fakeClock struct{ current time.Time }

func (c *fakeClock) Now() time.Time          { return c.current }
func (c *fakeClock) Advance(d time.Duration) { c.current = c.current.Add(d) }

// 恢复观察期内实体仍在每 10 秒被评估一次；评估不得打断 48 小时的计时。
func TestRecoveryToStable_ReachedWhileEntityKeepsBeingEvaluated(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	clock := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	sm.now = clock.Now
	ctx := context.Background()
	entity := "ns/service/svc-a"

	// 事件中 → 风险回落满 10 分钟 → 进入恢复观察期
	sm.mu.Lock()
	sm.entries[testRef(entity)] = &aiops.StateMachineEntry{EntityKey: entity, CurrentState: aiops.StateIncident, IncidentID: "inc-1"}
	sm.mu.Unlock()
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)
	clock.Advance(11 * time.Minute)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)
	if got := sm.GetEntry(testCluster, entity).CurrentState; got != aiops.StateRecovery {
		t.Fatalf("前置条件：应进入 recovery，实际 %s", got)
	}

	// 47 小时：持续评估 + 定期检查，不应提前关闭
	for i := 0; i < 47; i++ {
		clock.Advance(time.Hour)
		sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)
		sm.CheckRecoveryToStable(ctx)
	}
	if cb.stable != 0 {
		t.Fatalf("未满 48 小时就关闭了")
	}

	// 再过 2 小时：满 48 小时，应关闭
	for i := 0; i < 2; i++ {
		clock.Advance(time.Hour)
		sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)
		sm.CheckRecoveryToStable(ctx)
	}
	if cb.stable != 1 {
		t.Fatalf("满 48 小时后应关闭 1 次，实际 %d 次", cb.stable)
	}
	// 关闭后实体仍在被评估，会重新得到一个健康的条目；不得还挂在 recovery 或带着旧事件号
	if entry := sm.GetEntry(testCluster, entity); entry != nil && (entry.CurrentState != aiops.StateHealthy || entry.IncidentID != "") {
		t.Fatalf("关闭后不应保留旧事件的条目，实际 %+v", entry)
	}
}

// 观察期内复发后再次恢复：48 小时从新的恢复时刻重新计。
func TestRecoveryToStable_RestartsAfterRecurrence(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	clock := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	sm.now = clock.Now
	ctx := context.Background()
	entity := "ns/service/svc-a"

	sm.RestoreEntry(&aiops.StateMachineEntry{
		ClusterID: testCluster, EntityKey: entity, CurrentState: aiops.StateRecovery, IncidentID: "inc-1",
		RecoveryStartedAt: clock.Now().Unix(),
	})

	clock.Advance(40 * time.Hour)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.9), nil) // 复发 → warning
	if got := sm.GetEntry(testCluster, entity).CurrentState; got != aiops.StateWarning {
		t.Fatalf("前置条件：应复发为 warning，实际 %s", got)
	}
	if got := sm.GetEntry(testCluster, entity).RecoveryStartedAt; got != 0 {
		t.Errorf("复发后恢复时刻应清空，实际 %d", got)
	}
}

// 重启后恢复的条目带着原来的恢复时刻：不从重启那一刻重新计 48 小时。
func TestRecoveryToStable_RestoredEntryKeepsOriginalStart(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	clock := &fakeClock{current: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)}
	sm.now = clock.Now
	ctx := context.Background()
	entity := "ns/service/svc-a"

	sm.RestoreEntry(&aiops.StateMachineEntry{
		ClusterID: testCluster, EntityKey: entity, CurrentState: aiops.StateRecovery, IncidentID: "inc-1",
		RecoveryStartedAt: clock.Now().Add(-47 * time.Hour).Unix(),
	})

	clock.Advance(2 * time.Hour)
	sm.Evaluate(ctx, "cluster-1", makeEntityRisks(entity, 0.05), nil)
	sm.CheckRecoveryToStable(ctx)
	if cb.stable != 1 {
		t.Fatalf("恢复时刻在 49 小时前，应关闭，实际关闭 %d 次", cb.stable)
	}
}

// 两个集群里的同名实体各走各的状态转换、各自创建事件；一个关闭不影响另一个。
func TestEvaluate_SameEntityKeyInTwoClusters_IndependentEntries(t *testing.T) {
	cb := &mockCallback{}
	sm := NewStateMachine(cb)
	clock := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	sm.now = clock.Now
	ctx := context.Background()
	entity := "default/pod/nginx"

	// 集群 A 的实体持续高风险 → Warning；集群 B 的同名实体一直健康
	sm.Evaluate(ctx, "cluster-a", makeEntityRisks(entity, 0.4), nil)
	sm.Evaluate(ctx, "cluster-b", makeEntityRisks(entity, 0.0), nil)
	clock.Advance(3 * time.Minute)
	sm.Evaluate(ctx, "cluster-a", makeEntityRisks(entity, 0.4), nil)
	sm.Evaluate(ctx, "cluster-b", makeEntityRisks(entity, 0.0), nil)

	entryA := sm.GetEntry("cluster-a", entity)
	entryB := sm.GetEntry("cluster-b", entity)
	if entryA == nil || entryA.CurrentState != aiops.StateWarning {
		t.Fatalf("集群 A 的实体应为 Warning，实际 %+v", entryA)
	}
	if entryB == nil || entryB.CurrentState != aiops.StateHealthy || entryB.IncidentID != "" {
		t.Errorf("集群 B 的同名实体应保持健康且没有事件，实际 %+v", entryB)
	}
	if entryA.ClusterID != "cluster-a" || entryB.ClusterID != "cluster-b" {
		t.Errorf("条目没有带上所属集群: A=%q B=%q", entryA.ClusterID, entryB.ClusterID)
	}
	if cb.warningCreated != 1 {
		t.Errorf("期望只创建 1 个事件，实际 %d 个", cb.warningCreated)
	}
}

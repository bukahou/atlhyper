// atlhyper_master_v2/aiops/statemachine/machine_test.go
// 状态机测试：以主体为键的状态转换、观察期、复发、峰值严重度、发生次数、失联关闭
package statemachine

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// recordingHandler 记下状态机发出的每一次事件变化
type recordingHandler struct {
	transitions []aiops.IncidentTransition
	risks       []*aiops.SubjectRisk
}

func (h *recordingHandler) OnTransition(_ context.Context, transition aiops.IncidentTransition, risk *aiops.SubjectRisk) {
	h.transitions = append(h.transitions, transition)
	h.risks = append(h.risks, risk)
}

func (h *recordingHandler) kinds() []aiops.TransitionKind {
	kinds := make([]aiops.TransitionKind, len(h.transitions))
	for i, tr := range h.transitions {
		kinds[i] = tr.Kind
	}
	return kinds
}

func (h *recordingHandler) count(kind aiops.TransitionKind) int {
	n := 0
	for _, tr := range h.transitions {
		if tr.Kind == kind {
			n++
		}
	}
	return n
}

func (h *recordingHandler) last(t *testing.T) aiops.IncidentTransition {
	t.Helper()
	if len(h.transitions) == 0 {
		t.Fatal("还没有任何事件变化")
	}
	return h.transitions[len(h.transitions)-1]
}

// fakeClock 可手动拨动的时钟
type fakeClock struct{ current time.Time }

func (c *fakeClock) Now() time.Time          { return c.current }
func (c *fakeClock) Advance(d time.Duration) { c.current = c.current.Add(d) }
func (c *fakeClock) Set(t time.Time)         { c.current = t }

const testCluster = "cluster-1"

var testEpoch = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// fixture 一台带假时钟、顺序事件号的状态机
type fixture struct {
	t       *testing.T
	sm      *StateMachine
	handler *recordingHandler
	clock   *fakeClock
	ctx     context.Context
}

func newFixture(t *testing.T) *fixture {
	handler := &recordingHandler{}
	clock := &fakeClock{current: testEpoch}
	issued := 0
	sm := NewStateMachine(handler, func(time.Time) string {
		issued++
		return fmt.Sprintf("inc-%d", issued)
	})
	sm.now = clock.Now
	return &fixture{t: t, sm: sm, handler: handler, clock: clock, ctx: context.Background()}
}

// member 一个成员实体的风险
func member(entityKey string, rFinal float64) *aiops.EntityRisk {
	return &aiops.EntityRisk{EntityKey: entityKey, RFinal: rFinal}
}

// subjectRisk 由成员拼出主体风险：领头是风险最高的，成员只含有风险的、从高到低
func subjectRisk(subjectKey string, members ...*aiops.EntityRisk) *aiops.SubjectRisk {
	sorted := append([]*aiops.EntityRisk(nil), members...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].RFinal > sorted[j].RFinal })
	sr := &aiops.SubjectRisk{SubjectKey: subjectKey, Lead: sorted[0], RFinal: sorted[0].RFinal}
	for _, m := range sorted {
		if m.RFinal > 0 {
			sr.Members = append(sr.Members, m)
		}
	}
	return sr
}

// selfRisk 主体就是实体自身（service、node 等）
func selfRisk(key string, rFinal float64) *aiops.SubjectRisk {
	return subjectRisk(key, member(key, rFinal))
}

func (f *fixture) evaluate(risks ...*aiops.SubjectRisk) {
	f.evaluateIn(testCluster, nil, risks...)
}

func (f *fixture) evaluateIn(clusterID string, clusterRisk *aiops.ClusterRisk, risks ...*aiops.SubjectRisk) {
	bySubject := make(map[string]*aiops.SubjectRisk, len(risks))
	for _, r := range risks {
		bySubject[r.SubjectKey] = r
	}
	f.sm.Evaluate(f.ctx, clusterID, bySubject, clusterRisk)
}

// hold 同一份风险持续 d：先评估一次，拨钟，再评估一次
func (f *fixture) hold(d time.Duration, risks ...*aiops.SubjectRisk) {
	f.evaluate(risks...)
	f.clock.Advance(d)
	f.evaluate(risks...)
}

func (f *fixture) state(subjectKey string) aiops.EntityState {
	f.t.Helper()
	entry := f.sm.GetEntry(testCluster, subjectKey)
	if entry == nil {
		f.t.Fatalf("主体 %s 没有状态机条目", subjectKey)
	}
	return entry.CurrentState
}

func (f *fixture) wantKinds(want ...aiops.TransitionKind) {
	f.t.Helper()
	got := f.handler.kinds()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		f.t.Fatalf("事件变化\n got  %v\n want %v", got, want)
	}
}

const (
	api = "shop/deployment/api"
	svc = "shop/service/api"
)

// ==================== 建事件 ====================

// 风险 > 0.2 持续 2 分钟才建事件；发出的变化字段齐全
func TestHealthyToWarning_CreatesIncidentAfterTwoMinutes(t *testing.T) {
	f := newFixture(t)
	risk := subjectRisk(api, member("shop/pod/api-a", 0.3), member("shop/pod/api-b", 0.45), member("shop/pod/api-c", 0))

	f.evaluate(risk)
	f.clock.Advance(119 * time.Second)
	f.evaluate(risk)
	if f.state(api) != aiops.StateHealthy {
		t.Fatalf("不满 2 分钟不应建事件，实际 %s", f.state(api))
	}
	f.wantKinds()

	f.clock.Advance(time.Second)
	f.evaluate(risk)

	f.wantKinds(aiops.TransitionCreated)
	want := aiops.IncidentTransition{
		Kind:             aiops.TransitionCreated,
		IncidentID:       "inc-1",
		ClusterID:        testCluster,
		SubjectKey:       api,
		RootCause:        "shop/pod/api-b",
		Members:          []string{"shop/pod/api-b", "shop/pod/api-a"},
		FromState:        aiops.StateHealthy,
		ToState:          aiops.StateWarning,
		Severity:         "low",
		RFinal:           0.45,
		PeakRisk:         0.45,
		Recurrence:       0,
		Occurrences24h:   1,
		ReachedIncident:  false,
		EpisodeStartedAt: testEpoch, // 异常起点 = 条件开始满足的时刻（比开事件早 2 分钟）
		At:               testEpoch.Add(2 * time.Minute),
	}
	if got := f.handler.last(t); !reflect.DeepEqual(got, want) {
		t.Errorf("建事件的变化\n got  %+v\n want %+v", got, want)
	}
	if f.handler.risks[0] != risk {
		t.Error("处理方应拿到触发本次变化的主体风险")
	}
	if entry := f.sm.GetEntry(testCluster, api); entry.IncidentID != "inc-1" || entry.CurrentState != aiops.StateWarning {
		t.Errorf("条目应进入警告并记下事件号，实际 %+v", entry)
	}
}

// 同一工作负载的两个 Pod 同时出问题：只建一个事件
func TestSameWorkloadTwoPods_OneIncident(t *testing.T) {
	f := newFixture(t)
	risk := subjectRisk(api, member("shop/pod/api-a", 0.6), member("shop/pod/api-b", 0.6))

	f.hold(3*time.Minute, risk)

	if n := f.handler.count(aiops.TransitionCreated); n != 1 {
		t.Fatalf("期望只建 1 个事件，实际 %d 个", n)
	}
	if got := f.handler.last(t).Members; len(got) != 2 {
		t.Errorf("事件应带上两个成员，实际 %v", got)
	}
}

// 风险中途回落：计时重来
func TestHealthyToWarning_ConditionResets(t *testing.T) {
	f := newFixture(t)

	f.hold(90*time.Second, selfRisk(svc, 0.3))
	f.evaluate(selfRisk(svc, 0.1)) // 回落
	f.clock.Advance(90 * time.Second)
	f.evaluate(selfRisk(svc, 0.3)) // 重新开始计时
	f.clock.Advance(90 * time.Second)
	f.evaluate(selfRisk(svc, 0.3))

	if f.state(svc) != aiops.StateHealthy {
		t.Fatalf("累计超过 2 分钟但中途断过，不应建事件")
	}
	f.clock.Advance(30 * time.Second)
	f.evaluate(selfRisk(svc, 0.3))
	if f.state(svc) != aiops.StateWarning {
		t.Fatalf("连续满 2 分钟应建事件，实际 %s", f.state(svc))
	}
}

// 两个集群里的同名主体各走各的
func TestSameSubjectKeyInTwoClusters_IndependentEntries(t *testing.T) {
	f := newFixture(t)

	f.evaluateIn("cluster-a", nil, selfRisk(svc, 0.4))
	f.evaluateIn("cluster-b", nil, selfRisk(svc, 0))
	f.clock.Advance(3 * time.Minute)
	f.evaluateIn("cluster-a", nil, selfRisk(svc, 0.4))
	f.evaluateIn("cluster-b", nil, selfRisk(svc, 0))

	entryA, entryB := f.sm.GetEntry("cluster-a", svc), f.sm.GetEntry("cluster-b", svc)
	if entryA == nil || entryA.CurrentState != aiops.StateWarning || entryA.ClusterID != "cluster-a" {
		t.Fatalf("集群 A 的主体应为警告，实际 %+v", entryA)
	}
	if entryB == nil || entryB.CurrentState != aiops.StateHealthy || entryB.IncidentID != "" {
		t.Errorf("集群 B 的同名主体应保持健康且没有事件，实际 %+v", entryB)
	}
	f.wantKinds(aiops.TransitionCreated)
	if got := f.handler.last(t).ClusterID; got != "cluster-a" {
		t.Errorf("变化应属于集群 A，实际 %s", got)
	}
}

// ==================== 升级 ====================

func (f *fixture) openWarning(subjectKey string, rFinal float64) {
	f.t.Helper()
	f.hold(2*time.Minute, selfRisk(subjectKey, rFinal))
	if f.state(subjectKey) != aiops.StateWarning {
		f.t.Fatalf("前置条件：%s 应进入警告，实际 %s", subjectKey, f.state(subjectKey))
	}
}

func (f *fixture) openIncident(subjectKey string, rFinal float64) {
	f.t.Helper()
	f.openWarning(subjectKey, rFinal)
	f.hold(5*time.Minute, selfRisk(subjectKey, rFinal))
	if f.state(subjectKey) != aiops.StateIncident {
		f.t.Fatalf("前置条件：%s 应升级为事件中，实际 %s", subjectKey, f.state(subjectKey))
	}
}

// recover 风险回落并持续 d，进入恢复观察
func (f *fixture) recover(subjectKey string, d time.Duration) {
	f.t.Helper()
	f.hold(d, selfRisk(subjectKey, 0.05))
	if f.state(subjectKey) != aiops.StateRecovery {
		f.t.Fatalf("前置条件：%s 应进入恢复观察，实际 %s", subjectKey, f.state(subjectKey))
	}
}

// 风险 > 0.5 持续 5 分钟升级为事件中
func TestWarningToIncident_AfterFiveMinutes(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.6)

	f.hold(5*time.Minute-time.Second, selfRisk(svc, 0.6))
	if f.state(svc) != aiops.StateWarning {
		t.Fatalf("不满 5 分钟不应升级")
	}
	f.clock.Advance(time.Second)
	f.evaluate(selfRisk(svc, 0.6))

	f.wantKinds(aiops.TransitionCreated, aiops.TransitionEscalated)
	got := f.handler.last(t)
	if got.FromState != aiops.StateWarning || got.ToState != aiops.StateIncident || !got.ReachedIncident {
		t.Errorf("升级的变化不对: %+v", got)
	}
	if got.IncidentID != "inc-1" || got.Severity != "medium" {
		t.Errorf("升级沿用同一事件号、严重度为峰值，实际 %s / %s", got.IncidentID, got.Severity)
	}
	if entry := f.sm.GetEntry(testCluster, svc); !entry.ReachedIncident {
		t.Error("条目应记下升级过")
	}
}

// 主体风险不到 0.5，但集群风险 > 80 也会升级
func TestWarningToIncident_ClusterRiskTrigger(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	critical := &aiops.ClusterRisk{Risk: 85}

	f.evaluateIn(testCluster, critical, selfRisk(svc, 0.3))
	f.clock.Advance(5 * time.Minute)
	f.evaluateIn(testCluster, critical, selfRisk(svc, 0.3))

	if f.state(svc) != aiops.StateIncident {
		t.Fatalf("集群风险 > 80 应升级，实际 %s", f.state(svc))
	}
}

// 警告期间风险在 0.15 ~ 0.5 之间：维持警告
func TestWarning_StaysInMiddleBand(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)

	f.hold(time.Hour, selfRisk(svc, 0.3))

	if f.state(svc) != aiops.StateWarning {
		t.Fatalf("应维持警告，实际 %s", f.state(svc))
	}
	f.wantKinds(aiops.TransitionCreated)
}

// 两个出口条件各计各的时：先低后高，不能把「低」攒下的时间算给「高」
func TestWarning_ConditionTimingNotSharedAcrossTargets(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)

	f.hold(4*time.Minute, selfRisk(svc, 0.05)) // 朝恢复方向计了 4 分钟
	f.clock.Advance(90 * time.Second)
	f.evaluate(selfRisk(svc, 0.6)) // 突然转高

	if f.state(svc) != aiops.StateWarning {
		t.Fatalf("风险刚转高不应立即升级，实际 %s", f.state(svc))
	}
	f.clock.Advance(5 * time.Minute)
	f.evaluate(selfRisk(svc, 0.6))
	if f.state(svc) != aiops.StateIncident {
		t.Fatalf("转高后满 5 分钟应升级，实际 %s", f.state(svc))
	}
}

// ==================== 恢复观察 ====================

// 只到警告的事件，风险回落 5 分钟后进入恢复观察，而不是直接关闭
func TestWarningToRecovery_NotClosedDirectly(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)

	f.hold(5*time.Minute, selfRisk(svc, 0.05))

	f.wantKinds(aiops.TransitionCreated, aiops.TransitionRecoveryStarted)
	entry := f.sm.GetEntry(testCluster, svc)
	if entry == nil || entry.CurrentState != aiops.StateRecovery || entry.IncidentID != "inc-1" {
		t.Fatalf("应进入恢复观察并保留事件，实际 %+v", entry)
	}
	got := f.handler.last(t)
	if got.FromState != aiops.StateWarning || got.ToState != aiops.StateRecovery || got.ReachedIncident {
		t.Errorf("恢复的变化不对: %+v", got)
	}
	if got.RootCause != svc {
		t.Errorf("恢复时仍应带着领头实体，实际 %q", got.RootCause)
	}
}

// 事件中：风险回落需持续 10 分钟
func TestIncidentToRecovery_AfterTenMinutes(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)

	f.hold(9*time.Minute, selfRisk(svc, 0.05))
	if f.state(svc) != aiops.StateIncident {
		t.Fatalf("不满 10 分钟不应进入恢复观察")
	}
	f.clock.Advance(time.Minute)
	f.evaluate(selfRisk(svc, 0.05))

	if f.state(svc) != aiops.StateRecovery {
		t.Fatalf("应进入恢复观察，实际 %s", f.state(svc))
	}
	if got := f.handler.last(t); got.Kind != aiops.TransitionRecoveryStarted || !got.ReachedIncident || got.Severity != "medium" {
		t.Errorf("恢复的变化应带着升级过与峰值严重度: %+v", got)
	}
}

// ==================== 复发 ====================

// 升级过的事件：观察期内风险一回到 0.2 以上立即复发
func TestRecurrence_EscalatedIncidentImmediate(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.recover(svc, 10*time.Minute)

	f.clock.Advance(20 * time.Hour)
	f.evaluate(selfRisk(svc, 0.3))

	if f.state(svc) != aiops.StateWarning {
		t.Fatalf("应立即复发为警告，实际 %s", f.state(svc))
	}
	got := f.handler.last(t)
	if got.Kind != aiops.TransitionRecurred || got.IncidentID != "inc-1" || got.Recurrence != 1 {
		t.Errorf("复发应并入同一事件、次数为 1: %+v", got)
	}
	if got.Occurrences24h != 2 || !got.ReachedIncident {
		t.Errorf("近 24 小时发生 2 次（建事件 + 本次复发）、升级过: %+v", got)
	}
	if entry := f.sm.GetEntry(testCluster, svc); entry.RecoveryStartedAt != 0 {
		t.Errorf("复发后恢复时刻应清空，实际 %d", entry.RecoveryStartedAt)
	}
}

// 没升级过的事件：观察期内再触发需持续 2 分钟，并且是复发而不是新事件
func TestRecurrence_NeverEscalatedNeedsTwoMinutes(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	f.recover(svc, 5*time.Minute)

	f.clock.Advance(3 * time.Hour) // 6 小时观察期内
	f.evaluate(selfRisk(svc, 0.3))
	f.clock.Advance(119 * time.Second)
	f.evaluate(selfRisk(svc, 0.3))
	if f.state(svc) != aiops.StateRecovery {
		t.Fatalf("不满 2 分钟不应复发，实际 %s", f.state(svc))
	}

	f.clock.Advance(time.Second)
	f.evaluate(selfRisk(svc, 0.3))

	f.wantKinds(aiops.TransitionCreated, aiops.TransitionRecoveryStarted, aiops.TransitionRecurred)
	got := f.handler.last(t)
	if got.IncidentID != "inc-1" || got.Recurrence != 1 || got.Occurrences24h != 2 || got.ReachedIncident {
		t.Errorf("应并入同一事件：次数 1、近 24 小时 2 次、没升级过: %+v", got)
	}
}

// 没升级过的事件：观察期内一次瞬时抖动不算复发
func TestRecurrence_NeverEscalatedBlipIgnored(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	f.recover(svc, 5*time.Minute)

	f.clock.Advance(time.Hour)
	f.evaluate(selfRisk(svc, 0.3))
	f.clock.Advance(time.Minute)
	f.evaluate(selfRisk(svc, 0.05))
	f.clock.Advance(5 * time.Minute)
	f.evaluate(selfRisk(svc, 0.3))

	if f.state(svc) != aiops.StateRecovery || f.handler.count(aiops.TransitionRecurred) != 0 {
		t.Fatalf("断开的两次抖动不应算复发，实际状态 %s", f.state(svc))
	}
}

// 近 24 小时发生次数：超过 24 小时的滑出
func TestOccurrences24h_SlidesOut(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6) // 建事件：第 0 小时
	f.recover(svc, 10*time.Minute)

	f.clock.Set(testEpoch.Add(10 * time.Hour))
	f.evaluate(selfRisk(svc, 0.3)) // 第 10 小时复发
	if got := f.handler.last(t); got.Kind != aiops.TransitionRecurred || got.Occurrences24h != 2 {
		t.Fatalf("第 10 小时复发：近 24 小时应为 2 次，实际 %+v", got)
	}
	f.recover(svc, 5*time.Minute)

	f.clock.Set(testEpoch.Add(30 * time.Hour))
	f.evaluate(selfRisk(svc, 0.3)) // 第 30 小时复发：建事件那次已滑出

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionRecurred || got.Recurrence != 2 {
		t.Fatalf("应是第 2 次复发，实际 %+v", got)
	}
	if got.Occurrences24h != 2 {
		t.Errorf("近 24 小时应为 2 次（第 10、30 小时），实际 %d", got.Occurrences24h)
	}
	if entry := f.sm.GetEntry(testCluster, svc); len(entry.OccurrenceTimes) != 2 {
		t.Errorf("条目里滑出的时刻应清掉，实际 %v", entry.OccurrenceTimes)
	}
}

// ==================== 观察期满关闭 ====================

func TestObservationPeriod_NeverEscalatedClosesAfterSixHours(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	f.recover(svc, 5*time.Minute)

	f.clock.Advance(6*time.Hour - time.Minute)
	f.evaluate(selfRisk(svc, 0.05))
	f.sm.CheckRecoveryToStable(f.ctx)
	if f.handler.count(aiops.TransitionClosed) != 0 {
		t.Fatal("不满 6 小时不应关闭")
	}

	f.clock.Advance(time.Minute)
	f.evaluate(selfRisk(svc, 0.05))
	f.sm.CheckRecoveryToStable(f.ctx)

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionClosed || got.FromState != aiops.StateRecovery || got.ToState != aiops.StateStable {
		t.Fatalf("满 6 小时应关闭，实际 %+v", got)
	}
	if got.IncidentID != "inc-1" || got.Severity != "low" || got.PeakRisk != 0.3 || got.RootCause != svc {
		t.Errorf("关闭的变化应带着事件号、峰值严重度、峰值风险、领头实体: %+v", got)
	}
	if f.handler.risks[len(f.handler.risks)-1] != nil {
		t.Error("观察期满关闭没有对应的评估，主体风险应为 nil")
	}
	if entry := f.sm.GetEntry(testCluster, svc); entry != nil {
		t.Errorf("关闭后条目应删除，实际 %+v", entry)
	}
}

// 升级过的事件观察 48 小时；期间主体仍在被评估，评估不得打断计时
func TestObservationPeriod_EscalatedClosesAfterFortyEightHours(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.recover(svc, 10*time.Minute)

	for i := 0; i < 47; i++ {
		f.clock.Advance(time.Hour)
		f.evaluate(selfRisk(svc, 0.05))
		f.sm.CheckRecoveryToStable(f.ctx)
	}
	if f.handler.count(aiops.TransitionClosed) != 0 {
		t.Fatal("升级过的事件不满 48 小时就关闭了")
	}

	f.clock.Advance(time.Hour)
	f.evaluate(selfRisk(svc, 0.05))
	f.sm.CheckRecoveryToStable(f.ctx)

	if f.handler.count(aiops.TransitionClosed) != 1 {
		t.Fatalf("满 48 小时应关闭 1 次，实际 %d 次", f.handler.count(aiops.TransitionClosed))
	}
	if got := f.handler.last(t); !got.ReachedIncident || got.Severity != "medium" {
		t.Errorf("关闭的变化应带着升级过与峰值严重度: %+v", got)
	}
}

// 关闭之后再出问题：全新的事件
func TestAfterClose_NewIncident(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	f.recover(svc, 5*time.Minute)
	f.clock.Advance(6 * time.Hour)
	f.sm.CheckRecoveryToStable(f.ctx)

	f.openWarning(svc, 0.3)

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionCreated || got.IncidentID != "inc-2" || got.Recurrence != 0 || got.Occurrences24h != 1 {
		t.Errorf("关闭后应建新事件、计数从头开始: %+v", got)
	}
}

// 观察期内复发后再次恢复：观察期从新的恢复时刻重新计
func TestObservationPeriod_RestartsAfterRecurrence(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	f.recover(svc, 5*time.Minute)

	f.clock.Advance(5 * time.Hour)
	f.hold(2*time.Minute, selfRisk(svc, 0.3)) // 复发
	f.recover(svc, 5*time.Minute)

	f.clock.Advance(5 * time.Hour)
	f.sm.CheckRecoveryToStable(f.ctx)
	if f.handler.count(aiops.TransitionClosed) != 0 {
		t.Fatal("复发后观察期应重新计，5 小时不应关闭")
	}
	f.clock.Advance(time.Hour)
	f.sm.CheckRecoveryToStable(f.ctx)
	if f.handler.count(aiops.TransitionClosed) != 1 {
		t.Fatal("复发后重新满 6 小时应关闭")
	}
}

// 重启后恢复的条目带着原来的恢复时刻与「升级过」：不从重启那一刻重新计
func TestObservationPeriod_RestoredEntryKeepsOriginalStart(t *testing.T) {
	f := newFixture(t)
	f.sm.RestoreEntry(&aiops.StateMachineEntry{
		ClusterID: testCluster, SubjectKey: svc, CurrentState: aiops.StateRecovery, IncidentID: "inc-9",
		ReachedIncident: true, Severity: "high", PeakRisk: 0.75, Recurrence: 1, RootCause: svc,
		RecoveryStartedAt: f.clock.Now().Add(-47 * time.Hour).Unix(),
	})

	f.clock.Advance(2 * time.Hour)
	f.evaluate(selfRisk(svc, 0.05))
	f.sm.CheckRecoveryToStable(f.ctx)

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionClosed || got.IncidentID != "inc-9" {
		t.Fatalf("恢复时刻在 49 小时前，应关闭，实际 %+v", got)
	}
	if got.Severity != "high" || got.PeakRisk != 0.75 || got.Recurrence != 1 {
		t.Errorf("关闭的变化应带着从库里取回的严重度、峰值、复发次数: %+v", got)
	}
}

// 重启后恢复的条目：发生时刻从库里取回，复发时接着数
func TestRestoredEntry_OccurrenceTimesContinue(t *testing.T) {
	f := newFixture(t)
	f.sm.RestoreEntry(&aiops.StateMachineEntry{
		ClusterID: testCluster, SubjectKey: svc, CurrentState: aiops.StateRecovery, IncidentID: "inc-9",
		Severity: "low", PeakRisk: 0.3, Recurrence: 1, RootCause: svc,
		RecoveryStartedAt: f.clock.Now().Unix(),
		OccurrenceTimes:   []int64{f.clock.Now().Add(-30 * time.Hour).Unix(), f.clock.Now().Add(-2 * time.Hour).Unix()},
	})

	f.hold(2*time.Minute, selfRisk(svc, 0.3))

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionRecurred || got.Recurrence != 2 || got.Occurrences24h != 2 {
		t.Errorf("应是第 2 次复发、近 24 小时 2 次（30 小时前那次不算）: %+v", got)
	}
}

// ==================== 严重度（峰值） ====================

// 严重度只升不降：风险升档时发 severity_raised，回落不降档
func TestSeverity_OnlyRises(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3) // low

	f.evaluate(selfRisk(svc, 0.75)) // 落入 high
	got := f.handler.last(t)
	if got.Kind != aiops.TransitionSeverityRaised || got.Severity != "high" || got.PeakRisk != 0.75 {
		t.Fatalf("风险升档应发 severity_raised（high）: %+v", got)
	}
	if got.FromState != aiops.StateWarning || got.ToState != aiops.StateWarning {
		t.Errorf("严重度升高不改变状态: %+v", got)
	}

	f.evaluate(selfRisk(svc, 0.75)) // 同一档：不重复发
	f.evaluate(selfRisk(svc, 0.3))  // 回落：不降档
	f.evaluate(selfRisk(svc, 0.55)) // medium，低于已记录的 high：不发
	f.wantKinds(aiops.TransitionCreated, aiops.TransitionSeverityRaised)

	f.evaluate(selfRisk(svc, 0.95)) // 再升到 critical
	got = f.handler.last(t)
	if got.Kind != aiops.TransitionSeverityRaised || got.Severity != "critical" {
		t.Fatalf("应再升到 critical: %+v", got)
	}
	if entry := f.sm.GetEntry(testCluster, svc); entry.Severity != "critical" || entry.PeakRisk != 0.95 {
		t.Errorf("条目应记下峰值严重度与峰值风险，实际 %s / %v", entry.Severity, entry.PeakRisk)
	}
}

// 事件中同样会升档；之后恢复、关闭都带着峰值严重度
func TestSeverity_RaisedDuringIncidentAndKeptAfterwards(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6) // medium

	f.evaluate(selfRisk(svc, 0.8))
	if got := f.handler.last(t); got.Kind != aiops.TransitionSeverityRaised || got.Severity != "high" || got.ToState != aiops.StateIncident {
		t.Fatalf("事件中风险升档应发 severity_raised: %+v", got)
	}

	f.recover(svc, 10*time.Minute)
	if got := f.handler.last(t); got.Severity != "high" || got.PeakRisk != 0.8 {
		t.Errorf("恢复时严重度与峰值风险保持峰值: %+v", got)
	}
}

// 状态转换与升档同时发生：只发一条状态转换，严重度并在里面
func TestSeverity_FoldedIntoStateTransition(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3) // low

	// 集群风险触发升级的那一轮，主体风险同时落入 medium
	critical := &aiops.ClusterRisk{Risk: 85}
	f.evaluateIn(testCluster, critical, selfRisk(svc, 0.45))
	f.clock.Advance(5 * time.Minute)
	f.evaluateIn(testCluster, critical, selfRisk(svc, 0.55))

	f.wantKinds(aiops.TransitionCreated, aiops.TransitionEscalated)
	if got := f.handler.last(t); got.Severity != "medium" {
		t.Errorf("升级那一条应带上新的峰值严重度 medium: %+v", got)
	}
}

// 恢复观察期与健康状态下不发 severity_raised
func TestSeverity_NotRaisedOutsideWarningAndIncident(t *testing.T) {
	f := newFixture(t)
	f.evaluate(selfRisk(svc, 0.95)) // 健康、还没建事件
	f.wantKinds()

	f.openWarning(svc, 0.3)
	f.recover(svc, 5*time.Minute)
	f.evaluate(selfRisk(svc, 0.95)) // 观察期内的高风险：没升级过，要等 2 分钟才算复发

	if f.handler.count(aiops.TransitionSeverityRaised) != 0 {
		t.Fatalf("观察期内不应发 severity_raised: %v", f.handler.kinds())
	}

	f.clock.Advance(2 * time.Minute)
	f.evaluate(selfRisk(svc, 0.95))
	if got := f.handler.last(t); got.Kind != aiops.TransitionRecurred || got.Severity != "critical" {
		t.Errorf("复发那一条应带上新的峰值严重度: %+v", got)
	}
}

// 峰值风险取事件至今的最高值，包括没有引起升档的那些评估
func TestPeakRisk_TracksHighestSeen(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.3)
	f.evaluate(selfRisk(svc, 0.48)) // 仍是 low，不发变化
	f.wantKinds(aiops.TransitionCreated)

	f.recover(svc, 5*time.Minute)

	if got := f.handler.last(t); got.PeakRisk != 0.48 || got.RFinal != 0.05 {
		t.Errorf("恢复时峰值应为 0.48、当前风险 0.05: %+v", got)
	}
}

// 升级时领头实体换了：变化带上当前的领头
func TestRootCause_FollowsCurrentLead(t *testing.T) {
	f := newFixture(t)
	f.hold(2*time.Minute, subjectRisk(api, member("shop/pod/api-old", 0.6)))
	if got := f.handler.last(t).RootCause; got != "shop/pod/api-old" {
		t.Fatalf("建事件时的领头应是 api-old，实际 %s", got)
	}

	// 滚动更新后 Pod 换了名字
	f.hold(5*time.Minute, subjectRisk(api, member("shop/pod/api-new", 0.6)))

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionEscalated || got.RootCause != "shop/pod/api-new" {
		t.Errorf("升级时领头应换成 api-new: %+v", got)
	}
	if entry := f.sm.GetEntry(testCluster, api); entry.RootCause != "shop/pod/api-new" {
		t.Errorf("条目应记下当前领头，实际 %s", entry.RootCause)
	}
}

// ==================== 失联关闭 ====================

func TestCleanupStaleEntries_ClosesAsStale(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.evaluate(selfRisk("shop/pod/healthy", 0)) // 一直健康、没有事件的主体

	// 35 分钟里本集群没有任何评估（Agent 断连）；只有另一个集群还在上报。
	// 本集群还在上报、只是主体不在了的情况是「对象已删除」，见 subject_gone_test.go
	f.clock.Advance(35 * time.Minute)
	f.evaluateIn("cluster-2", nil, selfRisk("shop/service/fresh", 0.3))

	f.sm.CleanupStaleEntries(f.ctx, 30*time.Minute)

	if f.sm.GetEntry(testCluster, svc) != nil || f.sm.GetEntry(testCluster, "shop/pod/healthy") != nil {
		t.Fatal("失联的条目应清掉")
	}
	if f.sm.GetEntry("cluster-2", "shop/service/fresh") == nil {
		t.Fatal("仍在被评估的条目不应清掉")
	}
	if n := f.handler.count(aiops.TransitionClosedStale); n != 1 {
		t.Fatalf("只有带事件的条目发失联关闭，期望 1 条，实际 %d 条", n)
	}
	got := f.handler.last(t)
	if got.Kind != aiops.TransitionClosedStale || got.IncidentID != "inc-1" {
		t.Fatalf("应发失联关闭: %+v", got)
	}
	if got.FromState != aiops.StateIncident || got.ToState != aiops.StateStable || got.Severity != "medium" || !got.ReachedIncident {
		t.Errorf("失联关闭应带着关闭前的状态与峰值严重度: %+v", got)
	}
	if f.handler.count(aiops.TransitionClosed) != 0 {
		t.Error("失联关闭不得当作观察期满关闭")
	}
}

// 从库里恢复、还没被评估过的条目（最近评估时刻为 0）不清理
func TestCleanupStaleEntries_SkipsNeverEvaluated(t *testing.T) {
	f := newFixture(t)
	f.sm.RestoreEntry(&aiops.StateMachineEntry{
		ClusterID: testCluster, SubjectKey: svc, CurrentState: aiops.StateIncident, IncidentID: "inc-9",
	})

	f.clock.Advance(time.Hour)
	f.sm.CleanupStaleEntries(f.ctx, 30*time.Minute)

	if f.sm.GetEntry(testCluster, svc) == nil {
		t.Fatal("没被评估过的条目不应清理")
	}
	f.wantKinds()
}

// ==================== 完整生命周期 ====================

func TestFullLifecycle(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.recover(svc, 10*time.Minute)
	f.clock.Advance(time.Hour)
	f.evaluate(selfRisk(svc, 0.75)) // 复发，同时落入 high
	f.recover(svc, 5*time.Minute)
	f.clock.Advance(48 * time.Hour)
	f.sm.CheckRecoveryToStable(f.ctx)

	f.wantKinds(
		aiops.TransitionCreated,
		aiops.TransitionEscalated,
		aiops.TransitionRecoveryStarted,
		aiops.TransitionRecurred,
		aiops.TransitionRecoveryStarted,
		aiops.TransitionClosed,
	)
	for _, tr := range f.handler.transitions {
		if tr.IncidentID != "inc-1" || tr.ClusterID != testCluster || tr.SubjectKey != svc || tr.At.IsZero() {
			t.Errorf("每条变化都应带着事件号、集群、主体与时刻: %+v", tr)
		}
	}
	if got := f.handler.last(t); got.Severity != "high" || got.Recurrence != 1 || got.PeakRisk != 0.75 {
		t.Errorf("关闭时应保留峰值严重度、复发次数、峰值风险: %+v", got)
	}
}

// ==================== 这一轮异常的起点 ====================

// 升级、恢复等后续变化都带着这一轮异常的起点（建事件的时刻），通知据此算持续时间；复发时起点换成复发的时刻
func TestEpisodeStartedAt_CarriedThroughAndResetOnRecurrence(t *testing.T) {
	f := newFixture(t)
	f.openWarning(svc, 0.6)
	created := f.handler.last(t)
	if !created.EpisodeStartedAt.Equal(created.At.Add(-2 * time.Minute)) {
		t.Fatalf("建事件时起点应是条件开始满足的时刻（开事件前 2 分钟）: %+v", created)
	}

	f.hold(5*time.Minute, selfRisk(svc, 0.6))
	escalated := f.handler.last(t)
	if escalated.Kind != aiops.TransitionEscalated || !escalated.EpisodeStartedAt.Equal(created.EpisodeStartedAt) {
		t.Fatalf("升级应沿用建事件的起点，实际 %v（建事件 %v）", escalated.EpisodeStartedAt, created.EpisodeStartedAt)
	}

	f.recover(svc, 10*time.Minute)
	if got := f.handler.last(t); !got.EpisodeStartedAt.Equal(created.EpisodeStartedAt) {
		t.Errorf("恢复的变化仍带这一轮的起点，实际 %v", got.EpisodeStartedAt)
	}

	f.clock.Advance(time.Hour)
	f.evaluate(selfRisk(svc, 0.3))
	recurred := f.handler.last(t)
	if recurred.Kind != aiops.TransitionRecurred || !recurred.EpisodeStartedAt.Equal(recurred.At) {
		t.Errorf("复发后起点应换成复发的时刻: %+v", recurred)
	}
}

// 异常持续超过 24 小时：近 24 小时的发生次数滑出窗口，但起点不丢
func TestEpisodeStartedAt_SurvivesBeyondTwentyFourHours(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	start := f.sm.GetEntry(testCluster, svc).EpisodeStartedAt

	f.hold(30*time.Hour, selfRisk(svc, 0.6))
	f.recover(svc, 10*time.Minute)
	got := f.handler.last(t)
	if got.Kind != aiops.TransitionRecoveryStarted || got.EpisodeStartedAt.Unix() != start {
		t.Errorf("30 小时后的恢复仍应带原来的起点 %d，实际 %v（%+v）", start, got.EpisodeStartedAt.Unix(), got)
	}
}

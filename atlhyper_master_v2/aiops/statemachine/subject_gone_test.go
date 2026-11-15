package statemachine

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// other 同一集群里另一个一直存在的主体：保证每轮评估的主体集合不为空
const other = "shop/service/other"

// 对象删除后持续 2 分钟不在评估里：进入恢复观察并标明「对象已删除」；不到 2 分钟不动
func TestSubjectGone_EntersRecoveryAfterHold(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)

	f.hold(time.Minute+50*time.Second, selfRisk(other, 0))
	if f.state(svc) != aiops.StateIncident {
		t.Fatalf("消失不满 2 分钟不应动，实际 %s", f.state(svc))
	}
	f.clock.Advance(10 * time.Second)
	f.evaluate(selfRisk(other, 0))

	got := f.handler.last(t)
	if got.Kind != aiops.TransitionRecoveryStarted || !got.SubjectGone || got.ToState != aiops.StateRecovery {
		t.Fatalf("应进入恢复观察并标明对象已删除: %+v", got)
	}
	entry := f.sm.GetEntry(testCluster, svc)
	if entry.RecoveryStartedAt != f.clock.Now().Unix() {
		t.Error("应记下进入观察期的时刻")
	}
}

// 消失后又回来（不到 2 分钟）：计时清零，不进入恢复
func TestSubjectGone_ReturnsBeforeHoldResets(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.hold(time.Minute, selfRisk(other, 0))
	f.evaluate(selfRisk(svc, 0.6), selfRisk(other, 0))
	f.hold(time.Minute+30*time.Second, selfRisk(other, 0))
	if f.state(svc) != aiops.StateIncident {
		t.Errorf("回来过一次后重新计时，1 分 30 秒不应进入恢复，实际 %s", f.state(svc))
	}
}

// 删除进入观察后同名对象又出问题：升级过的事件立即复发，并入同一事件
func TestSubjectGone_RecreatedAndFailingRecurs(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.hold(2*time.Minute, selfRisk(other, 0))
	f.evaluate(selfRisk(other, 0))

	f.clock.Advance(10 * time.Minute)
	f.evaluate(selfRisk(svc, 0.6), selfRisk(other, 0))
	got := f.handler.last(t)
	if got.Kind != aiops.TransitionRecurred || got.IncidentID != "inc-1" || got.SubjectGone {
		t.Errorf("重新部署后又出问题应立即复发并入原事件: %+v", got)
	}
}

// 删除后的观察期不会被「失联」提前关掉：每轮评估都确认它不在，算作有数据；观察期满正常关闭
func TestSubjectGone_NotClosedAsStale(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	f.hold(2*time.Minute, selfRisk(other, 0))
	f.evaluate(selfRisk(other, 0))

	f.hold(40*time.Minute, selfRisk(other, 0))
	f.sm.CleanupStaleEntries(context.Background(), 30*time.Minute)
	if f.state(svc) != aiops.StateRecovery {
		t.Fatalf("Agent 还在上报时不应按失联关闭，实际 %s", f.state(svc))
	}

	f.clock.Advance(48 * time.Hour)
	f.sm.CheckRecoveryToStable(context.Background())
	if got := f.handler.last(t); got.Kind != aiops.TransitionClosed {
		t.Errorf("观察期满应按恢复关闭: %+v", got)
	}
}

// 本轮评估里一个主体都没有（快照异常）：不判定任何主体已删除
func TestSubjectGone_EmptyRoundIgnored(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	for i := 0; i < 5; i++ {
		f.clock.Advance(time.Minute)
		f.sm.Evaluate(context.Background(), testCluster, map[string]*aiops.SubjectRisk{}, nil)
	}
	if f.state(svc) != aiops.StateIncident {
		t.Errorf("空的一轮不应判定删除，实际 %s", f.state(svc))
	}
}

// 没有事件的主体消失：只清条目，不发变化
func TestSubjectGone_HealthyEntryDropped(t *testing.T) {
	f := newFixture(t)
	f.evaluate(selfRisk(svc, 0.05), selfRisk(other, 0))
	before := len(f.handler.kinds())
	f.hold(3*time.Minute, selfRisk(other, 0))
	if f.sm.GetEntry(testCluster, svc) != nil || len(f.handler.kinds()) != before {
		t.Error("没有事件的主体消失后应清掉条目、不发变化")
	}
}

// 另一个集群的主体不受这一轮评估影响
func TestSubjectGone_OtherClusterUntouched(t *testing.T) {
	f := newFixture(t)
	f.openIncident(svc, 0.6)
	for i := 0; i < 4; i++ {
		f.clock.Advance(time.Minute)
		f.sm.Evaluate(context.Background(), "cluster-2", map[string]*aiops.SubjectRisk{other: selfRisk(other, 0)}, nil)
	}
	if f.state(svc) != aiops.StateIncident {
		t.Errorf("别的集群的评估不应影响本集群的主体，实际 %s", f.state(svc))
	}
}

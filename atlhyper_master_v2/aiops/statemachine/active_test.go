package statemachine

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// 只列出本集群里警告 / 事件中的条目；恢复观察、没有事件的、别的集群的都不列
func TestActiveIncidents(t *testing.T) {
	f := newFixture(t)
	// 同一轮里一起评估（单独评估一个会让另一个看起来已被删除）
	both := func() []*aiops.SubjectRisk {
		return []*aiops.SubjectRisk{selfRisk(svc, 0.6), selfRisk(api, 0.3), selfRisk("shop/service/quiet", 0.05)}
	}
	f.hold(2*time.Minute, both()...) // 建事件
	f.hold(5*time.Minute, both()...) // svc 升级
	f.evaluateIn("cluster-2", nil, selfRisk(svc, 0.3))
	f.clock.Advance(3 * time.Minute)
	f.evaluateIn("cluster-2", nil, selfRisk(svc, 0.3))

	active := f.sm.ActiveIncidents(testCluster)
	if len(active) != 2 || active[0].SubjectKey != api || active[1].SubjectKey != svc {
		t.Fatalf("应列出本集群两条活跃条目（按主体键排序）: %+v", active)
	}
	if active[0].CurrentState != aiops.StateWarning || active[1].CurrentState != aiops.StateIncident {
		t.Errorf("状态不对: %s / %s", active[0].CurrentState, active[1].CurrentState)
	}
}

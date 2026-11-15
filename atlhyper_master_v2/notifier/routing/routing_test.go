package routing

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

func intPtr(v int) *int { return &v }

func testReceivers() []*database.NotifyReceiver {
	return []*database.NotifyReceiver{
		{ID: 1, Name: "值班频道", Type: "slack", Enabled: true},
		{ID: 2, Name: "运维邮件", Type: "email", Enabled: true},
		{ID: 3, Name: "开发频道", Type: "slack", Enabled: false},
	}
}

func receiverIDs(receivers []*database.NotifyReceiver) []int64 {
	ids := make([]int64, len(receivers))
	for i, r := range receivers {
		ids[i] = r.ID
	}
	return ids
}

func sameIDs(got []int64, want ...int64) bool {
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

func TestRoute(t *testing.T) {
	prodHigh := &database.NotifyRule{ID: 1, Name: "生产高严重度", Position: 1, Enabled: true,
		ClusterIDs: []string{"prod"}, Sources: []string{"incident"}, MinSeverity: "high", ReceiverIDs: []int64{1, 2}}
	prodAny := &database.NotifyRule{ID: 2, Name: "生产全部", Position: 2, Enabled: true,
		ClusterIDs: []string{"prod"}, MinSeverity: "low", ReceiverIDs: []int64{2}}
	disabled := &database.NotifyRule{ID: 3, Name: "停用的", Position: 0, Enabled: false, MinSeverity: "low", ReceiverIDs: []int64{3}}
	catchAll := &database.NotifyRule{ID: 4, Name: "空条件", Position: 9, Enabled: true, MinSeverity: "low", ReceiverIDs: []int64{1}}

	policy := database.DefaultNotifyPolicy()
	policy.MinSeverity = "medium"

	cases := []struct {
		name         string
		rules        []*database.NotifyRule
		target       Target
		wantRuleID   int64
		wantSend     []int64
		wantBelowMin bool
	}{
		{"按顺序：第一条命中即停", []*database.NotifyRule{prodAny, prodHigh},
			Target{"prod", alert.SourceIncident, alert.SeverityCritical}, 1, []int64{1, 2}, false},
		{"严重度不够第一条，落到第二条", []*database.NotifyRule{prodHigh, prodAny},
			Target{"prod", alert.SourceIncident, alert.SeverityMedium}, 2, []int64{2}, false},
		{"来源不在第一条里，落到第二条", []*database.NotifyRule{prodHigh, prodAny},
			Target{"prod", alert.SourceHeartbeat, alert.SeverityCritical}, 2, []int64{2}, false},
		{"禁用的规则排在最前也不生效", []*database.NotifyRule{disabled, prodHigh},
			Target{"prod", alert.SourceIncident, alert.SeverityHigh}, 1, []int64{1, 2}, false},
		{"空条件匹配任何集群与来源", []*database.NotifyRule{prodHigh, catchAll},
			Target{"test", alert.SourceK8sEvent, alert.SeverityLow}, 4, []int64{1}, false},
		{"都不命中：默认策略，空的默认接收人 = 全部已启用的", []*database.NotifyRule{prodHigh},
			Target{"test", alert.SourceIncident, alert.SeverityHigh}, 0, []int64{1, 2}, false},
		{"默认策略下严重度不到门槛：不发送", []*database.NotifyRule{prodHigh},
			Target{"test", alert.SourceIncident, alert.SeverityLow}, 0, nil, true},
		{"没有任何规则", nil,
			Target{"prod", alert.SourceIncident, alert.SeverityMedium}, 0, []int64{1, 2}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			decision := Route(policy, c.rules, testReceivers(), c.target)
			if decision.RuleID() != c.wantRuleID {
				t.Errorf("命中的规则是 %d，期望 %d", decision.RuleID(), c.wantRuleID)
			}
			if decision.BelowMinSeverity != c.wantBelowMin {
				t.Errorf("BelowMinSeverity = %v，期望 %v", decision.BelowMinSeverity, c.wantBelowMin)
			}
			if got := receiverIDs(decision.Deliverable()); !sameIDs(got, c.wantSend...) {
				t.Errorf("会发给 %v，期望 %v", got, c.wantSend)
			}
		})
	}
}

func TestRoute_DefaultReceiverIDs(t *testing.T) {
	policy := database.DefaultNotifyPolicy()
	policy.DefaultReceiverIDs = []int64{3, 2, 99} // 3 已停用，99 不存在

	decision := Route(policy, nil, testReceivers(), Target{"prod", alert.SourceIncident, alert.SeverityHigh})
	if got := receiverIDs(decision.Receivers); !sameIDs(got, 3, 2) {
		t.Errorf("指定的接收人应按给定顺序列出（不存在的略过），实际 %v", got)
	}
	if got := receiverIDs(decision.Deliverable()); !sameIDs(got, 2) {
		t.Errorf("停用的接收人不会收到，实际 %v", got)
	}
}

func TestRoute_RuleReceiversKeepDisabledForDisplay(t *testing.T) {
	rule := &database.NotifyRule{ID: 1, Position: 1, Enabled: true, MinSeverity: "low", ReceiverIDs: []int64{3, 1}}
	decision := Route(database.DefaultNotifyPolicy(), []*database.NotifyRule{rule}, testReceivers(),
		Target{"prod", alert.SourceIncident, alert.SeverityLow})
	if got := receiverIDs(decision.Receivers); !sameIDs(got, 3, 1) {
		t.Errorf("规则指定的接收人应全部列出（含停用的），实际 %v", got)
	}
	if got := receiverIDs(decision.Deliverable()); !sameIDs(got, 1) {
		t.Errorf("只有启用的会收到，实际 %v", got)
	}
}

func TestRoute_TimingOverride(t *testing.T) {
	policy := database.DefaultNotifyPolicy() // 30 秒 / 5 分钟 / 4 小时
	override := &database.NotifyRule{ID: 1, Position: 1, Enabled: true, MinSeverity: "low", ReceiverIDs: []int64{1},
		GroupWaitSeconds: intPtr(0), RepeatIntervalSeconds: intPtr(3600)}

	decision := Route(policy, []*database.NotifyRule{override}, testReceivers(), Target{"prod", alert.SourceIncident, alert.SeverityLow})
	want := Timing{GroupWait: 0, GroupInterval: 5 * time.Minute, RepeatInterval: time.Hour}
	if decision.Timing != want {
		t.Errorf("规则里填了的时间参数覆盖全局值、没填的用全局值：实际 %+v，期望 %+v", decision.Timing, want)
	}

	fallback := Route(policy, nil, testReceivers(), Target{"prod", alert.SourceIncident, alert.SeverityLow})
	if fallback.Timing != (Timing{30 * time.Second, 5 * time.Minute, 4 * time.Hour}) {
		t.Errorf("默认策略应用全局时间参数，实际 %+v", fallback.Timing)
	}
}

// 规则的顺序以 Position 为准，与传入切片的顺序无关
func TestRoute_OrdersByPositionNotSliceOrder(t *testing.T) {
	late := &database.NotifyRule{ID: 1, Position: 5, Enabled: true, MinSeverity: "low", ReceiverIDs: []int64{1}}
	early := &database.NotifyRule{ID: 2, Position: 1, Enabled: true, MinSeverity: "low", ReceiverIDs: []int64{2}}
	decision := Route(database.DefaultNotifyPolicy(), []*database.NotifyRule{late, early}, testReceivers(),
		Target{"prod", alert.SourceIncident, alert.SeverityLow})
	if decision.RuleID() != 2 {
		t.Errorf("应命中位置靠前的规则 2，实际 %d", decision.RuleID())
	}
}

func TestReceiversForRule(t *testing.T) {
	policy := database.DefaultNotifyPolicy()
	rules := []*database.NotifyRule{{ID: 7, Enabled: true, ReceiverIDs: []int64{3, 2}}}

	if got := receiverIDs(ReceiversForRule(policy, rules, testReceivers(), 7)); !sameIDs(got, 2) {
		t.Errorf("规则的接收人里只有启用的会收到，实际 %v", got)
	}
	if got := receiverIDs(ReceiversForRule(policy, rules, testReceivers(), 0)); !sameIDs(got, 1, 2) {
		t.Errorf("默认策略：全部已启用的接收人，实际 %v", got)
	}
	if got := receiverIDs(ReceiversForRule(policy, rules, testReceivers(), 99)); !sameIDs(got, 1, 2) {
		t.Errorf("规则已不存在时退回默认策略，实际 %v", got)
	}
	policy.DefaultReceiverIDs = []int64{2}
	if got := receiverIDs(ReceiversForRule(policy, nil, testReceivers(), 0)); !sameIDs(got, 2) {
		t.Errorf("默认策略指定了接收人时只发给它们，实际 %v", got)
	}
}

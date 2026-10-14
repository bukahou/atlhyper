package incident

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

func tl(at time.Time, eventType, detail string) *database.AIOpsIncidentTimeline {
	return &database.AIOpsIncidentTimeline{Timestamp: at, EventType: eventType, Detail: detail}
}

// 异常时长 = 开始 → 最后一段的恢复时刻；48h 观察期不算进去
func TestActiveWindow(t *testing.T) {
	t0 := time.Date(2026, 10, 14, 18, 21, 34, 0, time.UTC)
	at := func(m int) time.Time { return t0.Add(time.Duration(m) * time.Minute) }

	cases := []struct {
		name          string
		timeline      []*database.AIOpsIncidentTimeline
		wantEnd       time.Time
		wantRecovered bool
	}{
		{"正常恢复", []*database.AIOpsIncidentTimeline{
			tl(at(0), aiops.TimelineStateChange, "状态变更: healthy → warning"),
			tl(at(6), aiops.TimelineStateChange, "状态变更 → incident (风险 56)"),
			tl(at(18), aiops.TimelineRecoveryStarted, "状态变更 → recovery (风险 0)"),
		}, at(18), true},
		{"复发后再恢复：取最后一段", []*database.AIOpsIncidentTimeline{
			tl(at(18), aiops.TimelineRecoveryStarted, "状态变更 → recovery (风险 0)"),
			tl(at(40), aiops.TimelineRecurrence, "复发 (风险 48)"),
			tl(at(55), aiops.TimelineRecoveryStarted, "状态变更 → recovery (风险 0)"),
			tl(at(0), aiops.TimelineStateChange, "状态变更: healthy → warning"), // 乱序输入也要按时间处理
		}, at(55), true},
		{"复发后仍在异常", []*database.AIOpsIncidentTimeline{
			tl(at(18), aiops.TimelineRecoveryStarted, "状态变更 → recovery (风险 0)"),
			tl(at(40), aiops.TimelineRecurrence, "复发 (风险 48)"),
		}, time.Time{}, false},
		{"仍在异常", []*database.AIOpsIncidentTimeline{
			tl(at(0), aiops.TimelineStateChange, "状态变更: healthy → warning"),
		}, time.Time{}, false},
		{"旧数据：recovery 以 state_change 记录", []*database.AIOpsIncidentTimeline{
			tl(at(0), aiops.TimelineStateChange, "状态变更: healthy → warning"),
			tl(at(18), aiops.TimelineStateChange, "状态变更 → recovery (风险 0)"),
		}, at(18), true},
	}
	for _, c := range cases {
		end, recovered := ActiveWindow(c.timeline)
		if recovered != c.wantRecovered || !end.Equal(c.wantEnd) {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", c.name, end, recovered, c.wantEnd, c.wantRecovered)
		}
	}
}

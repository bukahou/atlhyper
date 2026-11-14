// atlhyper_master_v2/aiops/incident/timeline.go
// 事件时间线操作
package incident

import (
	"context"
	"sort"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// timelineWrite 一条时间线写入。timestamp 是事情发生的时刻，重试时不变。
func (s *Store) timelineWrite(incidentID string, timestamp time.Time, eventType, entityKey, detail string) pendingWrite {
	entry := &database.AIOpsIncidentTimeline{
		IncidentID: incidentID,
		Timestamp:  timestamp,
		EventType:  eventType,
		EntityKey:  entityKey,
		Detail:     detail,
	}
	return pendingWrite{"添加时间线 " + incidentID + " " + eventType, func(ctx context.Context) error {
		return s.repo.AddTimeline(ctx, entry)
	}}
}

// AddTimeline 外部调用添加时间线
func (s *Store) AddTimeline(ctx context.Context, incidentID string, timestamp time.Time, eventType, entityKey, detail string) {
	s.submit(ctx, s.timelineWrite(incidentID, timestamp, eventType, entityKey, detail))
}

// signalTimelineWrite 一条「异常检测」时间线：写明本次状态变化由哪些信号推动
func (s *Store) signalTimelineWrite(incidentID string, now time.Time, risk *aiops.EntityRisk) pendingWrite {
	return s.timelineWrite(incidentID, now, aiops.TimelineAnomalyDetected, risk.EntityKey, signalDetail(risk))
}

// signalDetail 信号逐条以「；」连接；本地无异常时说明风险来自依赖传播
func signalDetail(risk *aiops.EntityRisk) string {
	if risk.RLocal == 0 || len(risk.Signals) == 0 {
		return "无本地异常信号（风险由依赖传播）"
	}
	parts := make([]string, len(risk.Signals))
	for i, sig := range risk.Signals {
		parts[i] = sig.Detail
	}
	return strings.Join(parts, "；")
}

// legacyRecoveryPrefix 2026-10-14 之前 recovery 以 state_change 记录，只能靠文案识别
const legacyRecoveryPrefix = "状态变更 → recovery"

// ActiveWindow 异常段的结束时刻：最后一次复发之后的首个恢复（48h 观察期不算异常）。
// recovered=false 表示仍在异常中（或时间线未记录恢复）
func ActiveWindow(timeline []*database.AIOpsIncidentTimeline) (end time.Time, recovered bool) {
	sorted := make([]*database.AIOpsIncidentTimeline, len(timeline))
	copy(sorted, timeline)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Timestamp.Before(sorted[j].Timestamp) })

	for _, t := range sorted {
		switch {
		case t.EventType == aiops.TimelineRecurrence:
			end, recovered = time.Time{}, false
		case !recovered && isRecoveryEntry(t):
			end, recovered = t.Timestamp, true
		}
	}
	return end, recovered
}

func isRecoveryEntry(t *database.AIOpsIncidentTimeline) bool {
	return t.EventType == aiops.TimelineRecoveryStarted ||
		(t.EventType == aiops.TimelineStateChange && strings.HasPrefix(t.Detail, legacyRecoveryPrefix))
}

// RecoveryStartedAt 事件进入恢复观察期的时刻：最后一次复发之后的首个恢复记录。
// ok=false 表示事件当前不在观察期，或时间线读不到。
//
// 状态机的条目只在内存里，Master 重启后据此取回观察期的起点，不从重启那一刻重新计 48 小时。
func (s *Store) RecoveryStartedAt(ctx context.Context, incidentID string) (startedAt time.Time, ok bool) {
	timeline, err := s.repo.GetTimeline(ctx, incidentID)
	if err != nil {
		log.Warn("读取事件时间线失败", "id", incidentID, "err", err)
		return time.Time{}, false
	}
	return ActiveWindow(timeline)
}

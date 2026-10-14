// atlhyper_master_v2/aiops/incident/timeline.go
// 事件时间线操作
package incident

import (
	"context"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// addTimeline 添加时间线条目
func (s *Store) addTimeline(ctx context.Context, incidentID string, timestamp time.Time, eventType, entityKey, detail string) {
	entry := &database.AIOpsIncidentTimeline{
		IncidentID: incidentID,
		Timestamp:  timestamp,
		EventType:  eventType,
		EntityKey:  entityKey,
		Detail:     detail,
	}
	if err := s.repo.AddTimeline(ctx, entry); err != nil {
		log.Error("添加时间线失败", "incident", incidentID, "event", eventType, "err", err)
	}
}

// AddTimeline 外部调用添加时间线
func (s *Store) AddTimeline(ctx context.Context, incidentID string, timestamp time.Time, eventType, entityKey, detail string) {
	s.addTimeline(ctx, incidentID, timestamp, eventType, entityKey, detail)
}

// addSignalTimeline 追加一条「异常检测」：写明本次状态变化由哪些信号推动
func (s *Store) addSignalTimeline(ctx context.Context, incidentID string, now time.Time, risk *aiops.EntityRisk) {
	if incidentID == "" || risk == nil {
		return
	}
	s.addTimeline(ctx, incidentID, now, aiops.TimelineAnomalyDetected, risk.EntityKey, signalDetail(risk))
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

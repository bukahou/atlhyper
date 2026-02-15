// atlhyper_master_v2/aiops/incident/timeline.go
// 事件时间线操作
package incident

import (
	"context"
	"time"

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

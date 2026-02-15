// atlhyper_master_v2/aiops/incident/patterns.go
// 历史事件模式查询
package incident

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// GetPatterns 获取与指定实体相关的历史事件模式
func (s *Store) GetPatterns(ctx context.Context, entityKey string, since time.Time) []*aiops.IncidentPattern {
	incidents, err := s.repo.ListByEntity(ctx, entityKey, since)
	if err != nil {
		log.Error("查询实体事件历史失败", "entity", entityKey, "err", err)
		return nil
	}

	if len(incidents) == 0 {
		return nil
	}

	// 统计模式
	var totalDuration int64
	var lastOccurrence time.Time
	aiopsIncidents := make([]*aiops.Incident, len(incidents))

	for i, inc := range incidents {
		v := toAIOpsIncident(inc)
		aiopsIncidents[i] = &v
		totalDuration += inc.DurationS
		if inc.StartedAt.After(lastOccurrence) {
			lastOccurrence = inc.StartedAt
		}
	}

	avgDuration := 0.0
	if len(incidents) > 0 {
		avgDuration = float64(totalDuration) / float64(len(incidents))
	}

	return []*aiops.IncidentPattern{
		{
			EntityKey:      entityKey,
			PatternCount:   len(incidents),
			AvgDuration:    avgDuration,
			LastOccurrence: lastOccurrence,
			Incidents:      aiopsIncidents,
		},
	}
}

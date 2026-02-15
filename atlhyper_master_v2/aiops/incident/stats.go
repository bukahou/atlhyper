// atlhyper_master_v2/aiops/incident/stats.go
// 事件统计
package incident

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// GetStats 获取事件统计
func (s *Store) GetStats(ctx context.Context, clusterID string, since time.Time) *aiops.IncidentStats {
	raw, err := s.repo.GetIncidentStats(ctx, clusterID, since)
	if err != nil {
		log.Error("获取事件统计失败", "cluster", clusterID, "err", err)
		return &aiops.IncidentStats{
			BySeverity: make(map[string]int),
			ByState:    make(map[string]int),
		}
	}

	// 计算复发率
	recurrenceRate := 0.0
	if raw.TotalIncidents > 0 {
		recurrenceRate = float64(raw.RecurringCount) / float64(raw.TotalIncidents) * 100
	}

	// 获取 Top 根因
	topRootCauses, err := s.repo.TopRootCauses(ctx, clusterID, since, 10)
	if err != nil {
		log.Error("获取根因统计失败", "cluster", clusterID, "err", err)
	}

	rootCauses := make([]aiops.RootCauseCount, len(topRootCauses))
	for i, rc := range topRootCauses {
		rootCauses[i] = aiops.RootCauseCount{
			EntityKey: rc.EntityKey,
			Count:     rc.Count,
		}
	}

	return &aiops.IncidentStats{
		TotalIncidents:  raw.TotalIncidents,
		ActiveIncidents: raw.ActiveIncidents,
		MTTR:            raw.MTTR,
		RecurrenceRate:  recurrenceRate,
		BySeverity:      raw.BySeverity,
		ByState:         raw.ByState,
		TopRootCauses:   rootCauses,
	}
}

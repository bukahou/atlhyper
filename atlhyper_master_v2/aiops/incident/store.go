// atlhyper_master_v2/aiops/incident/store.go
// 事件存储: 封装 Repository，提供业务层 CRUD
package incident

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps.Incident")

// Store 事件存储
type Store struct {
	repo database.AIOpsIncidentRepository
}

// NewStore 创建事件存储
func NewStore(repo database.AIOpsIncidentRepository) *Store {
	return &Store{repo: repo}
}

// Create 创建事件
func (s *Store) Create(ctx context.Context, clusterID, entityKey string, risk *aiops.EntityRisk, now time.Time) string {
	id := fmt.Sprintf("inc-%d", now.UnixMilli())
	severity := aiops.SeverityFromRisk(risk.RFinal)

	inc := &database.AIOpsIncident{
		ID:        id,
		ClusterID: clusterID,
		State:     string(aiops.StateWarning),
		Severity:  severity,
		RootCause: entityKey,
		PeakRisk:  risk.RFinal,
		StartedAt: now,
		CreatedAt: now,
	}

	if err := s.repo.CreateIncident(ctx, inc); err != nil {
		log.Error("创建事件失败", "id", id, "err", err)
		return ""
	}

	// 添加触发实体
	entity := &database.AIOpsIncidentEntity{
		IncidentID: id,
		EntityKey:  entityKey,
		EntityType: aiops.ExtractEntityType(entityKey),
		RLocal:     risk.RLocal,
		RFinal:     risk.RFinal,
		Role:       "root_cause",
	}
	if err := s.repo.AddEntity(ctx, entity); err != nil {
		log.Error("添加事件实体失败", "id", id, "entity", entityKey, "err", err)
	}

	// 添加时间线
	s.addTimeline(ctx, id, now, aiops.TimelineStateChange, entityKey, "状态变更: healthy → warning")

	log.Info("事件已创建", "id", id, "entity", entityKey, "severity", severity)
	return id
}

// UpdateState 更新事件状态
func (s *Store) UpdateState(ctx context.Context, incidentID string, state aiops.EntityState, risk *aiops.EntityRisk, now time.Time) {
	severity := aiops.SeverityFromRisk(risk.RFinal)
	if err := s.repo.UpdateState(ctx, incidentID, string(state), severity); err != nil {
		log.Error("更新事件状态失败", "id", incidentID, "err", err)
		return
	}

	// 更新峰值风险
	if err := s.repo.UpdatePeakRisk(ctx, incidentID, risk.RFinal); err != nil {
		log.Error("更新峰值风险失败", "id", incidentID, "err", err)
	}

	detail := fmt.Sprintf("状态变更 → %s (R_final=%.3f)", state, risk.RFinal)
	s.addTimeline(ctx, incidentID, now, aiops.TimelineStateChange, risk.EntityKey, detail)
}

// Resolve 解决事件
func (s *Store) Resolve(ctx context.Context, incidentID, entityKey string, now time.Time) {
	if err := s.repo.UpdateState(ctx, incidentID, string(aiops.StateStable), ""); err != nil {
		log.Error("解决事件失败", "id", incidentID, "err", err)
		return
	}
	if err := s.repo.Resolve(ctx, incidentID, now); err != nil {
		log.Error("设置解决时间失败", "id", incidentID, "err", err)
	}
	s.addTimeline(ctx, incidentID, now, aiops.TimelineStateChange, entityKey, "事件已解决: recovery → stable")
}

// UpdateRootCause 更新根因实体
func (s *Store) UpdateRootCause(ctx context.Context, incidentID, rootCause string) {
	if err := s.repo.UpdateRootCause(ctx, incidentID, rootCause); err != nil {
		log.Error("更新根因失败", "id", incidentID, "err", err)
	}
}

// IncrementRecurrence 递增复发次数
func (s *Store) IncrementRecurrence(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time) {
	if err := s.repo.IncrementRecurrence(ctx, incidentID); err != nil {
		log.Error("递增复发次数失败", "id", incidentID, "err", err)
		return
	}
	detail := fmt.Sprintf("复发 (R_final=%.3f)", risk.RFinal)
	s.addTimeline(ctx, incidentID, now, aiops.TimelineRecurrence, risk.EntityKey, detail)
}

// GetIncident 获取事件详情
func (s *Store) GetIncident(ctx context.Context, incidentID string) *aiops.IncidentDetail {
	inc, err := s.repo.GetByID(ctx, incidentID)
	if err != nil || inc == nil {
		return nil
	}

	entities, _ := s.repo.GetEntities(ctx, incidentID)
	timeline, _ := s.repo.GetTimeline(ctx, incidentID)

	return &aiops.IncidentDetail{
		Incident: toAIOpsIncident(inc),
		Entities: toAIOpsEntities(entities),
		Timeline: toAIOpsTimeline(timeline),
	}
}

// GetIncidents 查询事件列表
func (s *Store) GetIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error) {
	dbOpts := database.AIOpsIncidentQueryOpts{
		ClusterID: opts.ClusterID,
		State:     opts.State,
		Severity:  opts.Severity,
		From:      opts.From,
		To:        opts.To,
		Limit:     opts.Limit,
		Offset:    opts.Offset,
	}

	incidents, err := s.repo.List(ctx, dbOpts)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count(ctx, dbOpts)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*aiops.Incident, len(incidents))
	for i, inc := range incidents {
		v := toAIOpsIncident(inc)
		result[i] = &v
	}

	return result, total, nil
}

// ==================== 类型转换 ====================

func toAIOpsIncident(inc *database.AIOpsIncident) aiops.Incident {
	return aiops.Incident{
		ID:         inc.ID,
		ClusterID:  inc.ClusterID,
		State:      aiops.EntityState(inc.State),
		Severity:   inc.Severity,
		RootCause:  inc.RootCause,
		PeakRisk:   inc.PeakRisk,
		StartedAt:  inc.StartedAt,
		ResolvedAt: inc.ResolvedAt,
		DurationS:  inc.DurationS,
		Recurrence: inc.Recurrence,
		Summary:    inc.Summary,
		CreatedAt:  inc.CreatedAt,
	}
}

func toAIOpsEntities(entities []*database.AIOpsIncidentEntity) []*aiops.IncidentEntity {
	result := make([]*aiops.IncidentEntity, len(entities))
	for i, e := range entities {
		result[i] = &aiops.IncidentEntity{
			IncidentID: e.IncidentID,
			EntityKey:  e.EntityKey,
			EntityType: e.EntityType,
			RLocal:     e.RLocal,
			RFinal:     e.RFinal,
			Role:       e.Role,
		}
	}
	return result
}

func toAIOpsTimeline(timeline []*database.AIOpsIncidentTimeline) []*aiops.IncidentTimeline {
	result := make([]*aiops.IncidentTimeline, len(timeline))
	for i, t := range timeline {
		result[i] = &aiops.IncidentTimeline{
			ID:         t.ID,
			IncidentID: t.IncidentID,
			Timestamp:  t.Timestamp,
			EventType:  t.EventType,
			EntityKey:  t.EntityKey,
			Detail:     t.Detail,
		}
	}
	return result
}

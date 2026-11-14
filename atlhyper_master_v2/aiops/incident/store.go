// atlhyper_master_v2/aiops/incident/store.go
// 事件存储: 封装 Repository，提供业务层 CRUD
package incident

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

// incidentSeq 自增序列号，避免同一毫秒内多个事件 ID 冲突
var incidentSeq atomic.Int64

var log = logger.Module("AIOps.Incident")

// Store 事件存储
type Store struct {
	repo database.AIOpsIncidentRepository

	// 写入的重试队列（见 write_queue.go）
	writeMu    sync.Mutex
	pending    []pendingWrite
	retryAfter time.Time        // 上次暂时性失败后，到这个时刻之前不再尝试写库
	now        func() time.Time // 可替换的时钟，测试用
}

// NewStore 创建事件存储
func NewStore(repo database.AIOpsIncidentRepository) *Store {
	return &Store{repo: repo, now: time.Now}
}

// Create 创建事件。
// 事件号在本地生成并立即返回；写库走重试队列，状态机据此关联这条事件之后的变化。
func (s *Store) Create(ctx context.Context, clusterID, entityKey string, risk *aiops.EntityRisk, now time.Time) string {
	seq := incidentSeq.Add(1)
	id := fmt.Sprintf("inc-%d%04d", now.UnixMilli(), seq%10000)
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
	// 触发实体
	entity := &database.AIOpsIncidentEntity{
		IncidentID: id,
		EntityKey:  entityKey,
		EntityType: aiops.ExtractEntityType(entityKey),
		RLocal:     risk.RLocal,
		RFinal:     risk.RFinal,
		Role:       "root_cause",
	}

	s.submit(ctx,
		pendingWrite{"创建事件 " + id, func(ctx context.Context) error { return s.repo.CreateIncident(ctx, inc) }},
		pendingWrite{"添加事件实体 " + id, func(ctx context.Context) error { return s.repo.AddEntity(ctx, entity) }},
		s.timelineWrite(id, now, aiops.TimelineStateChange, entityKey, "状态变更: healthy → warning"),
		s.signalTimelineWrite(id, now, risk),
	)

	log.Info("事件已创建", "id", id, "entity", entityKey, "severity", severity)
	return id
}

// UpdateState 更新事件状态
func (s *Store) UpdateState(ctx context.Context, incidentID string, state aiops.EntityState, risk *aiops.EntityRisk, now time.Time) {
	if incidentID == "" {
		return
	}
	severity := aiops.SeverityFromRisk(risk.RFinal)
	detail := fmt.Sprintf("状态变更 → %s (风险 %s)", state, displayRisk(risk.RFinal))

	writes := []pendingWrite{
		{"更新事件状态 " + incidentID, func(ctx context.Context) error {
			return s.repo.UpdateState(ctx, incidentID, string(state), severity)
		}},
		{"更新峰值风险 " + incidentID, func(ctx context.Context) error {
			return s.repo.UpdatePeakRisk(ctx, incidentID, risk.RFinal)
		}},
		// 实体风险随事件演进更新（曾停在创建时的值，界面「最终风险」与事件峰值对不上）
		s.entityRiskWrite(incidentID, risk),
	}
	if state == aiops.StateRecovery {
		// 独立类型：异常时长据此结构化识别（文案前缀不变，旧数据仍靠它兼容）
		writes = append(writes, s.timelineWrite(incidentID, now, aiops.TimelineRecoveryStarted, risk.EntityKey, detail))
	} else {
		writes = append(writes,
			s.timelineWrite(incidentID, now, aiops.TimelineStateChange, risk.EntityKey, detail),
			s.signalTimelineWrite(incidentID, now, risk),
		)
	}
	s.submit(ctx, writes...)
}

// Resolve 解决事件
func (s *Store) Resolve(ctx context.Context, incidentID, entityKey string, now time.Time) {
	if incidentID == "" {
		return // 从未建过事件的实体（一直健康）被清理时，没有可关闭的事件
	}
	s.submit(ctx,
		pendingWrite{"关闭事件 " + incidentID, func(ctx context.Context) error {
			return s.repo.UpdateState(ctx, incidentID, string(aiops.StateStable), "")
		}},
		pendingWrite{"设置解决时间 " + incidentID, func(ctx context.Context) error {
			return s.repo.Resolve(ctx, incidentID, now)
		}},
		s.timelineWrite(incidentID, now, aiops.TimelineStateChange, entityKey, "事件已解决: recovery → stable"),
	)
}

// UpdateRootCause 更新根因实体
func (s *Store) UpdateRootCause(ctx context.Context, incidentID, rootCause string) {
	s.submit(ctx, pendingWrite{"更新根因 " + incidentID, func(ctx context.Context) error {
		return s.repo.UpdateRootCause(ctx, incidentID, rootCause)
	}})
}

// IncrementRecurrence 递增复发次数
func (s *Store) IncrementRecurrence(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time) {
	if incidentID == "" {
		return
	}
	detail := fmt.Sprintf("复发 (风险 %s)", displayRisk(risk.RFinal))
	s.submit(ctx,
		pendingWrite{"递增复发次数 " + incidentID, func(ctx context.Context) error {
			return s.repo.IncrementRecurrence(ctx, incidentID)
		}},
		s.entityRiskWrite(incidentID, risk),
		s.timelineWrite(incidentID, now, aiops.TimelineRecurrence, risk.EntityKey, detail),
		s.signalTimelineWrite(incidentID, now, risk),
	)
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

// entityRiskWrite 更新事件中该实体的风险值，保留其角色与类型；实体不在事件中则不新增
func (s *Store) entityRiskWrite(incidentID string, risk *aiops.EntityRisk) pendingWrite {
	return pendingWrite{"更新实体风险 " + incidentID, func(ctx context.Context) error {
		entities, err := s.repo.GetEntities(ctx, incidentID)
		if err != nil {
			return err
		}
		for _, e := range entities {
			if e.EntityKey != risk.EntityKey {
				continue
			}
			e.RLocal, e.RFinal = risk.RLocal, risk.RFinal
			return s.repo.AddEntity(ctx, e) // (incident_id, entity_key) 主键覆盖写
		}
		return nil
	}}
}

// displayRisk 时间线文字用界面同款 0-100 刻度（核心层为 [0,1]，约定见 gateway/handler/aiops/scale_risk.go）
func displayRisk(v float64) string {
	return fmt.Sprintf("%.0f", math.Round(v*100))
}

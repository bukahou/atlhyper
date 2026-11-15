// atlhyper_master_v2/aiops/incident/store.go
// 事件存储: 封装 Repository，提供事件的查询；事件变化的落库见 transition.go
package incident

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

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

// GetIncident 获取事件详情
func (s *Store) GetIncident(ctx context.Context, incidentID string) *aiops.IncidentDetail {
	inc, err := s.repo.GetByID(ctx, incidentID)
	if err != nil || inc == nil {
		return nil
	}

	entities, _ := s.repo.GetEntities(ctx, incidentID)
	timeline, _ := s.repo.GetTimeline(ctx, incidentID)
	signals, err := s.repo.GetSignals(ctx, incidentID)
	if err != nil {
		log.Warn("读取事件证据失败", "id", incidentID, "err", err)
	}

	detail := &aiops.IncidentDetail{
		Incident: toAIOpsIncident(inc),
		Entities: toAIOpsEntities(entities),
		Timeline: toAIOpsTimeline(timeline),
	}
	detail.Evidence = BuildEvidence(signals)
	detail.Headline = BuildHeadline(detail.Evidence, len(detail.Entities))
	detail.TimelineView = BuildTimelineView(detail.Timeline)
	detail.Members = baseMembers(detail.Entities)
	return detail
}

// baseMembers 成员的基础信息（来自实体表）；当前状态由查询层按快照补上
func baseMembers(entities []*aiops.IncidentEntity) []aiops.IncidentMember {
	members := make([]aiops.IncidentMember, 0, len(entities))
	for _, e := range entities {
		members = append(members, aiops.IncidentMember{
			EntityKey: e.EntityKey, EntityType: e.EntityType, Namespace: e.Namespace, Name: e.Name,
			Role: e.Role, RFinal: e.RFinal, IsLead: e.Role == roleRootCause,
		})
	}
	return members
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
	subjectNamespace, subjectType, subjectName := aiops.SplitEntityKey(inc.Subject)
	return aiops.Incident{
		ID:               inc.ID,
		ClusterID:        inc.ClusterID,
		State:            aiops.EntityState(inc.State),
		Severity:         inc.Severity,
		Subject:          inc.Subject,
		SubjectType:      subjectType,
		SubjectNamespace: subjectNamespace,
		SubjectName:      subjectName,
		RootCause:        inc.RootCause,
		PeakRisk:         inc.PeakRisk,
		StartedAt:        inc.StartedAt,
		EscalatedAt:      inc.EscalatedAt,
		ResolvedAt:       inc.ResolvedAt,
		CloseReason:      inc.CloseReason,
		DurationS:        inc.DurationS,
		Recurrence:       inc.Recurrence,
		Summary:          inc.Summary,
		CreatedAt:        inc.CreatedAt,
	}
}

func toAIOpsEntities(entities []*database.AIOpsIncidentEntity) []*aiops.IncidentEntity {
	result := make([]*aiops.IncidentEntity, len(entities))
	for i, e := range entities {
		namespace, _, name := aiops.SplitEntityKey(e.EntityKey)
		result[i] = &aiops.IncidentEntity{
			IncidentID: e.IncidentID,
			EntityKey:  e.EntityKey,
			EntityType: e.EntityType,
			Namespace:  namespace,
			Name:       name,
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

// displayRisk 时间线文字用界面同款 0-100 刻度（核心层为 [0,1]，约定见 gateway/handler/aiops/scale_risk.go）
func displayRisk(v float64) string {
	return fmt.Sprintf("%.0f", math.Round(v*100))
}

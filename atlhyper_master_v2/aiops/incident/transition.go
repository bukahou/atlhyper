// atlhyper_master_v2/aiops/incident/transition.go
// 事件变化的落库：状态机每发出一条 IncidentTransition，在这里换成对事件表、实体表、时间线的写入
package incident

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// 事件实体的角色
const (
	roleRootCause = "root_cause" // 领头实体
	roleAffected  = "affected"   // 其余成员
)

// incidentSeq 自增序列号，避免同一毫秒内多个事件号冲突
var incidentSeq atomic.Int64

// NewIncidentID 分配事件号。
// 事件号在建事件的那一刻就定下、不等写库：写库走重试队列，状态机要靠事件号关联这条事件之后的变化。
func NewIncidentID(now time.Time) string {
	seq := incidentSeq.Add(1)
	return fmt.Sprintf("inc-%d%04d", now.UnixMilli(), seq%10000)
}

// ApplyTransition 把一次事件变化落库。
//
// risk 是触发这次变化的那一轮主体风险；关闭类变化没有对应的评估，为 nil。
// 写入一律走重试队列（见 write_queue.go），本方法不阻塞在数据库上。
func (s *Store) ApplyTransition(ctx context.Context, transition aiops.IncidentTransition, risk *aiops.SubjectRisk) {
	if transition.IncidentID == "" {
		return
	}

	var writes []pendingWrite
	switch transition.Kind {
	case aiops.TransitionCreated:
		writes = s.createdWrites(transition, risk)
	case aiops.TransitionEscalated:
		writes = s.escalatedWrites(transition, risk)
	case aiops.TransitionSeverityRaised:
		writes = s.severityRaisedWrites(transition, risk)
	case aiops.TransitionRecoveryStarted:
		writes = s.recoveryStartedWrites(transition)
	case aiops.TransitionRecurred:
		writes = s.recurredWrites(transition, risk)
	case aiops.TransitionClosed:
		writes = s.closedWrites(transition, aiops.CloseReasonRecovered, "事件已关闭: 恢复观察期满")
	case aiops.TransitionClosedStale:
		writes = s.closedWrites(transition, aiops.CloseReasonStale, "事件已关闭: 主体长时间没有数据（对象已删除或 Agent 断连）")
	default:
		log.Warn("未知的事件变化，未落库", "kind", transition.Kind, "incident", transition.IncidentID)
		return
	}
	s.submit(ctx, writes...)
}

// createdWrites 健康 → 警告：建事件行、写成员、时间线
func (s *Store) createdWrites(t aiops.IncidentTransition, risk *aiops.SubjectRisk) []pendingWrite {
	id := t.IncidentID
	inc := &database.AIOpsIncident{
		ID:        id,
		ClusterID: t.ClusterID,
		State:     string(t.ToState),
		Severity:  t.Severity,
		Subject:   t.SubjectKey,
		RootCause: t.RootCause,
		PeakRisk:  t.PeakRisk,
		StartedAt: t.At,
		CreatedAt: t.At,
	}
	if inc.State == "" {
		inc.State = string(aiops.StateWarning)
	}

	writes := []pendingWrite{
		{"创建事件 " + id, func(ctx context.Context) error { return s.repo.CreateIncident(ctx, inc) }},
	}
	writes = append(writes, s.memberWrites(id, risk)...)
	writes = append(writes, s.timelineWrite(id, t.At, aiops.TimelineStateChange, t.RootCause, "状态变更: healthy → warning"))
	writes = append(writes, s.leadSignalWrites(id, t.At, risk)...)

	log.Info("事件已创建", "id", id, "subject", t.SubjectKey, "rootCause", t.RootCause, "severity", t.Severity)
	return writes
}

// escalatedWrites 警告 → 事件中：状态、峰值严重度、首次升级时间、根因与成员、时间线
func (s *Store) escalatedWrites(t aiops.IncidentTransition, risk *aiops.SubjectRisk) []pendingWrite {
	id := t.IncidentID
	writes := []pendingWrite{
		s.stateWrite(id, aiops.StateIncident),
		s.severityWrite(t),
		{"记录升级时间 " + id, func(ctx context.Context) error { return s.repo.MarkEscalated(ctx, id, t.At) }},
	}
	writes = append(writes, s.leadWrites(t, risk)...)
	writes = append(writes, s.timelineWrite(id, t.At, aiops.TimelineStateChange, t.RootCause,
		fmt.Sprintf("状态变更 → %s (风险 %s)", aiops.StateIncident, displayRisk(t.RFinal))))
	return append(writes, s.leadSignalWrites(id, t.At, risk)...)
}

// severityRaisedWrites 严重度升高一档以上：峰值严重度、根因与成员、时间线；状态不变
func (s *Store) severityRaisedWrites(t aiops.IncidentTransition, risk *aiops.SubjectRisk) []pendingWrite {
	id := t.IncidentID
	writes := []pendingWrite{s.severityWrite(t)}
	writes = append(writes, s.leadWrites(t, risk)...)
	writes = append(writes, s.timelineWrite(id, t.At, aiops.TimelineSeverityRaised, t.RootCause,
		fmt.Sprintf("严重度升高 → %s (风险 %s)", t.Severity, displayRisk(t.RFinal))))
	return append(writes, s.leadSignalWrites(id, t.At, risk)...)
}

// recoveryStartedWrites → 恢复观察：状态、至今的峰值风险、时间线。成员不动（这一轮的风险已经回落）
func (s *Store) recoveryStartedWrites(t aiops.IncidentTransition) []pendingWrite {
	id := t.IncidentID
	return []pendingWrite{
		s.stateWrite(id, aiops.StateRecovery),
		s.peakRiskWrite(t),
		// 独立类型：异常时长据此结构化识别（文案前缀不变，旧数据仍靠它兼容）
		s.timelineWrite(id, t.At, aiops.TimelineRecoveryStarted, t.RootCause,
			fmt.Sprintf("状态变更 → %s (风险 %s)", aiops.StateRecovery, displayRisk(t.RFinal))),
	}
}

// recurredWrites 恢复观察 → 警告：复发次数、状态、峰值严重度、根因与成员、时间线
func (s *Store) recurredWrites(t aiops.IncidentTransition, risk *aiops.SubjectRisk) []pendingWrite {
	id := t.IncidentID
	writes := []pendingWrite{
		{"递增复发次数 " + id, func(ctx context.Context) error { return s.repo.IncrementRecurrence(ctx, id) }},
		s.stateWrite(id, aiops.StateWarning),
		s.severityWrite(t),
	}
	writes = append(writes, s.leadWrites(t, risk)...)
	writes = append(writes, s.timelineWrite(id, t.At, aiops.TimelineRecurrence, t.RootCause,
		fmt.Sprintf("复发 (风险 %s)", displayRisk(t.RFinal))))
	return append(writes, s.leadSignalWrites(id, t.At, risk)...)
}

// closedWrites 关闭：至今的峰值风险、结束时间与关闭原因、时间线。严重度是峰值，关闭时保留
func (s *Store) closedWrites(t aiops.IncidentTransition, reason, detail string) []pendingWrite {
	id := t.IncidentID
	return []pendingWrite{
		s.peakRiskWrite(t),
		{"关闭事件 " + id, func(ctx context.Context) error { return s.repo.Resolve(ctx, id, t.At, reason) }},
		s.timelineWrite(id, t.At, aiops.TimelineStateChange, t.RootCause, detail),
	}
}

// ==================== 单步写入 ====================

func (s *Store) stateWrite(incidentID string, state aiops.EntityState) pendingWrite {
	return pendingWrite{"更新事件状态 " + incidentID, func(ctx context.Context) error {
		return s.repo.UpdateState(ctx, incidentID, string(state))
	}}
}

// severityWrite 抬高峰值严重度与峰值风险（库里只升不降）
func (s *Store) severityWrite(t aiops.IncidentTransition) pendingWrite {
	return pendingWrite{"更新峰值严重度 " + t.IncidentID, func(ctx context.Context) error {
		return s.repo.RaiseSeverity(ctx, t.IncidentID, t.Severity, t.PeakRisk)
	}}
}

func (s *Store) peakRiskWrite(t aiops.IncidentTransition) pendingWrite {
	return pendingWrite{"更新峰值风险 " + t.IncidentID, func(ctx context.Context) error {
		return s.repo.UpdatePeakRisk(ctx, t.IncidentID, t.PeakRisk)
	}}
}

// leadWrites 风险抬升类变化共有的两步：事件的根因跟到当前领头、成员写入实体表
func (s *Store) leadWrites(t aiops.IncidentTransition, risk *aiops.SubjectRisk) []pendingWrite {
	var writes []pendingWrite
	if t.RootCause != "" {
		writes = append(writes, pendingWrite{"更新根因 " + t.IncidentID, func(ctx context.Context) error {
			return s.repo.UpdateRootCause(ctx, t.IncidentID, t.RootCause)
		}})
	}
	return append(writes, s.memberWrites(t.IncidentID, risk)...)
}

// leadSignalWrites 一条「异常检测」时间线：写明领头实体由哪些信号推动
func (s *Store) leadSignalWrites(incidentID string, at time.Time, risk *aiops.SubjectRisk) []pendingWrite {
	if risk == nil || risk.Lead == nil {
		return nil
	}
	return []pendingWrite{s.signalTimelineWrite(incidentID, at, risk.Lead)}
}

// memberWrites 把本轮有风险的成员写进事件实体表：领头为 root_cause，其余为 affected。
//
// 已在表里的成员更新为本轮的风险值，新出现的成员补进去，不在本轮的成员保留原样；
// 之前的领头若已不是领头（Pod 重建换了名字等），降为 affected，表里始终只有一个 root_cause。
//
// 成员的值在这里就拷贝下来：写入可能被推迟重试，届时评分器里的对象已是下一轮的了。
func (s *Store) memberWrites(incidentID string, risk *aiops.SubjectRisk) []pendingWrite {
	if risk == nil || len(risk.Members) == 0 {
		return nil
	}

	leadEntity := ""
	if risk.Lead != nil && risk.Lead.RFinal > 0 {
		leadEntity = risk.Lead.EntityKey
	}
	members := make([]*database.AIOpsIncidentEntity, len(risk.Members))
	for i, member := range risk.Members {
		role := roleAffected
		if member.EntityKey == leadEntity {
			role = roleRootCause
		}
		entityType := member.EntityType
		if entityType == "" {
			entityType = aiops.ExtractEntityType(member.EntityKey)
		}
		members[i] = &database.AIOpsIncidentEntity{
			IncidentID: incidentID,
			EntityKey:  member.EntityKey,
			EntityType: entityType,
			RLocal:     member.RLocal,
			RFinal:     member.RFinal,
			Role:       role,
		}
	}

	return []pendingWrite{{"写入事件成员 " + incidentID, func(ctx context.Context) error {
		if leadEntity != "" {
			existing, err := s.repo.GetEntities(ctx, incidentID)
			if err != nil {
				return err
			}
			for _, entity := range existing {
				if entity.Role != roleRootCause || entity.EntityKey == leadEntity {
					continue
				}
				entity.Role = roleAffected
				if err := s.repo.AddEntity(ctx, entity); err != nil { // (incident_id, entity_key) 主键覆盖写
					return err
				}
			}
		}
		for _, member := range members {
			if err := s.repo.AddEntity(ctx, member); err != nil {
				return err
			}
		}
		return nil
	}}}
}

// atlhyper_master_v2/aiops/statemachine/trigger.go
// 状态评估 + 转换触发 + 事件变化的组装
package statemachine

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Evaluate 评估一个集群里所有主体的状态转换
func (sm *StateMachine) Evaluate(
	ctx context.Context,
	clusterID string,
	subjectRisks map[string]*aiops.SubjectRisk,
	clusterRisk *aiops.ClusterRisk,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()

	for subjectKey, risk := range subjectRisks {
		if risk == nil {
			continue
		}
		entry := sm.getOrCreate(subjectRef{clusterID, subjectKey})
		entry.LastRFinal = risk.RFinal
		entry.LastEvaluatedAt = now.Unix()
		entry.GoneSince = 0

		sm.evaluateSubject(ctx, entry, risk, clusterRisk, now)
	}

	sm.handleGoneSubjects(ctx, clusterID, subjectRisks, now)
}

// handleGoneSubjects 本集群里这一轮评估没出现的主体：对象已被删除。
//
// 能跑到这里说明 Agent 还在上报，所以「不在」本身就是确定的信息：
//   - 刷新 LastEvaluatedAt，不让它被当成失联关闭（失联只留给 Agent 断连）
//   - 连续消失满 goneHold：警告 / 事件中的进入恢复观察（标明对象已删除）；没有事件的条目直接清掉；
//     已在恢复观察的照常等观察期满
//
// 这一轮一个主体都没有时（快照异常）不做判断
func (sm *StateMachine) handleGoneSubjects(ctx context.Context, clusterID string, subjectRisks map[string]*aiops.SubjectRisk, now time.Time) {
	if len(subjectRisks) == 0 {
		return
	}
	for ref, entry := range sm.entries {
		if ref.clusterID != clusterID {
			continue
		}
		if _, present := subjectRisks[ref.subjectKey]; present {
			continue
		}
		entry.LastEvaluatedAt = now.Unix()
		if entry.GoneSince == 0 {
			entry.GoneSince = now.Unix()
		}
		if now.Sub(time.Unix(entry.GoneSince, 0)) < goneHold {
			continue
		}
		switch {
		case entry.IncidentID == "":
			delete(sm.entries, ref)
		case entry.CurrentState == aiops.StateWarning || entry.CurrentState == aiops.StateIncident:
			from := entry.CurrentState
			entry.CurrentState = aiops.StateRecovery
			entry.RecoveryStartedAt = now.Unix()
			entry.ConditionMetSince, entry.ConditionTarget = 0, ""
			sm.emit(ctx, aiops.TransitionRecoveryStarted, entry, from, nil, now, func(t *aiops.IncidentTransition) { t.SubjectGone = true })
			log.Info("主体的对象已删除，进入恢复观察", "cluster", clusterID, "subject", ref.subjectKey, "incident", entry.IncidentID)
		}
	}
}

// exitCondition 当前状态下正在满足的出口条件
type exitCondition struct {
	to          aiops.EntityState
	minDuration time.Duration
}

// matchExitCondition 当前状态下第一个满足的出口条件；都不满足返回 nil。
// 警告有两个出口，先看恢复再看升级：风险已回落时，集群风险再高也不升级这个主体。
func matchExitCondition(entry *aiops.StateMachineEntry, rFinal float64, clusterRisk *aiops.ClusterRisk) *exitCondition {
	switch entry.CurrentState {
	case aiops.StateHealthy:
		if rFinal > riskRaised {
			return &exitCondition{aiops.StateWarning, raisedHold}
		}
	case aiops.StateWarning:
		if rFinal < riskCleared {
			return &exitCondition{aiops.StateRecovery, warningClearedHold}
		}
		if rFinal > riskEscalated || (clusterRisk != nil && clusterRisk.Risk > clusterRiskEscalated) {
			return &exitCondition{aiops.StateIncident, escalatedHold}
		}
	case aiops.StateIncident:
		if rFinal < riskCleared {
			return &exitCondition{aiops.StateRecovery, incidentClearedHold}
		}
	case aiops.StateRecovery:
		if rFinal > riskRaised {
			if entry.ReachedIncident {
				return &exitCondition{aiops.StateWarning, 0} // 升级过的事件：复发立即生效
			}
			return &exitCondition{aiops.StateWarning, raisedHold}
		}
	}
	return nil
}

// evaluateSubject 评估单个主体：先看状态转换，没有转换再看严重度是否升档
func (sm *StateMachine) evaluateSubject(
	ctx context.Context,
	entry *aiops.StateMachineEntry,
	risk *aiops.SubjectRisk,
	clusterRisk *aiops.ClusterRisk,
	now time.Time,
) {
	if entry.IncidentID != "" && risk.RFinal > entry.PeakRisk {
		entry.PeakRisk = risk.RFinal
	}

	cond := matchExitCondition(entry, risk.RFinal, clusterRisk)
	if cond == nil {
		entry.ConditionMetSince, entry.ConditionTarget = 0, ""
	} else {
		if entry.ConditionMetSince == 0 || entry.ConditionTarget != cond.to {
			entry.ConditionMetSince, entry.ConditionTarget = now.Unix(), cond.to
		}
		if now.Sub(time.Unix(entry.ConditionMetSince, 0)) >= cond.minDuration {
			sm.transition(ctx, entry, risk, cond.to, now)
			entry.ConditionMetSince, entry.ConditionTarget = 0, ""
			return
		}
	}

	// 严重度是峰值：警告 / 事件中时，本轮风险落入更高一档就升档
	if entry.CurrentState == aiops.StateWarning || entry.CurrentState == aiops.StateIncident {
		if raiseSeverity(entry, risk) {
			entry.RootCause = leadKey(risk, entry.RootCause)
			sm.emit(ctx, aiops.TransitionSeverityRaised, entry, entry.CurrentState, risk, now)
		}
	}
}

// transition 执行状态转换并发出对应的事件变化
func (sm *StateMachine) transition(
	ctx context.Context,
	entry *aiops.StateMachineEntry,
	risk *aiops.SubjectRisk,
	to aiops.EntityState,
	now time.Time,
) {
	from := entry.CurrentState
	entry.CurrentState = to

	var kind aiops.TransitionKind
	switch {
	case from == aiops.StateHealthy && to == aiops.StateWarning:
		kind = aiops.TransitionCreated
		entry.IncidentID = sm.newIncidentID(now)
		entry.PeakRisk = risk.RFinal
		entry.Severity = ""
		entry.ReachedIncident = false
		entry.Recurrence = 0
		entry.OccurrenceTimes = []int64{now.Unix()}
		entry.EpisodeStartedAt = now.Unix()

	case to == aiops.StateIncident:
		kind = aiops.TransitionEscalated
		entry.ReachedIncident = true

	case to == aiops.StateRecovery:
		kind = aiops.TransitionRecoveryStarted
		entry.RecoveryStartedAt = now.Unix()

	case from == aiops.StateRecovery && to == aiops.StateWarning:
		kind = aiops.TransitionRecurred
		entry.RecoveryStartedAt = 0 // 复发：观察期作废，下次恢复重新计
		entry.Recurrence++
		entry.OccurrenceTimes = append(recentOccurrences(entry.OccurrenceTimes, now), now.Unix())
		entry.EpisodeStartedAt = now.Unix()
	}

	if kind != aiops.TransitionRecoveryStarted {
		// 风险抬升类的转换：严重度升档并在这条变化里带出，不另发 severity_raised
		raiseSeverity(entry, risk)
		entry.RootCause = leadKey(risk, entry.RootCause)
	}

	sm.emit(ctx, kind, entry, from, risk, now)

	log.Info("状态转换",
		"cluster", entry.ClusterID,
		"subject", entry.SubjectKey,
		"incident", entry.IncidentID,
		"from", from,
		"to", to,
		"rFinal", risk.RFinal,
	)
}

// raiseSeverity 本轮风险对应的严重度高于已记录的峰值时升档，返回是否升了
func raiseSeverity(entry *aiops.StateMachineEntry, risk *aiops.SubjectRisk) bool {
	current := aiops.SeverityFromRisk(risk.RFinal)
	if aiops.SeverityRank(current) <= aiops.SeverityRank(entry.Severity) {
		return false
	}
	entry.Severity = current
	return true
}

// leadKey 本轮的领头实体；本轮没有带风险的领头时沿用之前记下的
func leadKey(risk *aiops.SubjectRisk, previous string) string {
	if risk.Lead != nil && risk.Lead.RFinal > 0 {
		return risk.Lead.EntityKey
	}
	return previous
}

// recentOccurrences 去掉滑出 24 小时窗口的发生时刻
func recentOccurrences(times []int64, now time.Time) []int64 {
	cutoff := now.Add(-occurrenceWindow).Unix()
	recent := make([]int64, 0, len(times)+1)
	for _, at := range times {
		if at > cutoff {
			recent = append(recent, at)
		}
	}
	return recent
}

// emit 组装一条字段齐全的事件变化交给处理方。risk 为 nil 表示没有对应的评估（关闭类）
func (sm *StateMachine) emit(
	ctx context.Context,
	kind aiops.TransitionKind,
	entry *aiops.StateMachineEntry,
	from aiops.EntityState,
	risk *aiops.SubjectRisk,
	now time.Time,
	adjust ...func(*aiops.IncidentTransition),
) {
	entry.OccurrenceTimes = recentOccurrences(entry.OccurrenceTimes, now)

	transition := aiops.IncidentTransition{
		Kind:            kind,
		IncidentID:      entry.IncidentID,
		ClusterID:       entry.ClusterID,
		SubjectKey:      entry.SubjectKey,
		RootCause:       entry.RootCause,
		FromState:       from,
		ToState:         entry.CurrentState,
		Severity:        entry.Severity,
		RFinal:          entry.LastRFinal,
		PeakRisk:        entry.PeakRisk,
		Recurrence:      entry.Recurrence,
		Occurrences24h:  len(entry.OccurrenceTimes),
		ReachedIncident: entry.ReachedIncident,
		At:              now,
	}
	if entry.EpisodeStartedAt != 0 {
		transition.EpisodeStartedAt = time.Unix(entry.EpisodeStartedAt, 0).In(now.Location())
	}
	if risk != nil {
		transition.RFinal = risk.RFinal
		for _, member := range risk.Members {
			transition.Members = append(transition.Members, member.EntityKey)
		}
	}

	for _, fn := range adjust {
		fn(&transition)
	}
	sm.handler.OnTransition(ctx, transition, risk)
}

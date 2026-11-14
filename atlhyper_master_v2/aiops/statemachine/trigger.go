// atlhyper_master_v2/aiops/statemachine/trigger.go
// 状态评估 + 转换触发
package statemachine

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Evaluate 评估所有实体的状态转换
func (sm *StateMachine) Evaluate(
	ctx context.Context,
	clusterID string,
	entityRisks map[string]*aiops.EntityRisk,
	clusterRisk *aiops.ClusterRisk,
) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()

	for entityKey, risk := range entityRisks {
		ref := aiops.EntityRef{ClusterID: clusterID, EntityKey: entityKey}
		entry := sm.getOrCreate(ref)
		entry.LastRFinal = risk.RFinal
		entry.LastEvaluatedAt = now.Unix()

		sm.evaluateEntity(ctx, ref, entry, risk, clusterRisk, now)
	}
}

// evaluateEntity 评估单个实体的状态转换
// 对同一状态的多个出口条件，只匹配第一个满足的条件
func (sm *StateMachine) evaluateEntity(
	ctx context.Context,
	ref aiops.EntityRef,
	entry *aiops.StateMachineEntry,
	risk *aiops.EntityRisk,
	clusterRisk *aiops.ClusterRisk,
	now time.Time,
) {
	// 查找当前状态下第一个满足的转换条件
	var matched *transitionCondition
	for i := range sm.conditions {
		cond := &sm.conditions[i]
		if entry.CurrentState != cond.FromState {
			continue
		}

		conditionMet := cond.RiskCheck(risk.RFinal)

		// 特殊处理: Warning → Incident 也可由 ClusterRisk > 80 触发
		if cond.FromState == aiops.StateWarning && cond.ToState == aiops.StateIncident {
			if clusterRisk != nil && clusterRisk.Risk > 80 {
				conditionMet = true
			}
		}

		if conditionMet {
			matched = cond
			break
		}
	}

	if matched == nil {
		entry.ConditionMetSince = 0
		return
	}

	if entry.ConditionMetSince == 0 {
		entry.ConditionMetSince = now.Unix()
	}

	duration := time.Duration(now.Unix()-entry.ConditionMetSince) * time.Second
	if duration >= matched.MinDuration {
		sm.transition(ctx, ref, entry, risk, *matched, now)
		entry.ConditionMetSince = 0
	}
}

// transition 执行状态转换
func (sm *StateMachine) transition(
	ctx context.Context,
	ref aiops.EntityRef,
	entry *aiops.StateMachineEntry,
	risk *aiops.EntityRisk,
	cond transitionCondition,
	now time.Time,
) {
	oldState := entry.CurrentState
	entry.CurrentState = cond.ToState

	switch {
	case oldState == aiops.StateHealthy && cond.ToState == aiops.StateWarning:
		incidentID := sm.callback.OnWarningCreated(ctx, ref.ClusterID, ref.EntityKey, risk, now)
		entry.IncidentID = incidentID

	case oldState == aiops.StateWarning && cond.ToState == aiops.StateHealthy:
		sm.callback.OnStable(ctx, entry.IncidentID, ref.EntityKey, now)
		delete(sm.entries, ref)

	case oldState == aiops.StateWarning && cond.ToState == aiops.StateIncident:
		sm.callback.OnStateEscalated(ctx, entry.IncidentID, aiops.StateIncident, risk, now)

	case oldState == aiops.StateIncident && cond.ToState == aiops.StateRecovery:
		entry.RecoveryStartedAt = now.Unix()
		sm.callback.OnRecoveryStarted(ctx, entry.IncidentID, risk, now)

	case oldState == aiops.StateRecovery && cond.ToState == aiops.StateWarning:
		entry.RecoveryStartedAt = 0 // 复发：观察期作废，下次恢复重新计
		sm.callback.OnRecurrence(ctx, entry.IncidentID, risk, now)
	}

	log.Info("状态转换",
		"cluster", ref.ClusterID,
		"entity", ref.EntityKey,
		"from", oldState,
		"to", cond.ToState,
		"rFinal", risk.RFinal,
	)
}

// CheckRecoveryToStable 检查 Recovery 状态的实体是否可以转为 Stable
// 条件: 进入恢复观察期满 48h 且期间没有复发（复发会离开 Recovery 并清空 RecoveryStartedAt）
//
// 计时用 RecoveryStartedAt 而不是 ConditionMetSince：观察期内实体仍在每轮被评估，
// 而评估在「没有出口条件满足」时会把 ConditionMetSince 清零。
func (sm *StateMachine) CheckRecoveryToStable(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()
	stableThreshold := 48 * time.Hour

	for ref, entry := range sm.entries {
		if entry.CurrentState != aiops.StateRecovery {
			continue
		}

		if entry.RecoveryStartedAt == 0 {
			// 恢复时刻未知（重启后时间线里查不到）：从现在起计
			entry.RecoveryStartedAt = now.Unix()
			continue
		}

		duration := time.Duration(now.Unix()-entry.RecoveryStartedAt) * time.Second
		if duration < stableThreshold {
			continue
		}

		entry.CurrentState = aiops.StateStable
		sm.callback.OnStable(ctx, entry.IncidentID, ref.EntityKey, now)

		log.Info("事件已关闭", "cluster", ref.ClusterID, "entity", ref.EntityKey, "incident", entry.IncidentID)
		delete(sm.entries, ref)
	}
}

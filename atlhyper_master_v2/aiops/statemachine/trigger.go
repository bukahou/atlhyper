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

	now := time.Now()

	for entityKey, risk := range entityRisks {
		entry := sm.getOrCreate(entityKey)
		entry.LastRFinal = risk.RFinal
		entry.LastEvaluatedAt = now.Unix()

		sm.evaluateEntity(ctx, clusterID, entry, risk, clusterRisk, now)
	}
}

// evaluateEntity 评估单个实体的状态转换
// 对同一状态的多个出口条件，只匹配第一个满足的条件
func (sm *StateMachine) evaluateEntity(
	ctx context.Context,
	clusterID string,
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
		sm.transition(ctx, clusterID, entry, risk, *matched, now)
		entry.ConditionMetSince = 0
	}
}

// transition 执行状态转换
func (sm *StateMachine) transition(
	ctx context.Context,
	clusterID string,
	entry *aiops.StateMachineEntry,
	risk *aiops.EntityRisk,
	cond transitionCondition,
	now time.Time,
) {
	oldState := entry.CurrentState
	entry.CurrentState = cond.ToState

	switch {
	case oldState == aiops.StateHealthy && cond.ToState == aiops.StateWarning:
		incidentID := sm.callback.OnWarningCreated(ctx, clusterID, entry.EntityKey, risk, now)
		entry.IncidentID = incidentID

	case oldState == aiops.StateWarning && cond.ToState == aiops.StateHealthy:
		sm.callback.OnStable(ctx, entry.IncidentID, entry.EntityKey, now)
		delete(sm.entries, entry.EntityKey)

	case oldState == aiops.StateWarning && cond.ToState == aiops.StateIncident:
		sm.callback.OnStateEscalated(ctx, entry.IncidentID, aiops.StateIncident, risk, now)

	case oldState == aiops.StateIncident && cond.ToState == aiops.StateRecovery:
		sm.callback.OnRecoveryStarted(ctx, entry.IncidentID, risk, now)

	case oldState == aiops.StateRecovery && cond.ToState == aiops.StateWarning:
		sm.callback.OnRecurrence(ctx, entry.IncidentID, risk, now)
	}

	log.Info("状态转换",
		"entity", entry.EntityKey,
		"from", oldState,
		"to", cond.ToState,
		"rFinal", risk.RFinal,
	)
}

// CheckRecoveryToStable 检查 Recovery 状态的实体是否可以转为 Stable
// 条件: 48h 内 R_final 未再 > 0.5
func (sm *StateMachine) CheckRecoveryToStable(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	stableThreshold := 48 * time.Hour

	for entityKey, entry := range sm.entries {
		if entry.CurrentState != aiops.StateRecovery {
			continue
		}

		if entry.ConditionMetSince == 0 {
			entry.ConditionMetSince = now.Unix()
			continue
		}

		duration := time.Duration(now.Unix()-entry.ConditionMetSince) * time.Second
		if duration < stableThreshold {
			continue
		}

		entry.CurrentState = aiops.StateStable
		sm.callback.OnStable(ctx, entry.IncidentID, entityKey, now)

		log.Info("事件已关闭", "entity", entityKey, "incident", entry.IncidentID)
		delete(sm.entries, entityKey)
	}
}

// atlhyper_master_v2/aiops/statemachine/machine.go
// 状态机管理器: 管理实体状态转换 + 转换回调
package statemachine

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps.SM")

// TransitionCallback 状态转换回调接口
// 解耦 StateMachine 和 incident.Store，由 Engine 实现此接口
type TransitionCallback interface {
	OnWarningCreated(ctx context.Context, clusterID, entityKey string, risk *aiops.EntityRisk, now time.Time) string
	OnStateEscalated(ctx context.Context, incidentID string, state aiops.EntityState, risk *aiops.EntityRisk, now time.Time)
	OnRecoveryStarted(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time)
	OnRecurrence(ctx context.Context, incidentID string, risk *aiops.EntityRisk, now time.Time)
	OnStable(ctx context.Context, incidentID string, entityKey string, now time.Time)
}

// transitionCondition 状态转换条件
type transitionCondition struct {
	FromState   aiops.EntityState
	ToState     aiops.EntityState
	RiskCheck   func(rFinal float64) bool
	MinDuration time.Duration
}

// StateMachine 状态机管理器
type StateMachine struct {
	mu         sync.RWMutex
	entries    map[string]*aiops.StateMachineEntry // entityKey -> entry
	callback   TransitionCallback
	conditions []transitionCondition
}

// NewStateMachine 创建状态机
func NewStateMachine(callback TransitionCallback) *StateMachine {
	sm := &StateMachine{
		entries:  make(map[string]*aiops.StateMachineEntry),
		callback: callback,
	}
	sm.conditions = []transitionCondition{
		{
			FromState:   aiops.StateHealthy,
			ToState:     aiops.StateWarning,
			RiskCheck:   func(r float64) bool { return r > 0.2 },
			MinDuration: 2 * time.Minute,
		},
		// Warning → Healthy（风险持续低于阈值 5 分钟 → 自动恢复）
		// 顺序重要：放在 Warning→Incident 之前，低风险时先匹配 →Healthy
		{
			FromState:   aiops.StateWarning,
			ToState:     aiops.StateHealthy,
			RiskCheck:   func(r float64) bool { return r < 0.15 },
			MinDuration: 5 * time.Minute,
		},
		{
			FromState:   aiops.StateWarning,
			ToState:     aiops.StateIncident,
			RiskCheck:   func(r float64) bool { return r > 0.5 },
			MinDuration: 5 * time.Minute,
		},
		{
			FromState:   aiops.StateIncident,
			ToState:     aiops.StateRecovery,
			RiskCheck:   func(r float64) bool { return r < 0.15 },
			MinDuration: 10 * time.Minute,
		},
		{
			FromState:   aiops.StateRecovery,
			ToState:     aiops.StateWarning,
			RiskCheck:   func(r float64) bool { return r > 0.2 },
			MinDuration: 0, // 复发立即触发
		},
	}
	return sm
}

// GetEntry 获取指定实体的状态机条目
func (sm *StateMachine) GetEntry(entityKey string) *aiops.StateMachineEntry {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.entries[entityKey]
}

// RestoreEntry 从外部注入状态机条目（启动恢复用）
func (sm *StateMachine) RestoreEntry(entry *aiops.StateMachineEntry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.entries[entry.EntityKey] = entry
}

// CleanupStaleEntries 清理长时间未评估的过期条目
// 实体从集群消失（Pod 被删除）后 SM 条目永远不会被 Evaluate 触达，需要超时自动关闭
func (sm *StateMachine) CleanupStaleEntries(ctx context.Context, staleThreshold time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for entityKey, entry := range sm.entries {
		if entry.LastEvaluatedAt == 0 {
			continue
		}
		age := time.Duration(now.Unix()-entry.LastEvaluatedAt) * time.Second
		if age > staleThreshold {
			entry.CurrentState = aiops.StateStable
			sm.callback.OnStable(ctx, entry.IncidentID, entityKey, now)
			delete(sm.entries, entityKey)
			log.Info("自动关闭过期事件", "entity", entityKey, "incident", entry.IncidentID)
		}
	}
}

// getOrCreate 获取或创建状态机条目
func (sm *StateMachine) getOrCreate(entityKey string) *aiops.StateMachineEntry {
	entry, ok := sm.entries[entityKey]
	if !ok {
		entry = &aiops.StateMachineEntry{
			EntityKey:    entityKey,
			CurrentState: aiops.StateHealthy,
		}
		sm.entries[entityKey] = entry
	}
	return entry
}

// atlhyper_master_v2/aiops/statemachine/machine.go
// 状态机管理器：以主体为键管理事件的状态，每次变化交给 TransitionHandler
package statemachine

import (
	"context"
	"sort"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIOps.SM")

// TransitionHandler 接收状态机产生的每一次事件变化。
//
// 由引擎实现（落库 + 发给订阅者），状态机因此不依赖 incident.Store。
// 调用时持有状态机的锁，所以各次变化严格先后有序；实现里不得回调状态机，也不得阻塞。
type TransitionHandler interface {
	// OnTransition 的 risk 是触发本次变化的那一轮主体风险；
	// 关闭类变化（观察期满、失联）没有对应的评估，risk 为 nil
	OnTransition(ctx context.Context, transition aiops.IncidentTransition, risk *aiops.SubjectRisk)
}

// 转换条件的阈值与持续时间
const (
	riskRaised    = 0.2  // 高于它：健康 → 警告、恢复观察 → 警告（复发）
	riskEscalated = 0.5  // 高于它：警告 → 事件中
	riskCleared   = 0.15 // 低于它：警告 / 事件中 → 恢复观察
	// clusterRiskEscalated 集群整体风险高于它时，警告中的主体不论自身风险都升级
	clusterRiskEscalated = 80.0

	raisedHold          = 2 * time.Minute  // 建事件，以及没升级过的事件复发
	escalatedHold       = 5 * time.Minute  // 升级
	warningClearedHold  = 5 * time.Minute  // 警告 → 恢复观察
	incidentClearedHold = 10 * time.Minute // 事件中 → 恢复观察

	// 恢复观察期：期满且期间没有复发才关闭。升级过的事件观察得更久
	observationEscalated      = 48 * time.Hour
	observationNeverEscalated = 6 * time.Hour

	// goneHold 主体连续不在评估里多久才认定对象已删除
	goneHold = 2 * time.Minute

	// occurrenceWindow 「近 24 小时发生次数」的窗口
	occurrenceWindow = 24 * time.Hour
)

// subjectRef 指向某个集群里的某个主体（主体键只在集群内唯一）
type subjectRef struct {
	clusterID  string
	subjectKey string
}

// StateMachine 状态机管理器
type StateMachine struct {
	mu            sync.RWMutex
	entries       map[subjectRef]*aiops.StateMachineEntry
	handler       TransitionHandler
	newIncidentID func(now time.Time) string
	now           func() time.Time // 可替换的时钟，测试用它模拟时间流逝
}

// NewStateMachine 创建状态机。newIncidentID 在建事件时分配事件号（由事件存储一侧提供）
func NewStateMachine(handler TransitionHandler, newIncidentID func(now time.Time) string) *StateMachine {
	return &StateMachine{
		entries:       make(map[subjectRef]*aiops.StateMachineEntry),
		handler:       handler,
		newIncidentID: newIncidentID,
		now:           time.Now,
	}
}

// GetEntry 获取指定主体的状态机条目
func (sm *StateMachine) GetEntry(clusterID, subjectKey string) *aiops.StateMachineEntry {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.entries[subjectRef{clusterID, subjectKey}]
}

// ActiveIncidents 某个集群里处于警告 / 事件中的条目（副本，按主体键排序）
func (sm *StateMachine) ActiveIncidents(clusterID string) []aiops.StateMachineEntry {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	var active []aiops.StateMachineEntry
	for ref, entry := range sm.entries {
		if ref.clusterID != clusterID || entry.IncidentID == "" {
			continue
		}
		if entry.CurrentState == aiops.StateWarning || entry.CurrentState == aiops.StateIncident {
			active = append(active, *entry)
		}
	}
	sort.Slice(active, func(i, j int) bool { return active[i].SubjectKey < active[j].SubjectKey })
	return active
}

// RestoreEntry 从外部注入状态机条目（启动恢复用）
func (sm *StateMachine) RestoreEntry(entry *aiops.StateMachineEntry) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.entries[subjectRef{entry.ClusterID, entry.SubjectKey}] = entry
}

// CheckRecoveryToStable 关闭观察期已满的事件。
// 条件：进入恢复观察期满（升级过 48 小时，没升级过 6 小时）且期间没有复发
// （复发会离开恢复观察并清空 RecoveryStartedAt）。
//
// 计时用 RecoveryStartedAt 而不是 ConditionMetSince：观察期内主体仍在每轮被评估，
// 而评估在「没有出口条件满足」时会把 ConditionMetSince 清零。
func (sm *StateMachine) CheckRecoveryToStable(ctx context.Context) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()
	for ref, entry := range sm.entries {
		if entry.CurrentState != aiops.StateRecovery {
			continue
		}
		if entry.RecoveryStartedAt == 0 {
			// 恢复时刻未知（重启后时间线里查不到）：从现在起计
			entry.RecoveryStartedAt = now.Unix()
			continue
		}
		if now.Sub(time.Unix(entry.RecoveryStartedAt, 0)) < observationPeriod(entry) {
			continue
		}

		sm.close(ctx, ref, entry, aiops.TransitionClosed, now)
		log.Info("事件已关闭（观察期满）", "cluster", ref.clusterID, "subject", ref.subjectKey, "incident", entry.IncidentID)
	}
}

// CleanupStaleEntries 清理长时间没有被评估的条目。
// 主体从集群消失（对象已删除）或 Agent 断连后，条目不会再被 Evaluate 触达；
// 带着事件的按「失联」关闭，与观察期满的关闭区分开。
func (sm *StateMachine) CleanupStaleEntries(ctx context.Context, staleThreshold time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := sm.now()
	for ref, entry := range sm.entries {
		if entry.LastEvaluatedAt == 0 {
			continue
		}
		if now.Sub(time.Unix(entry.LastEvaluatedAt, 0)) <= staleThreshold {
			continue
		}
		if entry.IncidentID == "" {
			delete(sm.entries, ref) // 一直健康、没有事件的主体：只清条目
			continue
		}

		sm.close(ctx, ref, entry, aiops.TransitionClosedStale, now)
		log.Info("事件已关闭（失联）", "cluster", ref.clusterID, "subject", ref.subjectKey, "incident", entry.IncidentID)
	}
}

// close 关闭条目上的事件并删除条目
func (sm *StateMachine) close(ctx context.Context, ref subjectRef, entry *aiops.StateMachineEntry, kind aiops.TransitionKind, now time.Time) {
	from := entry.CurrentState
	entry.CurrentState = aiops.StateStable
	sm.emit(ctx, kind, entry, from, nil, now)
	delete(sm.entries, ref)
}

// observationPeriod 恢复观察期的长度
func observationPeriod(entry *aiops.StateMachineEntry) time.Duration {
	if entry.ReachedIncident {
		return observationEscalated
	}
	return observationNeverEscalated
}

// getOrCreate 获取或创建状态机条目
func (sm *StateMachine) getOrCreate(ref subjectRef) *aiops.StateMachineEntry {
	entry, ok := sm.entries[ref]
	if !ok {
		entry = &aiops.StateMachineEntry{
			ClusterID:    ref.clusterID,
			SubjectKey:   ref.subjectKey,
			CurrentState: aiops.StateHealthy,
		}
		sm.entries[ref] = entry
	}
	return entry
}

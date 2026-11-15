// atlhyper_master_v2/aiops/core/signal_recorder.go
// 事件进行中（警告 / 事件中）按轮累计成员的信号，作为详情页的关键证据。
//
// 每次快照都会评估一轮，但证据不需要那么密：同一个事件最多每分钟记一次
package core

import (
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// signalRecordInterval 同一个事件两次累计证据的最小间隔
const signalRecordInterval = time.Minute

// signalThrottle 记下每个事件上次累计证据的时刻
type signalThrottle struct {
	mu   sync.Mutex
	last map[string]time.Time // incidentID → 上次累计的时刻
}

// due 这个事件现在该不该累计；该的话顺便记下这次的时刻
func (t *signalThrottle) due(incidentID string, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.last == nil {
		t.last = map[string]time.Time{}
	}
	if last, ok := t.last[incidentID]; ok && now.Sub(last) < signalRecordInterval {
		return false
	}
	t.last[incidentID] = now
	// 关闭的事件不会再来：顺手清掉一小时没动过的记录，免得越积越多
	for id, at := range t.last {
		if now.Sub(at) > time.Hour {
			delete(t.last, id)
		}
	}
	return true
}

// recordActiveSignals 对本轮仍在警告 / 事件中的事件累计成员信号
func (e *engine) recordActiveSignals(clusterID string, subjectRisks map[string]*aiops.SubjectRisk, now time.Time) {
	if e.incidentStore == nil || e.sm == nil {
		return
	}
	for subjectKey, risk := range subjectRisks {
		entry := e.sm.GetEntry(clusterID, subjectKey)
		if entry == nil || entry.IncidentID == "" ||
			(entry.CurrentState != aiops.StateWarning && entry.CurrentState != aiops.StateIncident) {
			continue
		}
		if !e.signalThrottle.due(entry.IncidentID, now) {
			continue
		}
		e.incidentStore.RecordSignals(e.bgCtx, entry.IncidentID, risk.Members, now)
	}
}

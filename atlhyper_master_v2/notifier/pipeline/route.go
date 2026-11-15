// atlhyper_master_v2/notifier/pipeline/route.go
// 路由、静默、抑制、去重、重复提醒：决定一条变化要不要通知，要的话进哪个分组
package pipeline

import (
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/routing"
	"AtlHyper/atlhyper_master_v2/notifier/silence"
)

// routePending 对所有「有变化等着通知」的告警做路由
func (p *AlertPipeline) routePending(current *settings, now time.Time) {
	for _, fingerprint := range p.sortedFingerprints() {
		state := p.alerts[fingerprint]
		if state == nil || state.wantKind == "" {
			continue
		}
		kind, force := state.wantKind, state.force
		state.wantKind, state.force = "", false
		p.route(state, kind, force, current, now)
	}
}

// route 一条变化的去向：不通知，或进入某个分组等待发出
func (p *AlertPipeline) route(state *alertState, kind alert.Kind, force bool, current *settings, now time.Time) {
	state.lastAttemptAt = now
	isResolved := kind == alert.KindResolved
	// skip 这条变化不通知。恢复通知一旦不发，这条已结束的告警就没有别的事了
	skip := func() {
		p.dequeue(state)
		if isResolved {
			p.finalize(state)
		}
	}

	if !current.sourceEnabled(state.alert.Source) {
		skip()
		return
	}
	if isResolved && (!current.policy.NotifyResolved || !state.announced()) {
		skip()
		return
	}
	decision := current.route(&state.alert)
	if decision.BelowMinSeverity {
		skip()
		return
	}
	if p.updateHold(state, current, now) {
		// 被静默或抑制压住：照常落库与显示，不发。压住期间结束的告警不补发触发、也不发恢复
		skip()
		return
	}
	if !force && !isResolved && kind != alert.KindRepeat && p.alreadyNotified(state, kind, decision.Timing, now) {
		return
	}
	p.enqueue(state, groupKey{ruleID: decision.RuleID(), clusterID: state.alert.ClusterID}, kind, decision.Timing, now)
}

// alreadyNotified 去重：同一条告警、同一种变化（同一严重度），在重复间隔内只发一次
func (p *AlertPipeline) alreadyNotified(state *alertState, kind alert.Kind, timing routing.Timing, now time.Time) bool {
	return kind == state.lastNotifiedKind &&
		state.alert.Severity == state.lastNotifiedSeverity &&
		now.Sub(state.lastNotifiedAt) < timing.RepeatInterval
}

// updateHold 重新判断这条告警是否被静默或抑制压住，有变化就落库；返回是否被压住
func (p *AlertPipeline) updateHold(state *alertState, current *settings, now time.Time) bool {
	silencedBy := silence.Match(current.silences, &state.alert, now)
	suppressedBy := ""
	if state.alert.Source != alert.SourceHeartbeat && p.clusterOffline(state.alert.ClusterID) {
		suppressedBy = SuppressedByClusterOffline
	}
	if silencedBy != state.silencedBy || suppressedBy != state.suppressedBy {
		state.silencedBy, state.suppressedBy = silencedBy, suppressedBy
		p.persist(state)
	}
	return state.held()
}

// clusterOffline 该集群的 Agent 离线告警是否正在持续
func (p *AlertPipeline) clusterOffline(clusterID string) bool {
	heartbeat := p.alerts[alert.HeartbeatFingerprint(clusterID)]
	return heartbeat != nil && heartbeat.alert.Status == alert.StatusFiring
}

// refreshHolds 每一轮重新判断全部告警的静默与抑制。
//
//   - 新被压住的：撤掉分组里待发的内容
//   - 刚解除的（静默到期、集群恢复上线）且仍在持续：按新告警发出
func (p *AlertPipeline) refreshHolds(current *settings, now time.Time) {
	for _, fingerprint := range p.sortedFingerprints() {
		state := p.alerts[fingerprint]
		wasHeld := state.held()
		isHeld := p.updateHold(state, current, now)
		firing := state.alert.Status == alert.StatusFiring
		switch {
		case isHeld && firing:
			p.dequeue(state)
			state.wantKind = ""
		case isHeld:
			p.finalize(state) // 恢复通知还没发出去就被压住了：不发
		case wasHeld && firing:
			state.wantKind, state.force = alert.KindFiring, true
		}
	}
}

// scheduleRepeats 持续中的告警，距上一次通知决定满了重复间隔，就再走一次路由。
//
// 通知过的发「重复提醒」；这次持续还没人收到过的（当时没有接收人、不到严重度门槛等）按原来的变化补发
func (p *AlertPipeline) scheduleRepeats(current *settings, now time.Time) {
	for _, fingerprint := range p.sortedFingerprints() {
		state := p.alerts[fingerprint]
		if state.alert.Status != alert.StatusFiring || state.held() || state.queued != nil || state.wantKind != "" {
			continue
		}
		interval := current.route(&state.alert).Timing.RepeatInterval
		if interval <= 0 || now.Sub(state.lastAttemptAt) < interval {
			continue
		}
		if state.announced() {
			state.wantKind = alert.KindRepeat
		} else {
			state.wantKind = state.alert.Kind
		}
	}
}

// enqueue 把一条变化放进分组。告警已经在某个分组里有待发内容时，两者合并成一条
func (p *AlertPipeline) enqueue(state *alertState, key groupKey, kind alert.Kind, timing routing.Timing, now time.Time) {
	fingerprint := state.alert.Fingerprint
	var waiting alert.Kind
	if state.queued != nil {
		if previous := p.groups[*state.queued]; previous != nil {
			waiting = previous.pending[fingerprint]
		}
		if *state.queued != key {
			p.dequeue(state) // 路由结果变了（如升级后命中另一条规则）：挪到新的分组
		}
	}

	group := p.groups[key]
	if group == nil {
		group = &groupState{pending: make(map[string]alert.Kind)}
		p.groups[key] = group
	}
	if len(group.pending) == 0 {
		group.firstPendingAt = now
	}
	if _, exists := group.pending[fingerprint]; !exists {
		group.order = append(group.order, fingerprint)
	}
	group.pending[fingerprint] = mergeKinds(waiting, kind)
	group.timing = timing
	state.queued = &key
}

// mergeKinds 同一条告警有一条变化还没发出去，又来了一条：只发一条，发哪一种。
//
//   - 待发的是「开始」类（触发 / 复发 / 反复），后到的是升级或重复提醒：仍发「开始」——
//     收件人看到的第一条必须是开始，内容在发出时取最新的
//   - 待发的是升级，后到的是重复提醒：仍发升级
//   - 其余情况以后到的为准（如恢复取代升级）
func mergeKinds(waiting, incoming alert.Kind) alert.Kind {
	switch {
	case waiting == "":
		return incoming
	case incoming == alert.KindResolved:
		return incoming
	case waiting.Opens() && !incoming.Opens():
		return waiting
	case waiting == alert.KindEscalated && incoming == alert.KindRepeat:
		return waiting
	default:
		return incoming
	}
}

// dequeue 撤掉这条告警在分组里待发的内容
func (p *AlertPipeline) dequeue(state *alertState) {
	if state.queued == nil {
		return
	}
	fingerprint := state.alert.Fingerprint
	if group := p.groups[*state.queued]; group != nil {
		delete(group.pending, fingerprint)
		for index, queued := range group.order {
			if queued == fingerprint {
				group.order = append(group.order[:index], group.order[index+1:]...)
				break
			}
		}
	}
	state.queued = nil
}

// pruneIdleGroups 清掉没有待发内容、且距上次发出已超过分组间隔的分组（它们对下一次发出的时间已无影响）
func (p *AlertPipeline) pruneIdleGroups(now time.Time) {
	for key, group := range p.groups {
		if len(group.pending) == 0 && now.Sub(group.lastFlushAt) > group.timing.GroupInterval {
			delete(p.groups, key)
		}
	}
}

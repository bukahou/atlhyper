// atlhyper_master_v2/notifier/pipeline/ingest.go
// 入口：把来源提交的告警落成状态，并标出「有什么变化等着通知」
package pipeline

import (
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// apply 把一条提交的告警落到状态上
func (p *AlertPipeline) apply(incoming alert.Alert, now time.Time) {
	state := p.alerts[incoming.Fingerprint]
	if incoming.Status == alert.StatusResolved {
		p.applyResolved(state, incoming, now)
		return
	}

	incoming.Status, incoming.EndsAt, incoming.UpdatedAt, incoming.Quiet = alert.StatusFiring, nil, now, false
	if incoming.Kind == "" || incoming.Kind == alert.KindResolved || incoming.Kind == alert.KindRepeat {
		incoming.Kind = alert.KindFiring
	}
	switch {
	case state == nil:
		p.openEpisode(incoming, now)
	case state.alert.Status == alert.StatusResolved:
		p.reopenBeforeResolvedWasSent(state, incoming)
	default:
		p.updateFiring(state, incoming)
	}
}

// openEpisode 一次新的持续：全新的告警，或结束之后再次触发
func (p *AlertPipeline) openEpisode(incoming alert.Alert, now time.Time) {
	if incoming.StartsAt.IsZero() {
		incoming.StartsAt = now
	}
	state := &alertState{alert: incoming, lastAttemptAt: now, wantKind: incoming.Kind}

	// 同一个指纹以前出现过（已结束）：累计通知次数与上次通知时间接着算
	ctx, cancel := p.storeContext()
	previous, err := p.repos.Alerts.Get(ctx, incoming.Fingerprint)
	cancel()
	if err != nil {
		log.Error("读取历史告警失败，通知次数从零算起", "fingerprint", incoming.Fingerprint, "err", err)
	}
	if previous != nil {
		state.notifyCount = previous.NotifyCount
		if previous.LastNotifiedAt != nil {
			state.lastNotifiedAt = *previous.LastNotifiedAt
		}
	}

	p.alerts[incoming.Fingerprint] = state
	p.persist(state)
}

// reopenBeforeResolvedWasSent 告警结束了、恢复通知还在分组里没发出去，它又触发了。
//
// 收件人眼里它一直在持续：撤掉那条没发出去的恢复，告警接着算作同一次持续，不另发通知 ——
// 否则几十秒内的一出一进会变成「恢复 + 复发」两条
func (p *AlertPipeline) reopenBeforeResolvedWasSent(state *alertState, incoming alert.Alert) {
	p.dequeue(state)
	incoming.StartsAt = state.alert.StartsAt
	state.alert = incoming
	state.wantKind = ""
	p.persist(state)
}

// updateFiring 持续中的告警又收到一次提交：更新内容；只有两种情况算「有变化、要通知」——
// 警告级的反复升级成了真正的事件，或严重度升高了
func (p *AlertPipeline) updateFiring(state *alertState, incoming alert.Alert) {
	previous := state.alert
	promoted := incoming.Kind == alert.KindFiring && previous.Kind == alert.KindFlapping
	raised := incoming.Severity.Rank() > previous.Severity.Rank()

	incoming.StartsAt = previous.StartsAt
	switch {
	case promoted:
		state.wantKind = alert.KindFiring
	case raised:
		incoming.Kind = alert.KindEscalated
		state.wantKind = alert.KindEscalated
	default:
		incoming.Kind = previous.Kind // 同样的事再报一次：只刷新内容
	}
	state.alert = incoming
	p.persist(state)
}

// applyResolved 告警结束。没有在持续的告警时什么也不做
func (p *AlertPipeline) applyResolved(state *alertState, incoming alert.Alert, now time.Time) {
	if state == nil || state.alert.Status != alert.StatusFiring {
		return
	}
	endsAt := now
	if incoming.EndsAt != nil {
		endsAt = *incoming.EndsAt
	}
	state.alert.Status, state.alert.Kind = alert.StatusResolved, alert.KindResolved
	state.alert.EndsAt, state.alert.UpdatedAt = &endsAt, now
	if incoming.Summary != "" {
		state.alert.Summary = incoming.Summary
	}
	if len(incoming.Details) > 0 {
		merged := make(map[string]string, len(state.alert.Details)+len(incoming.Details))
		for key, value := range state.alert.Details {
			merged[key] = value
		}
		for key, value := range incoming.Details {
			merged[key] = value
		}
		state.alert.Details = merged
	}
	state.wantKind = ""
	p.persist(state)

	// 安静结束（并不是真的恢复），或这次持续从没通知过人：不发恢复通知
	if incoming.Quiet || !state.announced() {
		p.finalize(state)
		return
	}
	state.wantKind = alert.KindResolved
}

// finalize 一条已结束的告警不再需要任何通知：撤掉它在分组里待发的内容，从内存里移除（库里的行保留）
func (p *AlertPipeline) finalize(state *alertState) {
	p.dequeue(state)
	delete(p.alerts, state.alert.Fingerprint)
}

// atlhyper_master_v2/notifier/pipeline/flush.go
// 分组发出：到期的分组按接收人拆成批次，每条告警写一条送达记录
package pipeline

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/routing"
)

// flushDueGroups 发出所有到期的分组
func (p *AlertPipeline) flushDueGroups(current *settings, now time.Time) {
	for _, key := range p.sortedGroupKeys() {
		group := p.groups[key]
		if len(group.pending) == 0 || now.Before(group.flushAt()) {
			continue
		}
		p.flushGroup(key, group, current, now)
	}
}

// flushGroup 把一个分组里待发的内容发出去：一个接收人一个批次。
// 没有任何可用接收人时，每条告警记一条 no-receiver —— 不算通知过，之后由重复提醒补发
func (p *AlertPipeline) flushGroup(key groupKey, group *groupState, current *settings, now time.Time) {
	var items []batchItem
	for _, fingerprint := range group.order {
		kind, waiting := group.pending[fingerprint]
		if waiting && p.alerts[fingerprint] != nil {
			items = append(items, batchItem{fingerprint: fingerprint, kind: kind})
		}
	}
	group.pending, group.order, group.lastFlushAt = make(map[string]alert.Kind), nil, now
	if len(items) == 0 {
		return
	}

	receivers := routing.ReceiversForRule(current.policy, current.rules, current.receivers, key.ruleID)
	if len(receivers) == 0 {
		p.recordDeliveries(newBatchID(), nil, items, database.NotifyDeliveryNoReceiver, now)
		p.warnNoReceiver(key.clusterID, len(items))
	}
	for _, receiver := range receivers {
		batchID := newBatchID()
		p.recordDeliveries(batchID, receiver, items, database.NotifyDeliveryPending, now)
		p.batchSeq++
		p.batches[batchID] = &batchState{id: batchID, seq: p.batchSeq, receiverID: receiver.ID, items: items, nextAttemptAt: now}
	}

	delivered := len(receivers) > 0
	for _, item := range items {
		state := p.alerts[item.fingerprint]
		state.queued = nil
		state.lastAttemptAt, state.lastNotifiedAt = now, now
		if delivered {
			state.notifyCount++
			if item.kind != alert.KindRepeat {
				state.lastNotifiedKind, state.lastNotifiedSeverity = item.kind, state.alert.Severity
			}
		}
		p.persist(state)
		if state.alert.Status == alert.StatusResolved {
			delete(p.alerts, item.fingerprint) // 恢复通知已交给批次，这条告警没有别的事了
		}
	}
}

// recordDeliveries 为一个批次写送达记录：每条告警一条。receiver 为 nil 表示没有接收人
func (p *AlertPipeline) recordDeliveries(batchID string, receiver *database.NotifyReceiver, items []batchItem, status string, now time.Time) {
	ctx, cancel := p.storeContext()
	defer cancel()
	for _, item := range items {
		delivery := &database.NotifyDelivery{
			BatchID: batchID, Fingerprint: item.fingerprint, AlertTitle: p.alerts[item.fingerprint].alert.Title,
			Kind: string(item.kind), Status: status, CreatedAt: now, UpdatedAt: now,
		}
		if receiver != nil {
			delivery.ReceiverID, delivery.ReceiverName, delivery.ReceiverType = receiver.ID, receiver.Name, receiver.Type
		}
		if err := p.repos.Deliveries.Create(ctx, delivery); err != nil {
			log.Error("送达记录落库失败", "fingerprint", item.fingerprint, "err", err)
		}
	}
}

// warnNoReceiver 记录「没有任何可用的接收人」。
//
// 没配接收人是一个持续的状态，而告警不断产生：同一个集群在一个时间窗内只记一次，
// droppedSinceLastLog 是上一条日志之后同集群未发送、也未记录的次数。每条告警本身都有 no-receiver 的送达记录可查
func (p *AlertPipeline) warnNoReceiver(clusterID string, alerts int) {
	allowed, dropped := p.noReceiverWarnings.allow(clusterID)
	if !allowed {
		return
	}
	log.Warn("没有任何可用的接收人，告警未发送",
		"cluster", clusterID, "alerts", alerts,
		"droppedSinceLastLog", dropped, "logInterval", noReceiverWarnWindow.String())
}

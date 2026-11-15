// atlhyper_master_v2/notifier/pipeline/restore.go
// 重启恢复与定期清理
package pipeline

import (
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// restoreFromStore Master 重启后从库里恢复：
//
//   - 持续中的告警：通知过的不重复发「触发」，重复提醒从上次通知的时刻接着算；
//     从没进入过发送的（重启前还在分组里等待）重新走一次路由
//   - 还没有定论的批次（待发 / 等重试）：接着发，已尝试的次数接着算
func (p *AlertPipeline) restoreFromStore() {
	now := p.now()
	ctx, cancel := p.storeContext()
	defer cancel()

	rows, err := p.repos.Alerts.ListFiring(ctx)
	if err != nil {
		log.Error("恢复持续中的告警失败", "err", err)
	}
	for _, row := range rows {
		state := stateFromRow(row)
		if state.lastNotifiedAt.IsZero() {
			state.lastAttemptAt, state.wantKind = now, state.alert.Kind
		} else {
			state.lastAttemptAt = state.lastNotifiedAt
		}
		p.alerts[row.Fingerprint] = state
	}

	deliveries, err := p.repos.Deliveries.ListUnfinished(ctx)
	if err != nil {
		log.Error("恢复未完成的批次失败", "err", err)
	}
	for _, delivery := range deliveries {
		batch := p.batches[delivery.BatchID]
		if batch == nil {
			p.batchSeq++
			batch = &batchState{id: delivery.BatchID, seq: p.batchSeq, receiverID: delivery.ReceiverID,
				attempts: delivery.Attempts, nextAttemptAt: now}
			if delivery.NextAttemptAt != nil {
				batch.nextAttemptAt = *delivery.NextAttemptAt
			}
			p.batches[delivery.BatchID] = batch
		}
		batch.items = append(batch.items, batchItem{fingerprint: delivery.Fingerprint, kind: alert.Kind(delivery.Kind)})
	}

	if len(p.alerts) > 0 || len(p.batches) > 0 {
		log.Info("已从库里恢复通知状态", "firingAlerts", len(p.alerts), "unfinishedBatches", len(p.batches))
	}
}

// cleanupIfDue 每小时清理一次：超过 30 天的送达记录、结束超过 30 天的告警、过期超过 30 天的静默
func (p *AlertPipeline) cleanupIfDue(now time.Time) {
	if !p.lastCleanupAt.IsZero() && now.Sub(p.lastCleanupAt) < cleanupInterval {
		return
	}
	p.lastCleanupAt = now
	before := now.Add(-retentionPeriod)
	ctx, cancel := p.storeContext()
	defer cancel()

	deliveries, err := p.repos.Deliveries.DeleteBefore(ctx, before)
	if err != nil {
		log.Error("清理送达记录失败", "err", err)
	}
	alerts, err := p.repos.Alerts.DeleteResolvedBefore(ctx, before)
	if err != nil {
		log.Error("清理已结束的告警失败", "err", err)
	}
	silences, err := p.repos.Silences.DeleteEndedBefore(ctx, before)
	if err != nil {
		log.Error("清理过期的静默失败", "err", err)
	}
	if deliveries+alerts+silences > 0 {
		log.Info("已清理过期的通知数据", "deliveries", deliveries, "alerts", alerts, "silences", silences)
	}
}

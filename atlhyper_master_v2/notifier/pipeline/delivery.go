// atlhyper_master_v2/notifier/pipeline/delivery.go
// 发送与重试：到点的批次渲染后并行发出，结果写回送达记录
package pipeline

import (
	"context"
	"sort"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/channel"
	"AtlHyper/atlhyper_master_v2/notifier/template"
)

// sendJob 一次待执行的发送
type sendJob struct {
	batch    *batchState
	receiver *database.NotifyReceiver
	message  *channel.Message
}

// sendDueBatches 发送所有到点的批次（首次发送与重试走同一条路）。
//
// 每次都按告警当前的内容重新渲染：重试发出去的是最新状态，而不是失败那一刻的快照
func (p *AlertPipeline) sendDueBatches(current *settings, now time.Time) {
	var due []*batchState
	for _, batch := range p.batches {
		if !batch.nextAttemptAt.After(now) {
			due = append(due, batch)
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i].seq < due[j].seq })

	var jobs []sendJob
	for _, batch := range due {
		receiver := current.receiver(batch.receiverID)
		switch {
		case receiver == nil:
			p.closeBatch(batch, database.NotifyDeliveryAbandoned, "接收人已删除", now)
			continue
		case !receiver.Enabled:
			p.closeBatch(batch, database.NotifyDeliveryAbandoned, "接收人已停用", now)
			continue
		}
		items := p.renderItems(batch)
		if len(items) == 0 {
			p.closeBatch(batch, database.NotifyDeliveryAbandoned, "告警已不存在", now)
			continue
		}
		message, err := p.renderer.Render(receiver.Type, items, now)
		if err != nil {
			p.closeBatch(batch, database.NotifyDeliveryAbandoned, "消息渲染失败: "+err.Error(), now)
			continue
		}
		jobs = append(jobs, sendJob{batch: batch, receiver: receiver, message: message})
	}

	for index, err := range p.runSendJobs(jobs) {
		p.recordSendResult(jobs[index], err, now)
	}
}

// renderItems 批次里各告警的当前内容。持续中的在内存里；已结束的从库里取
func (p *AlertPipeline) renderItems(batch *batchState) []template.Item {
	ctx, cancel := p.storeContext()
	defer cancel()

	items := make([]template.Item, 0, len(batch.items))
	for _, item := range batch.items {
		if state := p.alerts[item.fingerprint]; state != nil {
			items = append(items, template.Item{Alert: state.alert, Kind: item.kind})
			continue
		}
		row, err := p.repos.Alerts.Get(ctx, item.fingerprint)
		if err != nil {
			log.Error("读取告警失败，本条不计入消息", "fingerprint", item.fingerprint, "err", err)
			continue
		}
		if row != nil {
			items = append(items, template.Item{Alert: alertFromRow(row), Kind: item.kind})
		}
	}
	return items
}

// runSendJobs 并行执行发送（并发数有上限），返回与 jobs 一一对应的结果。
// 并行是为了让一个慢的接收人不拖住别的；等全部结束再返回，状态仍只由本协程修改
func (p *AlertPipeline) runSendJobs(jobs []sendJob) []error {
	results := make([]error, len(jobs))
	slots := make(chan struct{}, p.sendConcurrency)
	var wg sync.WaitGroup
	for index, job := range jobs {
		wg.Add(1)
		slots <- struct{}{}
		go func(index int, job sendJob) {
			defer wg.Done()
			defer func() { <-slots }()
			ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
			defer cancel()
			results[index] = p.sender.Send(ctx, job.receiver, job.message)
		}(index, job)
	}
	wg.Wait()
	return results
}

// recordSendResult 记下一次发送的结果：成功即结束；失败则安排重试，重试用尽后放弃
func (p *AlertPipeline) recordSendResult(job sendJob, sendErr error, now time.Time) {
	batch := job.batch
	batch.attempts++
	if sendErr == nil {
		p.closeBatch(batch, database.NotifyDeliverySuccess, "", now)
		return
	}

	// 发送器按约定不会把凭据放进错误里；这里再抹一遍，送达记录与日志是最后一道
	reason := truncate(channel.ScrubSecrets(sendErr.Error(), channel.ReceiverSecrets(job.receiver)...), maxLastErrorLength)
	if batch.attempts > len(retryBackoff) {
		log.Warn("通知发送失败，重试已用尽，放弃", "receiver", job.receiver.Name, "attempts", batch.attempts, "err", reason)
		p.closeBatch(batch, database.NotifyDeliveryAbandoned, reason, now)
		return
	}

	batch.nextAttemptAt = now.Add(retryBackoff[batch.attempts-1])
	log.Warn("通知发送失败，稍后重试", "receiver", job.receiver.Name, "attempts", batch.attempts,
		"retryIn", retryBackoff[batch.attempts-1].String(), "err", reason)
	nextAttemptAt := batch.nextAttemptAt
	p.updateBatch(batch, database.NotifyDeliveryResult{
		Status: database.NotifyDeliveryFailed, Attempts: batch.attempts, LastError: reason,
		NextAttemptAt: &nextAttemptAt, UpdatedAt: now,
	})
}

// closeBatch 批次有了定论（成功 / 放弃）：写回送达记录并从内存里移除
func (p *AlertPipeline) closeBatch(batch *batchState, status, reason string, now time.Time) {
	p.updateBatch(batch, database.NotifyDeliveryResult{Status: status, Attempts: batch.attempts, LastError: reason, UpdatedAt: now})
	delete(p.batches, batch.id)
}

func (p *AlertPipeline) updateBatch(batch *batchState, result database.NotifyDeliveryResult) {
	ctx, cancel := p.storeContext()
	defer cancel()
	if err := p.repos.Deliveries.UpdateBatch(ctx, batch.id, result); err != nil {
		log.Error("送达记录更新失败", "batch", batch.id, "err", err)
	}
}

// truncate 按字符数截断
func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "…"
}

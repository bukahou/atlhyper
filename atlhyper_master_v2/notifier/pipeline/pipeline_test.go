package pipeline

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

const (
	groupWait     = 30 * time.Second
	groupInterval = 5 * time.Minute
	repeatEvery   = 4 * time.Hour
)

// 触发后要等满分组等待才发；发出后告警上记下通知时间与次数，送达记录为成功
func TestFiringIsSentAfterGroupWait(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))

	h.advance(groupWait - time.Second)
	h.wantSent(0)

	h.advance(time.Second)
	messages := h.wantSent(1)
	if messages[0].receiverID != receiverID || !strings.Contains(messages[0].subject, "[prod] 高 · shop/deployment/api") ||
		!strings.Contains(messages[0].body, "触发") {
		t.Errorf("消息不对: %+v", messages[0])
	}
	h.wantDeliveries("incident:api", "firing/success")

	stored := h.storedAlert("incident:api")
	if stored.Status != "firing" || stored.NotifyCount != 1 || stored.LastNotifiedKind != "firing" ||
		stored.LastNotifiedAt == nil || !stored.LastNotifiedAt.Equal(h.clock.now()) || !stored.StartsAt.Equal(testStart) {
		t.Errorf("告警的通知状态不对: %+v", stored)
	}
	delivery := h.deliveries("incident:api")[0]
	if delivery.ReceiverName != "值班频道" || delivery.ReceiverType != "slack" || delivery.Attempts != 1 ||
		delivery.AlertTitle != "shop/deployment/api 风险异常" || delivery.BatchID == "" {
		t.Errorf("送达记录不对: %+v", delivery)
	}
}

// 同一集群在等待窗口内的告警合并成一条；别的集群各发各的
func TestGroupWaitMergesSameCluster(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(10 * time.Second)
	h.submit(incidentAlert("db", "prod", alert.SeverityCritical))
	h.submit(incidentAlert("web", "test", alert.SeverityLow))

	h.advance(20 * time.Second) // 第一条满 30 秒：prod 组发出，带上晚到的那条
	messages := h.wantSent(1)
	if !strings.Contains(messages[0].subject, "[prod] 2 条告警（最高严重）") ||
		!strings.Contains(messages[0].body, "shop/deployment/api") || !strings.Contains(messages[0].body, "shop/deployment/db") {
		t.Errorf("prod 的两条应合并成一条: %+v", messages[0])
	}
	apiBatch, dbBatch := h.deliveries("incident:api")[0].BatchID, h.deliveries("incident:db")[0].BatchID
	if apiBatch != dbBatch {
		t.Errorf("同一条消息里的告警应共用批次号：%s / %s", apiBatch, dbBatch)
	}

	h.advance(10 * time.Second) // test 集群的那条自己满 30 秒
	messages = h.wantSent(2)
	if !strings.Contains(messages[1].subject, "[test] 低 · shop/deployment/web") {
		t.Errorf("test 集群应单独发一条: %+v", messages[1])
	}
}

// 一组刚发过，同组的新内容要等满分组间隔
func TestGroupIntervalDelaysNextMessage(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)

	h.advance(time.Minute)
	h.submit(incidentAlert("db", "prod", alert.SeverityHigh))
	h.advance(groupWait) // 自己的等待满了，但距上次发送才 1 分半
	h.wantSent(1)

	h.advance(groupInterval - time.Minute - groupWait - time.Second)
	h.wantSent(1)
	h.advance(time.Second) // 距上次发送满 5 分钟
	h.wantSent(2)
	h.wantDeliveries("incident:db", "firing/success")
}

// 持续未结束的告警每隔重复间隔再提醒一次
func TestRepeatReminder(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	h.advance(repeatEvery - time.Minute)
	h.wantSent(1)
	h.advance(time.Minute) // 到点：进入分组
	h.advance(groupWait)   // 同样走分组等待
	messages := h.wantSent(2)
	if !strings.Contains(messages[1].body, "重复提醒") {
		t.Errorf("第二条应是重复提醒: %+v", messages[1])
	}

	h.advance(repeatEvery)
	h.advance(groupWait)
	h.wantSent(3)
	h.wantDeliveries("incident:api", "firing/success", "repeat/success", "repeat/success")
	if stored := h.storedAlert("incident:api"); stored.NotifyCount != 3 || stored.Kind != "firing" || stored.LastNotifiedKind != "firing" {
		t.Errorf("重复提醒只累计次数，不改变告警的种类: %+v", stored)
	}
}

// 去重：同一条告警的同一种变化不重复发
func TestDuplicateFiringIsNotResent(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh)) // 还没发出去就又来一次
	h.advance(groupWait)
	h.wantSent(1)

	for i := 0; i < 3; i++ { // 发出去之后来源又重复提交
		h.advance(time.Minute)
		h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	}
	h.advance(groupInterval)
	h.wantSent(1)
	h.wantDeliveries("incident:api", "firing/success")
}

// 严重度升高：更新告警并再发一次（升级）
func TestEscalationIsSent(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	h.advance(groupInterval)
	h.submit(withKind(incidentAlert("api", "prod", alert.SeverityCritical), alert.KindEscalated))
	h.advance(groupWait)
	messages := h.wantSent(2)
	if !strings.Contains(messages[1].subject, "严重") || !strings.Contains(messages[1].body, "升级") {
		t.Errorf("应发出升级通知: %+v", messages[1])
	}
	stored := h.storedAlert("incident:api")
	if stored.Severity != "critical" || stored.Kind != "escalated" || !stored.StartsAt.Equal(testStart) {
		t.Errorf("升级应更新严重度与种类、保留开始时间: %+v", stored)
	}
}

// 触发通知还没发出去就升级了：收件人看到的第一条仍是「触发」，内容是最新的
func TestEscalationBeforeFirstMessageKeepsFiring(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(10 * time.Second)
	h.submit(withKind(incidentAlert("api", "prod", alert.SeverityCritical), alert.KindEscalated))
	h.advance(20 * time.Second)

	messages := h.wantSent(1)
	if !strings.Contains(messages[0].body, "触发") || !strings.Contains(messages[0].subject, "严重") {
		t.Errorf("应是一条「触发」，严重度为最新值: %+v", messages[0])
	}
	h.wantDeliveries("incident:api", "firing/success")
}

// 恢复：通知过的告警结束时发恢复通知
func TestResolvedIsSent(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.advance(time.Hour)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(groupWait)

	messages := h.wantSent(2)
	if !strings.Contains(messages[1].body, "恢复") {
		t.Errorf("应发出恢复通知: %+v", messages[1])
	}
	h.wantDeliveries("incident:api", "firing/success", "resolved/success")
	stored := h.storedAlert("incident:api")
	if stored.Status != "resolved" || stored.Kind != "resolved" || stored.EndsAt == nil || !stored.EndsAt.Equal(testStart.Add(groupWait+time.Hour)) {
		t.Errorf("告警应已结束: %+v", stored)
	}

	// 结束后不再有重复提醒
	h.advance(2 * repeatEvery)
	h.wantSent(2)
}

// 还没通知过就结束了：触发与恢复都不发
func TestResolvedBeforeFirstMessageSendsNothing(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(10 * time.Second)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(time.Hour)

	h.wantSent(0)
	h.wantDeliveries("incident:api")
	if stored := h.storedAlert("incident:api"); stored.Status != "resolved" {
		t.Errorf("告警应已结束: %+v", stored)
	}
}

// 策略关掉恢复通知
func TestNotifyResolvedDisabled(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.setPolicy(func(p *database.NotifyPolicy) { p.NotifyResolved = false })
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(time.Hour)

	h.wantSent(1)
	if stored := h.storedAlert("incident:api"); stored.Status != "resolved" {
		t.Errorf("告警应已结束: %+v", stored)
	}
}

// 「安静结束」（失联关闭等）：告警结束，但不发恢复通知
func TestQuietResolveSendsNothing(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	quiet := resolved(incidentAlert("api", "prod", alert.SeverityHigh))
	quiet.Quiet = true
	h.submit(quiet)
	h.advance(time.Hour)

	h.wantSent(1)
	h.wantDeliveries("incident:api", "firing/success")
	if stored := h.storedAlert("incident:api"); stored.Status != "resolved" || stored.EndsAt == nil {
		t.Errorf("告警应已结束: %+v", stored)
	}
}

// 没有在持续的告警，收到「结束」什么也不做
func TestResolveWithoutFiringAlertIsIgnored(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(time.Hour)
	h.wantSent(0)
	if row, _ := h.db.NotifyAlert.Get(t.Context(), "incident:api"); row != nil {
		t.Errorf("不应凭空落一条已结束的告警: %+v", row)
	}
}

// 恢复后复发：重新触发，带复发通知；累计次数接着算，开始时间重置
func TestRecurrenceStartsNewEpisode(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(groupInterval)
	h.wantSent(2)

	h.advance(time.Hour)
	recurredAt := h.clock.now()
	h.submit(withKind(incidentAlert("api", "prod", alert.SeverityHigh), alert.KindRecurred))
	h.advance(groupWait)
	messages := h.wantSent(3)
	if !strings.Contains(messages[2].body, "复发") {
		t.Errorf("应发出复发通知: %+v", messages[2])
	}
	stored := h.storedAlert("incident:api")
	if stored.Status != "firing" || stored.Kind != "recurred" || stored.NotifyCount != 3 || stored.EndsAt != nil || !stored.StartsAt.Equal(recurredAt) {
		t.Errorf("复发后的告警不对: %+v", stored)
	}
}

// 警告级的反复告警在持续，事件随后真正升级：这是新的变化，按「触发」再发一次
func TestFlappingAlertPromotedToFiring(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(withKind(incidentAlert("api", "prod", alert.SeverityMedium), alert.KindFlapping))
	h.advance(groupWait)
	messages := h.wantSent(1)
	if !strings.Contains(messages[0].body, "反复") {
		t.Errorf("第一条应是反复通知: %+v", messages[0])
	}

	// 再反复一次：同一种变化，不重复发
	h.advance(groupInterval)
	h.submit(withKind(incidentAlert("api", "prod", alert.SeverityMedium), alert.KindFlapping))
	h.advance(groupInterval)
	h.wantSent(1)

	h.submit(incidentAlert("api", "prod", alert.SeverityMedium)) // 进入事件中
	h.advance(groupWait)
	messages = h.wantSent(2)
	if !strings.Contains(messages[1].body, "触发") {
		t.Errorf("升级为事件后应发「触发」: %+v", messages[1])
	}
	h.wantDeliveries("incident:api", "flapping/success", "firing/success")
	if stored := h.storedAlert("incident:api"); stored.Kind != "firing" {
		t.Errorf("告警的种类应变为 firing: %+v", stored)
	}
}

// 恢复通知还没发出去告警又触发了：收件人眼里它一直在持续，两条都不发；之后真正结束时照常发恢复
func TestRefireBeforeResolvedWasSentIsSilent(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)

	h.advance(time.Minute)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(10 * time.Second) // 恢复通知还在等分组间隔
	h.submit(withKind(incidentAlert("api", "prod", alert.SeverityHigh), alert.KindRecurred))
	h.advance(groupInterval)
	h.wantSent(1)
	stored := h.storedAlert("incident:api")
	if stored.Status != "firing" || stored.EndsAt != nil || !stored.StartsAt.Equal(testStart) {
		t.Errorf("告警应接着算作同一次持续: %+v", stored)
	}

	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(groupWait)
	h.wantSent(2)
	h.wantDeliveries("incident:api", "firing/success", "resolved/success")
}

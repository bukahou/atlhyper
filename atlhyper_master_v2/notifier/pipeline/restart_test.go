package pipeline

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// Master 重启：持续中的告警从库里恢复，来源再提交一次也不重复发「触发」；重复提醒的计时接着算
func TestRestartDoesNotResendFiring(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.submit(heartbeatAlert("test"))
	h.advance(groupWait)
	h.wantSent(2)

	h.advance(time.Hour)
	h.restart()
	h.submit(heartbeatAlert("test")) // 心跳来源重启后看到 Agent 仍离线，又提交一次
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupInterval)
	h.wantSent(2)

	// 距第一次通知满 4 小时：重复提醒（而不是从重启时刻重新计时）
	h.advance(repeatEvery - time.Hour - groupInterval - time.Minute)
	h.wantSent(2)
	h.advance(time.Minute)
	h.advance(groupWait)
	messages := h.wantSent(4)
	for _, message := range messages[2:] {
		if !strings.Contains(message.body, "重复提醒") {
			t.Errorf("重启后到点的应是重复提醒: %+v", message)
		}
	}
}

// 重启时还在分组等待里、没来得及发的告警：恢复后发出，只发一次
func TestRestartSendsAlertThatWasStillWaiting(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(10 * time.Second)
	h.restart()

	h.advance(time.Second)
	h.advance(groupWait)
	messages := h.wantSent(1)
	if !strings.Contains(messages[0].body, "触发") {
		t.Errorf("应补发触发通知: %+v", messages[0])
	}
	h.advance(groupInterval)
	h.wantSent(1)
}

// 重启时发送失败、正在等重试的批次：恢复后接着重试，次数接着算，不另起一条
func TestRestartResumesFailedBatch(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	h.sender.fail(receiverID, errSendFailed)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantDeliveries("incident:api", "firing/failed")

	h.restart()
	h.sender.fail(receiverID, nil)
	h.advance(29 * time.Second)
	h.wantSent(0)
	h.advance(time.Second)
	h.wantSent(1)

	rows := h.deliveries("incident:api")
	if len(rows) != 1 || rows[0].Status != database.NotifyDeliverySuccess || rows[0].Attempts != 2 {
		t.Errorf("应在原来的送达记录上重试成功: %+v", rows)
	}
}

// 重启后抑制关系仍在：离线告警持续时，同集群的告警继续被压住
func TestRestartKeepsSuppression(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(heartbeatAlert("prod"))
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)

	h.restart()
	h.advance(groupInterval)
	h.wantSent(1)
	h.wantDeliveries("incident:api")
}

// 入队是有界的：队列满时丢弃并计数，不阻塞调用方
func TestSubmitDropsWhenQueueIsFull(t *testing.T) {
	h := newHarness(t)
	h.p = NewAlertPipeline(Config{Repos: h.p.repos, Sender: h.sender, Renderer: h.p.renderer, Now: h.clock.now, QueueSize: 2})

	accepted := 0
	for i := 0; i < 5; i++ { // 管道没有启动，没人消费
		if h.p.Submit(incidentAlert("api", "prod", alert.SeverityHigh)) {
			accepted++
		}
	}
	if accepted != 2 || h.p.Dropped() != 3 {
		t.Errorf("队列容量 2：应接受 2 条、丢弃 3 条，实际接受 %d、丢弃 %d", accepted, h.p.Dropped())
	}
	if h.p.SubmitDeferred(func(*database.NotifyPolicy) (alert.Alert, bool) { return alert.Alert{}, false }) {
		t.Error("队列满时延迟构造的提交也应被丢弃")
	}
	if h.p.Dropped() != 4 {
		t.Errorf("丢弃计数应为 4，实际 %d", h.p.Dropped())
	}
}

// Start / Stop：启动后入队的告警被后台协程处理并发出；Stop 等协程退出，可重复调用
func TestStartProcessesQueueAndStopIsIdempotent(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.setPolicy(func(p *database.NotifyPolicy) { p.GroupWaitSeconds = 0 })
	h.sender.notify = make(chan struct{}, 4)
	h.p = NewAlertPipeline(Config{Repos: h.p.repos, Sender: h.sender, Renderer: h.p.renderer, Now: h.clock.now,
		TickInterval: 5 * time.Millisecond})

	if err := h.p.Start(); err != nil {
		t.Fatal(err)
	}
	if !h.p.Submit(incidentAlert("api", "prod", alert.SeverityHigh)) {
		t.Fatal("入队应成功")
	}
	select {
	case <-h.sender.notify:
	case <-time.After(10 * time.Second):
		t.Fatal("后台协程没有把告警发出去")
	}
	h.p.Stop()
	h.p.Stop()

	h.wantSent(1)
	if h.p.Submit(incidentAlert("db", "prod", alert.SeverityHigh)) {
		t.Error("停止后不再接受新的告警")
	}
}

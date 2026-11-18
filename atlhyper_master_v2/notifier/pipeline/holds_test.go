package pipeline

import (
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// 抑制：集群的 Agent 离线告警在持续时，该集群的其他告警不发；别的集群不受影响
func TestClusterOfflineSuppressesOtherAlerts(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(heartbeatAlert("prod"))
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.submit(incidentAlert("web", "test", alert.SeverityHigh))
	h.advance(groupWait)

	messages := h.wantSent(2)
	if !strings.Contains(messages[0].subject, "[prod] 严重 · Agent 离线") || !strings.Contains(messages[1].subject, "[test]") {
		t.Errorf("应只发出 prod 的离线告警与 test 的事件告警: %+v", messages)
	}
	h.wantDeliveries("incident:api")
	if stored := h.storedAlert("incident:api"); stored.SuppressedBy != "cluster-offline" || stored.Status != "firing" {
		t.Errorf("被抑制的告警照常落库并标记原因: %+v", stored)
	}
	if stored := h.storedAlert("heartbeat:prod"); stored.SuppressedBy != "" {
		t.Errorf("离线告警自己不被抑制: %+v", stored)
	}

	// 抑制期间也没有重复提醒
	h.advance(repeatEvery)
	h.advance(groupWait)
	h.wantDeliveries("incident:api")
}

// 已经通知过的告警，在集群离线后也被压住；集群恢复上线后仍在持续的，按新告警发出
func TestSuppressionAppliesToExistingAlertsAndReleases(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)

	h.advance(time.Minute)
	h.submit(heartbeatAlert("prod"))
	h.advance(groupInterval)
	h.wantSent(2) // 离线告警
	if stored := h.storedAlert("incident:api"); stored.SuppressedBy != "cluster-offline" {
		t.Errorf("集群离线后，已有的告警也应标记为被抑制: %+v", stored)
	}

	h.advance(time.Hour)
	h.submit(resolved(heartbeatAlert("prod")))
	h.advance(time.Second) // 这一轮解除抑制
	if stored := h.storedAlert("incident:api"); stored.SuppressedBy != "" {
		t.Errorf("集群恢复后应解除抑制: %+v", stored)
	}
	h.advance(groupWait)
	messages := h.wantSent(3)
	last := messages[2]
	if !strings.Contains(last.subject, "2 条告警") || !strings.Contains(last.body, "恢复") || !strings.Contains(last.body, "触发") {
		t.Errorf("恢复上线的通知与重新发出的告警应合并在一条里: %+v", last)
	}
	h.wantDeliveries("incident:api", "firing/success", "firing/success")
}

// 静默：命中的告警照常落库，标记静默 id，不发；静默到期时仍在持续的按新告警发出
func TestSilenceHoldsAndReleasesOnExpiry(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	silenceID := h.addSilence(&database.NotifySilence{ClusterID: "prod", Namespace: "shop",
		StartsAt: testStart.Add(-time.Minute), EndsAt: testStart.Add(time.Hour)})

	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.submit(incidentAlert("web", "test", alert.SeverityHigh)) // 别的集群不在静默里
	h.advance(groupWait)
	messages := h.wantSent(1)
	if !strings.Contains(messages[0].subject, "[test]") {
		t.Errorf("只有不在静默里的告警会发出: %+v", messages)
	}
	if stored := h.storedAlert("incident:api"); stored.SilencedBy != silenceID || stored.Status != "firing" {
		t.Errorf("被静默的告警照常落库并记下静默 id: %+v", stored)
	}

	h.advance(time.Hour) // 静默到期
	if stored := h.storedAlert("incident:api"); stored.SilencedBy != 0 {
		t.Errorf("静默到期后应清掉标记: %+v", stored)
	}
	h.advance(groupWait)
	messages = h.wantSent(2)
	if !strings.Contains(messages[1].subject, "[prod]") || !strings.Contains(messages[1].body, "触发") {
		t.Errorf("静默到期后仍在持续的应按新告警发出: %+v", messages[1])
	}
	h.wantDeliveries("incident:api", "firing/success")
}

// 静默期间结束的告警：不补发触发，也不发恢复
func TestAlertResolvedDuringSilenceSendsNothing(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.addSilence(&database.NotifySilence{ClusterID: "prod", StartsAt: testStart, EndsAt: testStart.Add(time.Hour)})

	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(10 * time.Minute)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(2 * time.Hour)
	h.advance(groupInterval)

	h.wantSent(0)
	h.wantDeliveries("incident:api")
}

// 通知过的告警，之后进入静默并在静默期间结束：不发恢复
func TestAnnouncedAlertResolvedDuringLaterSilence(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)

	h.addSilence(&database.NotifySilence{ClusterID: "prod", StartsAt: h.clock.now(), EndsAt: h.clock.now().Add(time.Hour)})
	h.advance(time.Minute)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(2 * time.Hour)
	h.advance(groupInterval)

	h.wantSent(1)
	h.wantDeliveries("incident:api", "firing/success")
}

// 默认策略的严重度门槛：不到的不发、不留送达记录
func TestBelowMinSeverityIsNotSent(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.setPolicy(func(p *database.NotifyPolicy) { p.MinSeverity = "high" })
	h.submit(incidentAlert("api", "prod", alert.SeverityMedium))
	h.submit(incidentAlert("db", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	messages := h.wantSent(1)
	if !strings.Contains(messages[0].subject, "shop/deployment/db") {
		t.Errorf("只有达到门槛的会发出: %+v", messages[0])
	}
	h.wantDeliveries("incident:api")
	if stored := h.storedAlert("incident:api"); stored.Status != "firing" || stored.NotifyCount != 0 {
		t.Errorf("不到门槛的告警照常落库: %+v", stored)
	}
}

// 路由规则：命中的规则决定接收人与时间参数；没命中的走默认策略
func TestRuleDecidesReceiversAndTiming(t *testing.T) {
	h := newHarness(t)
	devID := h.addReceiver("开发频道")
	oncallID := h.addReceiver("值班频道")
	noWait := 0
	h.addRule(&database.NotifyRule{Name: "生产高严重度", Enabled: true, ClusterIDs: []string{"prod"},
		MinSeverity: "high", ReceiverIDs: []int64{oncallID}, GroupWaitSeconds: &noWait})

	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.submit(incidentAlert("web", "test", alert.SeverityHigh))
	h.advance(time.Second) // 规则把分组等待设成 0：下一轮就发
	messages := h.wantSent(1)
	if messages[0].receiverID != oncallID || !strings.Contains(messages[0].subject, "[prod]") {
		t.Errorf("生产的高严重度告警应只发给值班频道: %+v", messages[0])
	}

	h.advance(groupWait) // test 集群走默认策略：全部已启用的接收人，全局的等待时间
	messages = h.wantSent(3)
	got := map[int64]bool{messages[1].receiverID: true, messages[2].receiverID: true}
	if !got[devID] || !got[oncallID] || !strings.Contains(messages[1].subject, "[test]") {
		t.Errorf("默认策略应发给全部已启用的接收人: %+v", messages[1:])
	}
	if rows := h.deliveries("incident:web"); len(rows) != 2 || rows[0].BatchID == rows[1].BatchID {
		t.Errorf("一个接收人一个批次: %+v", rows)
	}
}

// 没有任何可用接收人：告警照常落库并记一条 no-receiver；配好接收人后由重复提醒补发
func TestNoReceiverIsRecordedAndSentLater(t *testing.T) {
	h := newHarness(t)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(0)
	h.wantDeliveries("incident:api", "firing/no-receiver")
	if stored := h.storedAlert("incident:api"); stored.NotifyCount != 0 || stored.LastNotifiedKind != "" {
		t.Errorf("没有接收人时不算通知过: %+v", stored)
	}

	h.addReceiver("值班频道")
	h.advance(repeatEvery)
	h.advance(groupWait)
	messages := h.wantSent(1)
	if !strings.Contains(messages[0].body, "触发") {
		t.Errorf("还没人收到过，补发时应仍是「触发」而不是「重复提醒」: %+v", messages[0])
	}
	h.wantDeliveries("incident:api", "firing/no-receiver", "firing/success")
}

// 停用的接收人不发
func TestDisabledReceiverIsSkipped(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	receiver, _ := h.db.NotifyReceiver.GetByID(t.Context(), receiverID)
	receiver.Enabled = false
	if err := h.db.NotifyReceiver.Update(t.Context(), receiver); err != nil {
		t.Fatal(err)
	}
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(0)
	h.wantDeliveries("incident:api", "firing/no-receiver")
}

// 关掉事件告警后，已有的事件告警不再提醒；心跳告警不受影响
func TestIncidentAlertsDisabledStopsNotifications(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.submit(heartbeatAlert("test"))
	h.advance(groupWait)
	h.wantSent(2)

	h.setPolicy(func(p *database.NotifyPolicy) { p.IncidentAlertsEnabled = false })
	h.advance(repeatEvery)
	h.advance(groupWait)
	messages := h.wantSent(3)
	if !strings.Contains(messages[2].subject, "Agent 离线") {
		t.Errorf("只有心跳告警继续提醒: %+v", messages[2])
	}
	h.wantDeliveries("incident:api", "firing/success")
}

// 延迟构造：入队的是「拿到当前策略后再产出告警」的函数
func TestSubmissionBuiltFromCurrentPolicy(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.setPolicy(func(p *database.NotifyPolicy) { p.WarningFlapThreshold = 7 })

	var seenThreshold int
	h.p.process(submission{build: func(policy *database.NotifyPolicy) (alert.Alert, bool) {
		seenThreshold = policy.WarningFlapThreshold
		return incidentAlert("api", "prod", alert.SeverityHigh), true
	}})
	h.p.process(submission{build: func(*database.NotifyPolicy) (alert.Alert, bool) {
		return incidentAlert("db", "prod", alert.SeverityHigh), false // 不产生告警
	}})
	h.advance(groupWait)

	if seenThreshold != 7 {
		t.Errorf("构造函数应拿到当前策略，实际阈值 %d", seenThreshold)
	}
	h.wantSent(1)
	if row, _ := h.db.NotifyAlert.Get(t.Context(), "incident:db"); row != nil {
		t.Errorf("构造函数说不产生告警时不应落库: %+v", row)
	}
}

// 没配接收人是持续的状态：同一个集群的「没有接收人」日志在时间窗内只记一次，但每条告警都有送达记录
func TestNoReceiverWarningIsRateLimitedPerCluster(t *testing.T) {
	h := newHarness(t)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait) // prod 第 1 次：记日志
	h.submit(incidentAlert("db", "prod", alert.SeverityHigh))
	h.advance(groupInterval) // prod 第 2 次，距上一条日志 5 分钟：压下
	h.submit(incidentAlert("cache", "test", alert.SeverityHigh))
	h.advance(groupWait) // 另一个集群：照常记

	prod, test := h.p.noReceiverWarnings.entries["prod"], h.p.noReceiverWarnings.entries["test"]
	if prod == nil || test == nil {
		t.Fatalf("两个集群都应记过日志: prod=%v test=%v", prod, test)
	}
	if prod.dropped != 1 || test.dropped != 0 {
		t.Errorf("prod 应压下 1 条、test 压下 0 条，实际 %d / %d", prod.dropped, test.dropped)
	}
	for _, id := range []string{"api", "db", "cache"} {
		h.wantDeliveries("incident:"+id, "firing/no-receiver")
	}
}

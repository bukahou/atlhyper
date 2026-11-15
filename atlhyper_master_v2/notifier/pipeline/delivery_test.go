package pipeline

import (
	"errors"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// 失败按 30 秒、2 分钟、10 分钟重试三次，仍失败记为「已放弃」；不影响下一条
func TestRetryThenAbandon(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	h.sender.fail(receiverID, errSendFailed)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))

	h.advance(groupWait) // 第 1 次
	steps := []struct {
		wait         time.Duration
		wantAttempts int
		wantStatus   string
	}{
		{0, 1, database.NotifyDeliveryFailed},
		{30 * time.Second, 2, database.NotifyDeliveryFailed},
		{2 * time.Minute, 3, database.NotifyDeliveryFailed},
		{10 * time.Minute, 4, database.NotifyDeliveryAbandoned},
	}
	for _, step := range steps {
		if step.wait > 0 {
			h.advance(step.wait - time.Second)
			if calls := h.sender.callCount(); calls != step.wantAttempts-1 {
				t.Fatalf("重试时间没到就发了：已调用 %d 次", calls)
			}
			h.advance(time.Second)
		}
		delivery := h.deliveries("incident:api")[0]
		if delivery.Attempts != step.wantAttempts || delivery.Status != step.wantStatus || delivery.LastError != "对端返回 500" {
			t.Fatalf("第 %d 次之后的送达记录不对: %+v", step.wantAttempts, delivery)
		}
	}
	if abandoned := h.deliveries("incident:api")[0]; abandoned.NextAttemptAt != nil {
		t.Errorf("放弃后不应再有下次重试时间: %+v", abandoned)
	}

	// 放弃之后不再重试；下一条告警照常发
	h.sender.fail(receiverID, nil)
	h.advance(time.Hour)
	if calls := h.sender.callCount(); calls != 4 {
		t.Errorf("放弃后不应再发，实际调用 %d 次", calls)
	}
	h.submit(incidentAlert("db", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantSent(1)
	h.wantDeliveries("incident:db", "firing/success")
}

// 重试成功；重试时按告警当前内容重新渲染
func TestRetrySucceedsWithCurrentContent(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	h.sender.fail(receiverID, errSendFailed)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.wantDeliveries("incident:api", "firing/failed")

	refreshed := incidentAlert("api", "prod", alert.SeverityHigh)
	refreshed.Summary = "成员增加到 5 个"
	h.submit(refreshed)
	h.sender.fail(receiverID, nil)
	h.advance(30 * time.Second)

	messages := h.wantSent(1)
	if !strings.Contains(messages[0].subject, "成员增加到 5 个") {
		t.Errorf("重试应使用告警的最新内容: %+v", messages[0])
	}
	delivery := h.deliveries("incident:api")[0]
	if delivery.Status != database.NotifyDeliverySuccess || delivery.Attempts != 2 || delivery.NextAttemptAt != nil || delivery.LastError != "" {
		t.Errorf("重试成功后的送达记录不对: %+v", delivery)
	}
}

// 发送器的报错里带着 Webhook 地址（Go 的 http.Client 就会这样）：写进送达记录前必须去掉
func TestLastErrorNeverContainsWebhookURL(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	h.sender.fail(receiverID, errors.New(`Post "`+testWebhookURL+`": dial tcp: lookup hooks.example.com: no such host`))
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	delivery := h.deliveries("incident:api")[0]
	if delivery.Status != database.NotifyDeliveryFailed || delivery.LastError == "" {
		t.Fatalf("应记下失败原因: %+v", delivery)
	}
	for _, leaked := range []string{testWebhookURL, "hooks.example.com", "/services/T000", "placeholder-not-real"} {
		if strings.Contains(delivery.LastError, leaked) {
			t.Errorf("送达记录的错误信息泄漏了 %q：%s", leaked, delivery.LastError)
		}
	}
	if !strings.Contains(delivery.LastError, "no such host") {
		t.Errorf("失败的大致原因应保留：%s", delivery.LastError)
	}
}

// 一个接收人失败不影响另一个
func TestOneReceiverFailingDoesNotBlockAnother(t *testing.T) {
	h := newHarness(t)
	brokenID := h.addReceiver("坏掉的频道")
	h.addReceiver("值班频道")
	h.sender.fail(brokenID, errSendFailed)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	h.wantSent(1)
	statuses := map[string]string{}
	for _, row := range h.deliveries("incident:api") {
		statuses[row.ReceiverName] = row.Status
	}
	if statuses["坏掉的频道"] != database.NotifyDeliveryFailed || statuses["值班频道"] != database.NotifyDeliverySuccess {
		t.Errorf("两个接收人的结果应各记各的: %v", statuses)
	}
}

// 等待重试期间接收人被删除：放弃，不再发
func TestReceiverDeletedBeforeRetry(t *testing.T) {
	h := newHarness(t)
	receiverID := h.addReceiver("值班频道")
	h.sender.fail(receiverID, errSendFailed)
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)

	if err := h.db.NotifyReceiver.Delete(t.Context(), receiverID); err != nil {
		t.Fatal(err)
	}
	h.advance(30 * time.Second)
	delivery := h.deliveries("incident:api")[0]
	if delivery.Status != database.NotifyDeliveryAbandoned || !strings.Contains(delivery.LastError, "接收人") || delivery.ReceiverName != "值班频道" {
		t.Errorf("接收人删除后应放弃，记录里仍保留它的名称: %+v", delivery)
	}
	if calls := h.sender.callCount(); calls != 1 {
		t.Errorf("不应再调用发送，实际 %d 次", calls)
	}
}

// 送达记录与已结束的告警保留 30 天
func TestCleanupAfterThirtyDays(t *testing.T) {
	h := newHarness(t)
	h.addReceiver("值班频道")
	h.addSilence(&database.NotifySilence{ClusterID: "other", StartsAt: testStart, EndsAt: testStart.Add(time.Hour)})
	h.submit(incidentAlert("api", "prod", alert.SeverityHigh))
	h.advance(groupWait)
	h.submit(resolved(incidentAlert("api", "prod", alert.SeverityHigh)))
	h.advance(groupInterval)
	h.wantDeliveries("incident:api", "firing/success", "resolved/success")

	h.advance(29 * 24 * time.Hour)
	if rows := h.deliveries("incident:api"); len(rows) != 2 {
		t.Errorf("30 天内的记录应保留，实际 %d 条", len(rows))
	}
	h.advance(2 * 24 * time.Hour)
	if rows := h.deliveries("incident:api"); len(rows) != 0 {
		t.Errorf("超过 30 天的送达记录应清理，实际 %d 条", len(rows))
	}
	if row, _ := h.db.NotifyAlert.Get(t.Context(), "incident:api"); row != nil {
		t.Errorf("结束超过 30 天的告警应清理: %+v", row)
	}
	if silences, _ := h.db.NotifySilence.List(t.Context()); len(silences) != 0 {
		t.Errorf("过期超过 30 天的静默应清理: %+v", silences)
	}
}

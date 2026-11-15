package template

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

var renderNow = time.Date(2026, 11, 15, 12, 5, 0, 0, time.UTC)

func incidentItem(kind alert.Kind) Item {
	return Item{Kind: kind, Alert: alert.Alert{
		Fingerprint: "incident:inc-123", Source: alert.SourceIncident, ClusterID: "prod",
		Severity: alert.SeverityHigh, Status: alert.StatusFiring, Kind: alert.KindFiring,
		Title: "shop/deployment/api 风险异常", Summary: "风险持续偏高，已升级为事件",
		Labels: map[string]string{alert.LabelNamespace: "shop", alert.LabelSubject: "shop/deployment/api", alert.LabelSubjectType: "deployment"},
		Details: map[string]string{
			alert.DetailIncidentID: "inc-123", alert.DetailRootCause: "shop/pod/api-7d9f", alert.DetailMemberCount: "3",
			alert.DetailPeakRisk: "72", alert.DetailRecurrence: "2", alert.DetailOccurrences24h: "4",
		},
		StartsAt: renderNow.Add(-65 * time.Minute),
	}}
}

func heartbeatItem() Item {
	return Item{Kind: alert.KindFiring, Alert: alert.Alert{
		Fingerprint: "heartbeat:prod", Source: alert.SourceHeartbeat, ClusterID: "prod",
		Severity: alert.SeverityCritical, Status: alert.StatusFiring, Kind: alert.KindFiring,
		Title: "Agent 离线", Summary: "Agent 心跳超时，可能已离线或网络异常",
		Details:  map[string]string{alert.DetailLastHeartbeat: "2026-11-15T11:58:00Z"},
		StartsAt: renderNow.Add(-2 * time.Minute),
	}}
}

func mustRenderer(t *testing.T, publicURL string) *AlertRenderer {
	t.Helper()
	renderer, err := NewAlertRenderer(publicURL)
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}

func assertContains(t *testing.T, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("缺少 %q：\n%s", want, text)
		}
	}
}

func TestRender_SingleSlack(t *testing.T) {
	msg, err := mustRenderer(t, "https://atlhyper.example.com/").Render("slack", []Item{incidentItem(alert.KindFiring)}, renderNow)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject != "[prod] 高 · shop/deployment/api · 风险持续偏高，已升级为事件" {
		t.Errorf("标题应为「[集群] 严重度 · 主体 · 摘要」，实际 %q", msg.Subject)
	}
	if msg.Format != "markdown" {
		t.Errorf("Slack 消息格式应为 markdown，实际 %q", msg.Format)
	}
	assertContains(t, msg.Body,
		"*变化:* 触发", "*主体:* shop/deployment/api", "*根因实体:* shop/pod/api-7d9f", "*成员数:* 3",
		"*峰值风险:* 72", "*持续时间:* 1 小时 5 分钟", "*复发次数:* 2",
		"<https://atlhyper.example.com/aiops/incidents?incident=inc-123|查看事件详情>")
}

func TestRender_SingleEmail(t *testing.T) {
	msg, err := mustRenderer(t, "https://atlhyper.example.com").Render("email", []Item{incidentItem(alert.KindEscalated)}, renderNow)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Format != "text" {
		t.Errorf("邮件格式应为 text，实际 %q", msg.Format)
	}
	assertContains(t, msg.Body, "变化: 升级", "主体: shop/deployment/api", "峰值风险: 72",
		"事件详情: https://atlhyper.example.com/aiops/incidents?incident=inc-123")
	if strings.Contains(msg.Body, "*") || strings.Contains(msg.Body, "<https") {
		t.Errorf("邮件正文不应带 Slack 标记：\n%s", msg.Body)
	}
}

func TestRender_KindTexts(t *testing.T) {
	kinds := map[alert.Kind]string{
		alert.KindFiring: "触发", alert.KindEscalated: "升级", alert.KindRecurred: "复发",
		alert.KindFlapping: "反复", alert.KindResolved: "恢复", alert.KindRepeat: "重复提醒",
	}
	renderer := mustRenderer(t, "")
	for kind, text := range kinds {
		for _, receiverType := range []string{"slack", "email"} {
			msg, err := renderer.Render(receiverType, []Item{incidentItem(kind)}, renderNow)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(msg.Body, "变化:* "+text) && !strings.Contains(msg.Body, "变化: "+text) {
				t.Errorf("%s / %s：正文里应写明变化是「%s」：\n%s", kind, receiverType, text, msg.Body)
			}
		}
	}
	// 反复类通知带上 24 小时内的发生次数
	msg, _ := renderer.Render("slack", []Item{incidentItem(alert.KindFlapping)}, renderNow)
	assertContains(t, msg.Body, "*近 24 小时发生次数:* 4")
}

func TestRender_LinkOnlyWhenPublicURLConfiguredAndIncident(t *testing.T) {
	without, _ := mustRenderer(t, "").Render("slack", []Item{incidentItem(alert.KindFiring)}, renderNow)
	if strings.Contains(without.Body, "查看事件详情") || strings.Contains(without.Body, "/aiops/incidents") {
		t.Errorf("没有配置对外地址时不应带链接：\n%s", without.Body)
	}
	heartbeat, _ := mustRenderer(t, "https://atlhyper.example.com").Render("slack", []Item{heartbeatItem()}, renderNow)
	if strings.Contains(heartbeat.Body, "/aiops/incidents") {
		t.Errorf("心跳告警没有事件详情页：\n%s", heartbeat.Body)
	}
	if heartbeat.Subject != "[prod] 严重 · Agent 离线 · Agent 心跳超时，可能已离线或网络异常" {
		t.Errorf("没有主体的告警用告警标题占位，实际 %q", heartbeat.Subject)
	}
	assertContains(t, heartbeat.Body, "*最后心跳:* 2026-11-15T11:58:00Z")
}

func TestRender_Group(t *testing.T) {
	second := incidentItem(alert.KindResolved)
	second.Alert.Severity = alert.SeverityCritical
	second.Alert.Labels = map[string]string{alert.LabelSubject: "shop/statefulset/db"}
	second.Alert.Summary = "风险已回落"
	ended := renderNow.Add(-5 * time.Minute)
	second.Alert.EndsAt = &ended
	items := []Item{incidentItem(alert.KindFiring), second, heartbeatItem()}

	for _, receiverType := range []string{"slack", "email"} {
		msg, err := mustRenderer(t, "https://atlhyper.example.com").Render(receiverType, items, renderNow)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Subject != "[prod] 3 条告警（最高严重）" {
			t.Errorf("%s：分组标题应为「[集群] N 条告警（最高严重度）」，实际 %q", receiverType, msg.Subject)
		}
		assertContains(t, msg.Body, "shop/deployment/api", "shop/statefulset/db", "Agent 离线", "触发", "恢复", "1 小时")
	}
}

// 告警内容来自集群（K8s 事件消息等），不可信：Slack 里的 <!channel>、链接语法必须被转义
func TestRender_SlackEscapesControlSequences(t *testing.T) {
	item := incidentItem(alert.KindFiring)
	item.Alert.Summary = "<!channel> see <https://phishing.example.com|here> & more"
	msg, _ := mustRenderer(t, "").Render("slack", []Item{item}, renderNow)
	for _, raw := range []string{"<!channel>", "<https://phishing.example.com|here>"} {
		if strings.Contains(msg.Body, raw) || strings.Contains(msg.Subject, raw) {
			t.Errorf("Slack 控制序列没有被转义 %q：\n%s\n%s", raw, msg.Subject, msg.Body)
		}
	}
	assertContains(t, msg.Body, "&lt;!channel&gt;", "&amp; more")
}

// 分组很大时消息要截断：Slack 一个文本块有长度上限，超了整条发不出去
func TestRender_LargeGroupIsTruncated(t *testing.T) {
	var items []Item
	for i := 0; i < 60; i++ {
		item := incidentItem(alert.KindFiring)
		item.Alert.Labels = map[string]string{alert.LabelSubject: fmt.Sprintf("shop/deployment/svc-%02d", i)}
		items = append(items, item)
	}
	msg, err := mustRenderer(t, "").Render("slack", items, renderNow)
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(msg.Body)) > slackBodyLimit {
		t.Errorf("Slack 正文超过上限：%d 个字符", len([]rune(msg.Body)))
	}
	assertContains(t, msg.Subject, "60 条告警")
	assertContains(t, msg.Body, "另有 40 条")
}

func TestRender_UnsupportedReceiverTypeAndEmptyItems(t *testing.T) {
	renderer := mustRenderer(t, "")
	if _, err := renderer.Render("dingtalk", []Item{incidentItem(alert.KindFiring)}, renderNow); err == nil {
		t.Error("不支持的接收人类型应报错")
	}
	if _, err := renderer.Render("slack", nil, renderNow); err == nil {
		t.Error("没有告警时应报错")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{20 * time.Second, "不到 1 分钟"},
		{5 * time.Minute, "5 分钟"},
		{65 * time.Minute, "1 小时 5 分钟"},
		{2 * time.Hour, "2 小时"},
		{49*time.Hour + 30*time.Minute, "2 天 1 小时"},
		{-time.Minute, "不到 1 分钟"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q，期望 %q", c.d, got, c.want)
		}
	}
}

func TestRenderTest(t *testing.T) {
	msg := mustRenderer(t, "").RenderTest("值班频道", renderNow)
	assertContains(t, msg.Subject, "测试")
	assertContains(t, msg.Body, "值班频道")
}

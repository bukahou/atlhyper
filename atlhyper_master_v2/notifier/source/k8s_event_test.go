package source

import (
	"context"
	"strings"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

type fakeEvents struct {
	events []*database.ClusterEvent
}

func (f *fakeEvents) GetLatestEventID(context.Context) (int64, error) {
	var latest int64
	for _, event := range f.events {
		if event.ID > latest {
			latest = event.ID
		}
	}
	return latest, nil
}

func (f *fakeEvents) GetEventsSince(_ context.Context, sinceID int64) ([]*database.ClusterEvent, error) {
	var newer []*database.ClusterEvent
	for _, event := range f.events {
		if event.ID > sinceID {
			newer = append(newer, event)
		}
	}
	return newer, nil
}

func (f *fakeEvents) ListByInvolvedResource(_ context.Context, clusterID, kind, namespace, name string) ([]*database.ClusterEvent, error) {
	var matched []*database.ClusterEvent
	for _, event := range f.events {
		if event.ClusterID == clusterID && event.InvolvedKind == kind && event.InvolvedNamespace == namespace && event.InvolvedName == name {
			matched = append(matched, event)
		}
	}
	return matched, nil
}

func backOffEvent(id int64, lastSeen time.Time) *database.ClusterEvent {
	return &database.ClusterEvent{ID: id, ClusterID: "prod", Reason: "BackOff", Message: "Back-off restarting failed container",
		InvolvedKind: "Pod", InvolvedNamespace: "shop", InvolvedName: "api-1", Count: 7, LastTimestamp: lastSeen}
}

func k8sPolicy(enabled bool) *fakePolicy {
	policy := database.DefaultNotifyPolicy()
	policy.K8sEventAlertsEnabled = enabled
	return &fakePolicy{policy: policy}
}

func newTestK8sEventSource(events *fakeEvents, policy *fakePolicy, firing *fakeFiringAlerts, submitter *fakeSubmitter) *K8sEventSource {
	return NewK8sEventSource(events, policy, firing, submitter, K8sEventConfig{
		CheckInterval: 30 * time.Second, Now: func() time.Time { return sourceNow },
	})
}

func TestK8sEventSource_NewEventBecomesAlert(t *testing.T) {
	events := &fakeEvents{events: []*database.ClusterEvent{backOffEvent(1, sourceNow)}}
	submitter := &fakeSubmitter{}
	src := newTestK8sEventSource(events, k8sPolicy(true), &fakeFiringAlerts{}, submitter)
	src.lastSeenID = 0

	src.check()
	if len(submitter.alerts) != 1 {
		t.Fatalf("新事件应产生 1 条告警，实际 %d", len(submitter.alerts))
	}
	got := submitter.alerts[0]
	if got.Fingerprint != "k8s_event:prod:Pod/shop/api-1:BackOff" || got.Source != alert.SourceK8sEvent || got.Severity != alert.SeverityLow ||
		got.Status != alert.StatusFiring || got.Kind != alert.KindFiring || got.ClusterID != "prod" ||
		got.Labels[alert.LabelNamespace] != "shop" || got.Labels[alert.LabelSubject] != "shop/pod/api-1" ||
		got.Details[alert.DetailEventReason] != "BackOff" || got.Details[alert.DetailEventCount] != "7" ||
		!strings.Contains(got.Summary, "Back-off") {
		t.Errorf("告警内容不对: %+v", got)
	}

	// 同一条事件不会重复提交
	src.check()
	if len(submitter.alerts) != 1 {
		t.Errorf("已处理过的事件不应再提交，实际 %d 条", len(submitter.alerts))
	}
}

// 开关默认关：不产生告警；关着的期间出现的事件，之后打开开关也不补报
func TestK8sEventSource_DisabledByPolicy(t *testing.T) {
	events := &fakeEvents{events: []*database.ClusterEvent{backOffEvent(1, sourceNow)}}
	policy := k8sPolicy(false)
	submitter := &fakeSubmitter{}
	src := newTestK8sEventSource(events, policy, &fakeFiringAlerts{}, submitter)
	src.lastSeenID = 0

	src.check()
	if len(submitter.alerts) != 0 {
		t.Fatalf("开关关闭时不应产生告警: %+v", submitter.alerts)
	}

	policy.policy.K8sEventAlertsEnabled = true
	events.events = append(events.events, backOffEvent(2, sourceNow))
	events.events[1].InvolvedName = "api-2"
	src.check()
	if len(submitter.alerts) != 1 || !strings.Contains(submitter.alerts[0].Fingerprint, "api-2") {
		t.Errorf("打开开关后只报新出现的事件: %+v", submitter.alerts)
	}
}

func k8sAlertRow(updatedAt time.Time) *database.NotifyAlert {
	return &database.NotifyAlert{
		Fingerprint: "k8s_event:prod:Pod/shop/api-1:BackOff", Source: string(alert.SourceK8sEvent), ClusterID: "prod",
		Severity: "low", Status: "firing", Kind: "firing", Title: "Pod shop/api-1 BackOff",
		Labels:    map[string]string{alert.LabelNamespace: "shop", alert.LabelSubject: "shop/pod/api-1"},
		Details:   map[string]string{alert.DetailEventKind: "Pod", alert.DetailEventName: "api-1", alert.DetailEventReason: "BackOff"},
		UpdatedAt: updatedAt,
	}
}

// 10 分钟内没有再出现：安静结束（不发恢复通知）
func TestK8sEventSource_ExpiresQuietlyAfterTenMinutes(t *testing.T) {
	events := &fakeEvents{events: []*database.ClusterEvent{backOffEvent(1, sourceNow.Add(-11*time.Minute))}}
	submitter := &fakeSubmitter{}

	fresh := &fakeFiringAlerts{rows: []*database.NotifyAlert{k8sAlertRow(sourceNow.Add(-9 * time.Minute))}}
	src := newTestK8sEventSource(events, k8sPolicy(true), fresh, submitter)
	src.lastSeenID = 1
	src.check()
	if len(submitter.alerts) != 0 {
		t.Fatalf("没到 10 分钟不应结束: %+v", submitter.alerts)
	}

	overdue := &fakeFiringAlerts{rows: []*database.NotifyAlert{k8sAlertRow(sourceNow.Add(-10 * time.Minute))}}
	src = newTestK8sEventSource(events, k8sPolicy(true), overdue, submitter)
	src.lastSeenID = 1
	src.check()
	if len(submitter.alerts) != 1 {
		t.Fatalf("超过 10 分钟没有再出现应结束，实际提交 %d 条", len(submitter.alerts))
	}
	got := submitter.alerts[0]
	if got.Status != alert.StatusResolved || !got.Quiet || got.Fingerprint != "k8s_event:prod:Pod/shop/api-1:BackOff" {
		t.Errorf("应是安静结束: %+v", got)
	}
}

// 事件在这 10 分钟里又出现过（K8s 只更新同一行的时间与次数）：告警续期，不结束、也不另发通知
func TestK8sEventSource_RecurringEventKeepsAlertAlive(t *testing.T) {
	events := &fakeEvents{events: []*database.ClusterEvent{backOffEvent(1, sourceNow.Add(-2*time.Minute))}}
	overdue := &fakeFiringAlerts{rows: []*database.NotifyAlert{k8sAlertRow(sourceNow.Add(-10 * time.Minute))}}
	submitter := &fakeSubmitter{}
	src := newTestK8sEventSource(events, k8sPolicy(true), overdue, submitter)
	src.lastSeenID = 1

	src.check()
	if len(submitter.alerts) != 1 || submitter.alerts[0].Status != alert.StatusFiring ||
		submitter.alerts[0].Fingerprint != "k8s_event:prod:Pod/shop/api-1:BackOff" {
		t.Errorf("事件还在出现时应续期而不是结束: %+v", submitter.alerts)
	}
}

// 关掉开关后，已有的 K8s 事件告警到点照样安静结束（不去查它是否还在出现）
func TestK8sEventSource_DisabledStillExpiresExistingAlerts(t *testing.T) {
	events := &fakeEvents{events: []*database.ClusterEvent{backOffEvent(1, sourceNow.Add(-time.Minute))}}
	overdue := &fakeFiringAlerts{rows: []*database.NotifyAlert{k8sAlertRow(sourceNow.Add(-10 * time.Minute))}}
	submitter := &fakeSubmitter{}
	src := newTestK8sEventSource(events, k8sPolicy(false), overdue, submitter)
	src.lastSeenID = 1

	src.check()
	if len(submitter.alerts) != 1 || submitter.alerts[0].Status != alert.StatusResolved || !submitter.alerts[0].Quiet {
		t.Errorf("开关关闭后已有告警应安静结束: %+v", submitter.alerts)
	}
}

func TestK8sEventAlert_ClusterScopedAndLongMessage(t *testing.T) {
	event := &database.ClusterEvent{ClusterID: "prod", Reason: "NodeNotReady", Message: strings.Repeat("长", 500),
		InvolvedKind: "Node", InvolvedName: "worker-1", Count: 1}
	got := K8sEventAlert(event)
	if got.Fingerprint != "k8s_event:prod:Node//worker-1:NodeNotReady" || got.Labels[alert.LabelNamespace] != "" ||
		got.Labels[alert.LabelSubject] != "_cluster/node/worker-1" || got.Labels[alert.LabelSubjectType] != "node" {
		t.Errorf("集群级资源的告警不对: %+v", got)
	}
	if len([]rune(got.Summary)) > maxEventSummaryRunes+1 {
		t.Errorf("过长的事件消息应截断，实际 %d 个字符", len([]rune(got.Summary)))
	}
}

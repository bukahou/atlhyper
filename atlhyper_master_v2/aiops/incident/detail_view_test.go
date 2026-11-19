package incident

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

var viewT0 = time.Date(2026, 11, 15, 5, 53, 0, 0, time.UTC)

func TestSignalOf(t *testing.T) {
	cases := map[string]string{
		"container_anomaly": aiops.SignalContainerCrash, "deployment_impact": aiops.SignalWorkloadUnavailable,
		"critical_event": aiops.SignalK8sEvent, "restart_count": aiops.SignalRestarts, "max_container_restarts": aiops.SignalRestarts,
		"not_ready_containers": aiops.SignalNotReady, "is_running": aiops.SignalNotReady,
		"cpu_usage": aiops.SignalResource, "psi_io": aiops.SignalResource, "disk_usage": aiops.SignalResource,
		"memory_pressure": aiops.SignalNodePressure, "pid_pressure": aiops.SignalNodePressure,
		"apm_high_error_rate": aiops.SignalAPMErrors, "apm_error_rate": aiops.SignalAPMErrors, "error_rate": aiops.SignalAPMErrors,
		"apm_high_p99_latency": aiops.SignalAPMLatency, "avg_latency": aiops.SignalAPMLatency,
		"log_error_spike": aiops.SignalLogErrors, "log_error_count": aiops.SignalLogErrors,
		"something_new": aiops.SignalOther,
	}
	for metric, want := range cases {
		if got := SignalOf(metric); got != want {
			t.Errorf("%s → %s，期望 %s", metric, got, want)
		}
	}
}

func TestCleanSignalDetail_StripsPodUID(t *testing.T) {
	raw := "BackOff: Back-off restarting failed container crasher in pod api-57464786c8-n9shq_atlhyper-dev(8a498a3e-ef21-4b20-982f-1b69ce501aab)"
	want := "BackOff: Back-off restarting failed container crasher in pod api-57464786c8-n9shq_atlhyper-dev"
	if got := cleanSignalDetail(raw); got != want {
		t.Errorf("得到 %q", got)
	}
	if got := cleanSignalDetail("容器 crasher: CrashLoopBackOff"); got != "容器 crasher: CrashLoopBackOff" {
		t.Errorf("没有 UID 时应原样保留，得到 %q", got)
	}
}

func TestSignalReason(t *testing.T) {
	cases := []struct{ signal, detail, want string }{
		{aiops.SignalContainerCrash, "容器 crasher: CrashLoopBackOff", "CrashLoopBackOff"},
		{aiops.SignalContainerCrash, "容器 app: OOMKilled", "OOMKilled"},
		{aiops.SignalWorkloadUnavailable, "Deployment demo/x 不可用 67%", "67"},
		{aiops.SignalK8sEvent, "BackOff: Back-off restarting failed container", "BackOff"},
		{aiops.SignalResource, "CPU 92%", ""},
		// 实际落库的原文带类别前缀（aiops.DescribeAnomaly 加的）
		{aiops.SignalContainerCrash, "容器异常: 容器 crasher: CrashLoopBackOff", "CrashLoopBackOff"},
		{aiops.SignalWorkloadUnavailable, "Deployment 影响: Deployment demo/x 不可用 100%", "100"},
		{aiops.SignalK8sEvent, "K8s 事件 BackOff: Back-off restarting failed container crasher in pod a_ns", "BackOff"},
		{aiops.SignalContainerCrash, "看不懂的格式", ""},
	}
	for _, c := range cases {
		if got := signalReason(c.signal, c.detail); got != c.want {
			t.Errorf("signalReason(%s, %q) = %q，期望 %q", c.signal, c.detail, got, c.want)
		}
	}
}

func sig(entity, metric string, first, last time.Duration, hits int, score float64, detail string) *database.AIOpsIncidentSignal {
	return &database.AIOpsIncidentSignal{IncidentID: "inc-1", EntityKey: entity, Metric: metric,
		FirstSeen: viewT0.Add(first), LastSeen: viewT0.Add(last), Hits: hits, MaxScore: score, Detail: detail}
}

// 同一信号类型跨成员、跨指标合并：中招成员数去重、首次取最早、最近取最晚、次数相加、最高分取大、原文取最近一条；按最高分排序
func TestBuildEvidence_GroupsBySignal(t *testing.T) {
	signals := []*database.AIOpsIncidentSignal{
		sig("ns/pod/a", "container_anomaly", 0, 7*time.Minute, 8, 0.9, "容器 crasher: CrashLoopBackOff"),
		sig("ns/pod/b", "container_anomaly", time.Minute, 6*time.Minute, 7, 0.8, "容器 crasher: OOMKilled"),
		sig("ns/pod/c", "container_anomaly", 2*time.Minute, 5*time.Minute, 4, 0.8, "容器 crasher: CrashLoopBackOff"),
		sig("ns/pod/a", "deployment_impact", 0, 7*time.Minute, 8, 0.95, "Deployment ns/x 不可用 100%"),
		sig("ns/pod/a", "critical_event", 0, 7*time.Minute, 8, 0.85,
			"BackOff: Back-off restarting failed container crasher in pod a_ns(8a498a3e-ef21-4b20-982f-1b69ce501aab)"),
		sig("ns/pod/b", "restart_count", 0, time.Minute, 1, 0.2, "重启 3 次"),
		sig("ns/pod/b", "max_container_restarts", 0, 2*time.Minute, 2, 0.3, "最大重启 3 次"),
	}
	evidence := BuildEvidence(signals)
	if len(evidence) != 4 {
		t.Fatalf("应合并成 4 类，实际 %d: %+v", len(evidence), evidence)
	}
	order := []string{aiops.SignalWorkloadUnavailable, aiops.SignalContainerCrash, aiops.SignalK8sEvent, aiops.SignalRestarts}
	for i, want := range order {
		if evidence[i].Signal != want {
			t.Fatalf("第 %d 条应是 %s，实际 %s", i, want, evidence[i].Signal)
		}
	}
	crash := evidence[1]
	if crash.AffectedMembers != 3 || !crash.FirstSeen.Equal(viewT0) || !crash.LastSeen.Equal(viewT0.Add(7*time.Minute)) ||
		crash.Hits != 19 || crash.MaxScore != 0.9 || crash.Detail != "容器 crasher: CrashLoopBackOff" || crash.Reason != "CrashLoopBackOff" {
		t.Errorf("容器崩溃这一类合并不对: %+v", crash)
	}
	if evidence[0].Reason != "100" || evidence[0].AffectedMembers != 1 {
		t.Errorf("不可用比例应解析为 100: %+v", evidence[0])
	}
	if evidence[2].Reason != "BackOff" || evidence[2].Detail != "BackOff: Back-off restarting failed container crasher in pod a_ns" {
		t.Errorf("K8s 事件的原因与去 UID 不对: %+v", evidence[2])
	}
	if restarts := evidence[3]; restarts.AffectedMembers != 1 || restarts.Hits != 3 {
		t.Errorf("同一成员的两个重启指标算一个成员、次数相加: %+v", restarts)
	}

	if got := BuildEvidence(nil); got == nil || len(got) != 0 {
		t.Errorf("没有信号时应是空切片，实际 %v", got)
	}
}

func TestBuildHeadline(t *testing.T) {
	if BuildHeadline(nil, 3) != nil {
		t.Error("没有证据时结论应为 nil")
	}
	evidence := []aiops.IncidentEvidence{{Signal: aiops.SignalContainerCrash, Reason: "CrashLoopBackOff", AffectedMembers: 3}}
	got := BuildHeadline(evidence, 3)
	if got == nil || got.Signal != aiops.SignalContainerCrash || got.Reason != "CrashLoopBackOff" || got.AffectedMembers != 3 || got.TotalMembers != 3 {
		t.Errorf("结论不对: %+v", got)
	}
}

func entry(offset time.Duration, eventType, detail string) *aiops.IncidentTimeline {
	return &aiops.IncidentTimeline{Timestamp: viewT0.Add(offset), EventType: eventType, Detail: detail}
}

// 状态与严重度的变化逐条转成结构化记录；检测记录合并成一条放在最后
func TestBuildTimelineView(t *testing.T) {
	timeline := []*aiops.IncidentTimeline{
		entry(0, aiops.TimelineStateChange, "状态变更: healthy → warning"),
		entry(0, aiops.TimelineAnomalyDetected, "容器 crasher: CrashLoopBackOff"),
		entry(30*time.Second, aiops.TimelineSeverityRaised, "严重度升高 → high (风险 77)"),
		entry(30*time.Second, aiops.TimelineAnomalyDetected, "x"),
		entry(5*time.Minute, aiops.TimelineStateChange, "状态变更 → incident (风险 88)"),
		entry(5*time.Minute, aiops.TimelineAnomalyDetected, "x"),
		entry(15*time.Minute, aiops.TimelineRecoveryStarted, "状态变更 → recovery (对象已删除)"),
		entry(20*time.Minute, aiops.TimelineRecurrence, "复发 (风险 60)"),
		entry(25*time.Minute, aiops.TimelineRootCauseIdentified, "旧版本写下的记录"),
		entry(30*time.Minute, aiops.TimelineStateChange, "事件已关闭: 恢复观察期满"),
		entry(31*time.Minute, aiops.TimelineStateChange, "事件已关闭: 主体长时间没有数据（对象已删除或 Agent 断连）"),
	}
	view := BuildTimelineView(timeline)
	kinds := []string{aiops.TimelineItemCreated, aiops.TimelineItemSeverityRaised, aiops.TimelineItemEscalated,
		aiops.TimelineItemRecoveryStarted, aiops.TimelineItemRecurred, aiops.TimelineItemNote,
		aiops.TimelineItemClosed, aiops.TimelineItemClosedStale, aiops.TimelineItemAnomaly}
	if len(view) != len(kinds) {
		t.Fatalf("应有 %d 条，实际 %d: %+v", len(kinds), len(view), view)
	}
	for i, want := range kinds {
		if view[i].Kind != want {
			t.Errorf("第 %d 条应是 %s，实际 %s", i, want, view[i].Kind)
		}
	}
	if view[1].Severity != "high" || view[1].Risk == nil || *view[1].Risk != 77 {
		t.Errorf("严重度升高应带档位与风险: %+v", view[1])
	}
	if view[2].Risk == nil || *view[2].Risk != 88 || view[4].Risk == nil || *view[4].Risk != 60 {
		t.Errorf("升级 / 复发应带风险: %+v %+v", view[2], view[4])
	}
	if view[3].Reason != TimelineReasonSubjectDeleted || view[3].Risk != nil {
		t.Errorf("对象删除引起的恢复应带原因码、不带风险: %+v", view[3])
	}
	if view[0].Risk != nil {
		t.Errorf("开事件的记录里没有风险，不应编一个: %+v", view[0])
	}
	if view[5].Detail != "旧版本写下的记录" {
		t.Errorf("认不出的类型原文放在 detail: %+v", view[5])
	}
	anomaly := view[8]
	if anomaly.Count != 3 || anomaly.FirstAt == nil || !anomaly.FirstAt.Equal(viewT0) || anomaly.LastAt == nil ||
		!anomaly.LastAt.Equal(viewT0.Add(5*time.Minute)) || !anomaly.At.Equal(viewT0) {
		t.Errorf("检测记录应合并成一条: %+v", anomaly)
	}

	if got := BuildTimelineView(nil); got == nil || len(got) != 0 {
		t.Errorf("没有时间线时应是空切片，实际 %v", got)
	}
}

func TestBuildEvidenceStatus(t *testing.T) {
	recorded := []aiops.IncidentEvidence{{Signal: aiops.SignalContainerCrash}}
	propagation := entry(0, aiops.TimelineAnomalyDetected, propagationOnlyDetail)
	local := entry(0, aiops.TimelineAnomalyDetected, "容器异常: 容器 app: CrashLoopBackOff")
	state := entry(0, aiops.TimelineStateChange, "状态变更: healthy → warning")

	cases := []struct {
		name     string
		evidence []aiops.IncidentEvidence
		timeline []*aiops.IncidentTimeline
		want     string
	}{
		{"有证据", recorded, []*aiops.IncidentTimeline{local}, aiops.EvidenceRecorded},
		{"检测记录全是依赖传播", nil, []*aiops.IncidentTimeline{state, propagation, propagation}, aiops.EvidencePropagationOnly},
		{"有本地信号的检测记录但没有证据行：改版前", nil, []*aiops.IncidentTimeline{propagation, local}, aiops.EvidenceNotRecorded},
		{"连检测记录都没有：更早的事件", nil, []*aiops.IncidentTimeline{state}, aiops.EvidenceNotRecorded},
	}
	for _, c := range cases {
		if got := BuildEvidenceStatus(c.evidence, c.timeline); got != c.want {
			t.Errorf("%s：得到 %s，期望 %s", c.name, got, c.want)
		}
	}
}

// 工作负载不可用排在最前、同时有容器崩溃：结论用崩溃作主证据，不可用比例作为影响带上
func TestBuildHeadline_CauseWithImpact(t *testing.T) {
	evidence := []aiops.IncidentEvidence{
		{Signal: aiops.SignalWorkloadUnavailable, Reason: "100", AffectedMembers: 3, MaxScore: 0.95},
		{Signal: aiops.SignalContainerCrash, Reason: "CrashLoopBackOff", AffectedMembers: 3, MaxScore: 0.9},
		{Signal: aiops.SignalK8sEvent, Reason: "BackOff", AffectedMembers: 3, MaxScore: 0.85},
	}
	got := BuildHeadline(evidence, 3)
	if got.Signal != aiops.SignalContainerCrash || got.Reason != "CrashLoopBackOff" || got.UnavailablePercent == nil || *got.UnavailablePercent != 100 {
		t.Errorf("应以崩溃为主证据并带上不可用 100%%: %+v", got)
	}
	onlyUnavailable := BuildHeadline(evidence[:1], 3)
	if onlyUnavailable.Signal != aiops.SignalWorkloadUnavailable || onlyUnavailable.UnavailablePercent != nil {
		t.Errorf("只有不可用时照旧: %+v", onlyUnavailable)
	}
}

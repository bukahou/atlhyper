package alert

import "testing"

func TestParseSeverity(t *testing.T) {
	cases := []struct {
		input  string
		want   Severity
		wantOK bool
	}{
		{"low", SeverityLow, true},
		{"medium", SeverityMedium, true},
		{"high", SeverityHigh, true},
		{"critical", SeverityCritical, true},
		{"", "", false},
		{"warning", "", false},
		{"HIGH", "", false},
	}
	for _, c := range cases {
		got, ok := ParseSeverity(c.input)
		if got != c.want || ok != c.wantOK {
			t.Errorf("ParseSeverity(%q) = (%q, %v)，期望 (%q, %v)", c.input, got, ok, c.want, c.wantOK)
		}
	}
}

func TestSeverity_AtLeast(t *testing.T) {
	cases := []struct {
		severity, min Severity
		want          bool
	}{
		{SeverityLow, SeverityLow, true},
		{SeverityLow, SeverityMedium, false},
		{SeverityHigh, SeverityMedium, true},
		{SeverityCritical, SeverityCritical, true},
		{SeverityHigh, SeverityCritical, false},
		{"unknown", SeverityLow, false}, // 认不出的严重度不满足任何门槛
		{SeverityLow, "unknown", true},  // 认不出的门槛按最低处理
	}
	for _, c := range cases {
		if got := c.severity.AtLeast(c.min); got != c.want {
			t.Errorf("%q.AtLeast(%q) = %v，期望 %v", c.severity, c.min, got, c.want)
		}
	}
}

func TestMaxSeverity(t *testing.T) {
	if got := MaxSeverity(SeverityLow, SeverityHigh); got != SeverityHigh {
		t.Errorf("期望 high，实际 %q", got)
	}
	if got := MaxSeverity(SeverityCritical, SeverityMedium); got != SeverityCritical {
		t.Errorf("期望 critical，实际 %q", got)
	}
}

func TestFingerprints(t *testing.T) {
	if got := IncidentFingerprint("inc-123"); got != "incident:inc-123" {
		t.Errorf("事件指纹不对: %q", got)
	}
	if got := HeartbeatFingerprint("prod"); got != "heartbeat:prod" {
		t.Errorf("心跳指纹不对: %q", got)
	}
	got := K8sEventFingerprint("prod", "Pod", "shop", "api-1", "BackOff")
	if got != "k8s_event:prod:Pod/shop/api-1:BackOff" {
		t.Errorf("K8s 事件指纹不对: %q", got)
	}
}

func TestKind_Opens(t *testing.T) {
	opens := map[Kind]bool{
		KindFiring: true, KindRecurred: true, KindFlapping: true,
		KindEscalated: false, KindResolved: false, KindRepeat: false,
	}
	for kind, want := range opens {
		if got := kind.Opens(); got != want {
			t.Errorf("%q.Opens() = %v，期望 %v", kind, got, want)
		}
	}
}

func TestAlert_LabelAndDetailAccessorsTolerateNilMaps(t *testing.T) {
	var a Alert
	if a.Label(LabelNamespace) != "" || a.Detail(DetailIncidentID) != "" {
		t.Error("空告警的标签与详情应为空串")
	}
	a.Labels = map[string]string{LabelSubject: "shop/deployment/api"}
	a.Details = map[string]string{DetailIncidentID: "inc-1"}
	if a.Label(LabelSubject) != "shop/deployment/api" || a.Detail(DetailIncidentID) != "inc-1" {
		t.Errorf("取值不对: %+v", a)
	}
}

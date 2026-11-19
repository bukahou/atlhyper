package convert

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

// 严重程度由后端定：Warning 里会直接导致故障的原因判为 critical；前端只按它上色
func TestEventSeverity(t *testing.T) {
	cases := []struct {
		eventType, reason string
		want              string
	}{
		{"Warning", "OOMKilling", model.EventSeverityCritical},
		{"Warning", "CrashLoopBackOff", model.EventSeverityCritical},
		{"Warning", "BackOff", model.EventSeverityWarning},
		{"Normal", "Scheduled", model.EventSeverityInfo},
		{"Normal", "OOMKilling", model.EventSeverityInfo}, // 只有 Warning 才升级
		{"Error", "Anything", model.EventSeverityError},
	}
	for _, c := range cases {
		if got := eventSeverity(c.eventType, c.reason); got != c.want {
			t.Errorf("eventSeverity(%q, %q) = %q, want %q", c.eventType, c.reason, got, c.want)
		}
	}
}

func TestEventCardsOf(t *testing.T) {
	rows := EventLogs([]cluster.Event{
		{Type: "Warning", Reason: "OOMKilling", InvolvedObject: model_v3.ResourceRef{Kind: "Pod"}},
		{Type: "Warning", Reason: "BackOff", InvolvedObject: model_v3.ResourceRef{Kind: "Pod"}},
		{Type: "Normal", Reason: "Scheduled", InvolvedObject: model_v3.ResourceRef{Kind: "Node"}},
	}, "c1")
	cards := EventCardsOf(rows)
	if cards.TotalEvents != 3 || cards.Critical != 1 || cards.Warning != 1 || cards.Info != 1 {
		t.Errorf("cards = %+v", cards)
	}
	if cards.TotalAlerts != 2 || cards.KindsCount != 2 || cards.CategoriesCount != 3 {
		t.Errorf("cards = %+v", cards)
	}
}

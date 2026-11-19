package convert

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

func TestEventLogFromModel_FieldMapping(t *testing.T) {
	src := cluster.Event{
		CommonMeta: model_v3.CommonMeta{
			UID:       "uid-1",
			Name:      "event-1",
			Namespace: "default",
			Kind:      "Event",
			NodeName:  "node-a",
			CreatedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		Type:    "Warning",
		Reason:  "FailedScheduling",
		Message: "0/6 nodes are available",
		Source:  "scheduler",
		InvolvedObject: model_v3.ResourceRef{
			Kind:      "Pod",
			Namespace: "default",
			Name:      "test-pod",
		},
		Count:          3,
		FirstTimestamp: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		LastTimestamp:  time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	result := EventLog(&src, "cluster-1")

	if result.ClusterID != "cluster-1" {
		t.Errorf("ClusterID: got %q, want %q", result.ClusterID, "cluster-1")
	}
	if result.Category != "FailedScheduling" {
		t.Errorf("Category: got %q, want %q", result.Category, "FailedScheduling")
	}
	if result.Kind != "Pod" {
		t.Errorf("Kind: got %q, want %q (should use involved_object.kind)", result.Kind, "Pod")
	}
	if result.Name != "test-pod" {
		t.Errorf("Name: got %q, want %q (should use involved_object.name)", result.Name, "test-pod")
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace: got %q, want %q", result.Namespace, "default")
	}
	// 来源组件、发生次数、来源主机各自一个字段（此前组件名被放进 node，次数没有下发）
	if result.Source != "scheduler" || result.Count != 3 || result.Node != "node-a" {
		t.Errorf("Source/Count/Node: got %q/%d/%q, want scheduler/3/node-a", result.Source, result.Count, result.Node)
	}
	if result.Severity != "critical" { // Warning + FailedScheduling 属于会直接导致故障的原因
		t.Errorf("Severity: got %q, want %q", result.Severity, "critical")
	}
	if result.Message != "0/6 nodes are available" {
		t.Errorf("Message: got %q, want %q", result.Message, "0/6 nodes are available")
	}
	if result.Reason != "FailedScheduling" {
		t.Errorf("Reason: got %q, want %q", result.Reason, "FailedScheduling")
	}
	// EventTime = last_timestamp, Time = first_timestamp
	if result.EventTime != "2025-01-15T10:00:00Z" {
		t.Errorf("EventTime: got %q, want %q", result.EventTime, "2025-01-15T10:00:00Z")
	}
	if result.Time != "2025-01-15T09:00:00Z" {
		t.Errorf("Time: got %q, want %q", result.Time, "2025-01-15T09:00:00Z")
	}
}

func TestEventLogFromModel_SeverityMapping(t *testing.T) {
	tests := []struct {
		eventType string
		want      string
	}{
		{"Warning", "warning"},
		{"Normal", "info"},
		{"Error", "error"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			src := cluster.Event{Type: tt.eventType}
			result := EventLog(&src, "c1")
			if result.Severity != tt.want {
				t.Errorf("Type=%q: got severity %q, want %q", tt.eventType, result.Severity, tt.want)
			}
		})
	}
}

func TestEventLogFromModel_FallbackNames(t *testing.T) {
	// When InvolvedObject is empty, fall back to event's own fields
	src := cluster.Event{
		CommonMeta: model_v3.CommonMeta{
			Name: "event-name",
		},
		Type: "Normal",
	}

	result := EventLog(&src, "c1")
	if result.Kind != "Event" {
		t.Errorf("Kind fallback: got %q, want %q", result.Kind, "Event")
	}
	if result.Name != "event-name" {
		t.Errorf("Name fallback: got %q, want %q", result.Name, "event-name")
	}
}

func TestEventLogFromDB_FieldMapping(t *testing.T) {
	src := &database.ClusterEvent{
		DedupKey:          "dedup-1",
		ClusterID:         "cluster-2",
		Namespace:         "kube-system",
		Name:              "coredns-abc",
		Type:              "Warning",
		Reason:            "Unhealthy",
		Message:           "Liveness probe failed",
		SourceComponent:   "kubelet",
		SourceHost:        "node-b",
		InvolvedKind:      "Pod",
		InvolvedName:      "coredns-pod",
		InvolvedNamespace: "kube-system",
		FirstTimestamp:    time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC),
		LastTimestamp:     time.Date(2025, 2, 1, 9, 0, 0, 0, time.UTC),
		Count:             5,
		CreatedAt:         time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC),
	}

	result := EventLogFromDB(src, "cluster-2")

	if result.ClusterID != "cluster-2" {
		t.Errorf("ClusterID: got %q, want %q", result.ClusterID, "cluster-2")
	}
	if result.Kind != "Pod" {
		t.Errorf("Kind: got %q, want %q", result.Kind, "Pod")
	}
	if result.Name != "coredns-pod" {
		t.Errorf("Name: got %q, want %q", result.Name, "coredns-pod")
	}
	if result.Namespace != "kube-system" {
		t.Errorf("Namespace: got %q, want %q", result.Namespace, "kube-system")
	}
	if result.Severity != "warning" {
		t.Errorf("Severity: got %q, want %q", result.Severity, "warning")
	}
	if result.Source != "kubelet" || result.Count != 5 || result.Node != "node-b" {
		t.Errorf("Source/Count/Node: got %q/%d/%q, want kubelet/5/node-b", result.Source, result.Count, result.Node)
	}
	if result.EventTime != "2025-02-01T09:00:00Z" {
		t.Errorf("EventTime: got %q, want %q", result.EventTime, "2025-02-01T09:00:00Z")
	}
}

func TestEventLogs_Plural(t *testing.T) {
	events := []cluster.Event{
		{CommonMeta: model_v3.CommonMeta{Name: "e1"}, Type: "Normal"},
		{CommonMeta: model_v3.CommonMeta{Name: "e2"}, Type: "Warning"},
	}

	result := EventLogs(events, "c1")
	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
	if result[0].Severity != "info" {
		t.Errorf("[0].Severity: got %q, want %q", result[0].Severity, "info")
	}
	if result[1].Severity != "warning" {
		t.Errorf("[1].Severity: got %q, want %q", result[1].Severity, "warning")
	}
}

func TestEventLogs_NilInput(t *testing.T) {
	result := EventLogs(nil, "c1")
	if result == nil {
		t.Error("should return non-nil empty slice")
	}
}

func TestEventLogsFromDB_Plural(t *testing.T) {
	events := []*database.ClusterEvent{
		{Name: "e1", Type: "Warning", Reason: "OOM"},
		{Name: "e2", Type: "Normal", Reason: "Pulled"},
	}
	result := EventLogsFromDB(events, "c1")
	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
}

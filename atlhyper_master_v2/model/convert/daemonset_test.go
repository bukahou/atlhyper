package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestDaemonSetItem_FieldMapping(t *testing.T) {
	src := &model_v2.DaemonSet{
		Summary: model_v2.DaemonSetSummary{
			Name:                   "fluent-bit",
			Namespace:              "logging",
			DesiredNumberScheduled: 5,
			CurrentNumberScheduled: 5,
			NumberReady:            4,
			NumberAvailable:        4,
			NumberMisscheduled:     0,
			CreatedAt:              time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			Age:                    "90d",
		},
	}

	item := DaemonSetItem(src)

	if item.Name != "fluent-bit" {
		t.Errorf("Name = %q, want %q", item.Name, "fluent-bit")
	}
	if item.Namespace != "logging" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "logging")
	}
	if item.Desired != 5 {
		t.Errorf("Desired = %d, want 5", item.Desired)
	}
	if item.Current != 5 {
		t.Errorf("Current = %d, want 5", item.Current)
	}
	if item.Ready != 4 {
		t.Errorf("Ready = %d, want 4", item.Ready)
	}
	if item.Available != 4 {
		t.Errorf("Available = %d, want 4", item.Available)
	}
	if item.Misscheduled != 0 {
		t.Errorf("Misscheduled = %d, want 0", item.Misscheduled)
	}
	if item.CreatedAt != "2024-02-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want RFC3339", item.CreatedAt)
	}
	if item.Age != "90d" {
		t.Errorf("Age = %q, want %q", item.Age, "90d")
	}
}

func TestDaemonSetItems_NilReturnsEmpty(t *testing.T) {
	result := DaemonSetItems(nil)
	if result == nil {
		t.Error("DaemonSetItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestDaemonSetItems_Multiple(t *testing.T) {
	src := []model_v2.DaemonSet{
		{Summary: model_v2.DaemonSetSummary{Name: "a", CreatedAt: time.Now()}},
		{Summary: model_v2.DaemonSetSummary{Name: "b", CreatedAt: time.Now()}},
	}
	result := DaemonSetItems(src)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Name != "a" || result[1].Name != "b" {
		t.Errorf("names = [%q, %q], want [a, b]", result[0].Name, result[1].Name)
	}
}

func TestDaemonSetDetail_FieldMapping(t *testing.T) {
	src := &model_v2.DaemonSet{
		Summary: model_v2.DaemonSetSummary{
			Name:                   "fluent-bit",
			Namespace:              "logging",
			DesiredNumberScheduled: 5,
			CurrentNumberScheduled: 5,
			NumberReady:            4,
			NumberAvailable:        4,
			NumberUnavailable:      1,
			NumberMisscheduled:     0,
			UpdatedNumberScheduled: 5,
			CreatedAt:              time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			Age:                    "90d",
			Selector:               "app=fluent-bit",
		},
		Labels: map[string]string{"app": "fluent-bit"},
	}

	d := DaemonSetDetail(src)

	if d.Name != "fluent-bit" {
		t.Errorf("Name = %q, want %q", d.Name, "fluent-bit")
	}
	if d.Desired != 5 {
		t.Errorf("Desired = %d, want 5", d.Desired)
	}
	if d.Current != 5 {
		t.Errorf("Current = %d, want 5", d.Current)
	}
	if d.Ready != 4 {
		t.Errorf("Ready = %d, want 4", d.Ready)
	}
	if d.Available != 4 {
		t.Errorf("Available = %d, want 4", d.Available)
	}
	if d.Unavailable != 1 {
		t.Errorf("Unavailable = %d, want 1", d.Unavailable)
	}
	if d.Misscheduled != 0 {
		t.Errorf("Misscheduled = %d, want 0", d.Misscheduled)
	}
	if d.UpdatedScheduled != 5 {
		t.Errorf("UpdatedScheduled = %d, want 5", d.UpdatedScheduled)
	}
	if d.Age != "90d" {
		t.Errorf("Age = %q, want %q", d.Age, "90d")
	}
}

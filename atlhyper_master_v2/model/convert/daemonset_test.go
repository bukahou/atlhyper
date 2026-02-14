package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

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

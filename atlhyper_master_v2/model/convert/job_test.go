package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestJobItem_FieldMapping(t *testing.T) {
	start := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	finish := time.Date(2026, 2, 13, 10, 5, 32, 0, time.UTC)
	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			Name:      "data-migration-v2",
			Namespace: "default",
			CreatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
		},
		Active:     0,
		Succeeded:  1,
		Failed:     0,
		Complete:   true,
		StartTime:  &start,
		FinishTime: &finish,
	}

	item := JobItem(src)

	if item.Name != "data-migration-v2" {
		t.Errorf("Name = %q, want %q", item.Name, "data-migration-v2")
	}
	if item.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "default")
	}
	if item.Active != 0 {
		t.Errorf("Active = %d, want 0", item.Active)
	}
	if item.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", item.Succeeded)
	}
	if item.Failed != 0 {
		t.Errorf("Failed = %d, want 0", item.Failed)
	}
	if !item.Complete {
		t.Error("Complete = false, want true")
	}
	if item.StartTime != "2026-02-13T10:00:00Z" {
		t.Errorf("StartTime = %q, want %q", item.StartTime, "2026-02-13T10:00:00Z")
	}
	if item.FinishTime != "2026-02-13T10:05:32Z" {
		t.Errorf("FinishTime = %q, want %q", item.FinishTime, "2026-02-13T10:05:32Z")
	}
	if item.CreatedAt != "2026-02-13T10:00:00Z" {
		t.Errorf("CreatedAt = %q, want %q", item.CreatedAt, "2026-02-13T10:00:00Z")
	}
	if item.Age == "" {
		t.Error("Age should not be empty")
	}
}

func TestJobItem_NilTimePointers(t *testing.T) {
	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			Name:      "running-job",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Active:     1,
		StartTime:  nil,
		FinishTime: nil,
	}

	item := JobItem(src)

	if item.StartTime != "" {
		t.Errorf("StartTime = %q, want empty string for nil", item.StartTime)
	}
	if item.FinishTime != "" {
		t.Errorf("FinishTime = %q, want empty string for nil", item.FinishTime)
	}
}

func TestJobItems_NilInput(t *testing.T) {
	result := JobItems(nil)
	if result == nil {
		t.Error("JobItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("JobItems(nil) length = %d, want 0", len(result))
	}
}

func TestJobItems_EmptyInput(t *testing.T) {
	result := JobItems([]model_v2.Job{})
	if result == nil {
		t.Error("JobItems([]) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("JobItems([]) length = %d, want 0", len(result))
	}
}

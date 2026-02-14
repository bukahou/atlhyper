package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestCronJobItem_FieldMapping(t *testing.T) {
	lastSchedule := time.Date(2026, 2, 14, 2, 0, 0, 0, time.UTC)
	lastSuccess := time.Date(2026, 2, 14, 2, 5, 0, 0, time.UTC)
	src := &model_v2.CronJob{
		CommonMeta: model_v2.CommonMeta{
			Name:      "backup-daily",
			Namespace: "default",
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Schedule:           "0 2 * * *",
		Suspend:            false,
		ActiveJobs:         0,
		LastScheduleTime:   &lastSchedule,
		LastSuccessfulTime: &lastSuccess,
	}

	item := CronJobItem(src)

	if item.Name != "backup-daily" {
		t.Errorf("Name = %q, want %q", item.Name, "backup-daily")
	}
	if item.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "default")
	}
	if item.Schedule != "0 2 * * *" {
		t.Errorf("Schedule = %q, want %q", item.Schedule, "0 2 * * *")
	}
	if item.Suspend {
		t.Error("Suspend = true, want false")
	}
	if item.ActiveJobs != 0 {
		t.Errorf("ActiveJobs = %d, want 0", item.ActiveJobs)
	}
	if item.LastScheduleTime != "2026-02-14T02:00:00Z" {
		t.Errorf("LastScheduleTime = %q, want %q", item.LastScheduleTime, "2026-02-14T02:00:00Z")
	}
	if item.LastSuccessfulTime != "2026-02-14T02:05:00Z" {
		t.Errorf("LastSuccessfulTime = %q, want %q", item.LastSuccessfulTime, "2026-02-14T02:05:00Z")
	}
	if item.Age == "" {
		t.Error("Age should not be empty")
	}
}

func TestCronJobItem_NilTimePointers(t *testing.T) {
	src := &model_v2.CronJob{
		CommonMeta: model_v2.CommonMeta{
			Name:      "new-cronjob",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Schedule:           "*/5 * * * *",
		LastScheduleTime:   nil,
		LastSuccessfulTime: nil,
	}

	item := CronJobItem(src)

	if item.LastScheduleTime != "" {
		t.Errorf("LastScheduleTime = %q, want empty", item.LastScheduleTime)
	}
	if item.LastSuccessfulTime != "" {
		t.Errorf("LastSuccessfulTime = %q, want empty", item.LastSuccessfulTime)
	}
}

func TestCronJobItems_NilInput(t *testing.T) {
	result := CronJobItems(nil)
	if result == nil {
		t.Error("CronJobItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestCronJobItems_EmptyInput(t *testing.T) {
	result := CronJobItems([]model_v2.CronJob{})
	if result == nil {
		t.Error("CronJobItems([]) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

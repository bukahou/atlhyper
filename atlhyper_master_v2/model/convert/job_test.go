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

// ============================================================
// JobDetail 测试
// ============================================================

func TestJobDetail_FieldMapping(t *testing.T) {
	start := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)
	finish := time.Date(2026, 2, 13, 10, 5, 32, 0, time.UTC)
	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			UID:       "abc-123",
			Name:      "data-migration-v2",
			Namespace: "default",
			OwnerKind: "CronJob",
			OwnerName: "backup-daily",
			Labels:    map[string]string{"app": "backup"},
			CreatedAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
		},
		Active:     0,
		Succeeded:  1,
		Failed:     0,
		Complete:   true,
		StartTime:  &start,
		FinishTime: &finish,
	}

	detail := JobDetail(src)

	if detail.UID != "abc-123" {
		t.Errorf("UID = %q, want %q", detail.UID, "abc-123")
	}
	if detail.Name != "data-migration-v2" {
		t.Errorf("Name = %q, want %q", detail.Name, "data-migration-v2")
	}
	if detail.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", detail.Namespace, "default")
	}
	if detail.OwnerKind != "CronJob" {
		t.Errorf("OwnerKind = %q, want %q", detail.OwnerKind, "CronJob")
	}
	if detail.OwnerName != "backup-daily" {
		t.Errorf("OwnerName = %q, want %q", detail.OwnerName, "backup-daily")
	}
	if detail.Active != 0 {
		t.Errorf("Active = %d, want 0", detail.Active)
	}
	if detail.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", detail.Succeeded)
	}
	if detail.Labels["app"] != "backup" {
		t.Errorf("Labels[app] = %q, want %q", detail.Labels["app"], "backup")
	}
	if detail.StartTime != "2026-02-13T10:00:00Z" {
		t.Errorf("StartTime = %q, want %q", detail.StartTime, "2026-02-13T10:00:00Z")
	}
	if detail.FinishTime != "2026-02-13T10:05:32Z" {
		t.Errorf("FinishTime = %q, want %q", detail.FinishTime, "2026-02-13T10:05:32Z")
	}
}

func TestJobDetail_Duration(t *testing.T) {
	tests := []struct {
		name     string
		start    *time.Time
		finish   *time.Time
		expected string
	}{
		{
			name:     "5 minutes 32 seconds",
			start:    timePtr(time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)),
			finish:   timePtr(time.Date(2026, 2, 13, 10, 5, 32, 0, time.UTC)),
			expected: "5m32s",
		},
		{
			name:     "2 hours 30 minutes",
			start:    timePtr(time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)),
			finish:   timePtr(time.Date(2026, 2, 13, 12, 30, 0, 0, time.UTC)),
			expected: "2h30m",
		},
		{
			name:     "30 seconds",
			start:    timePtr(time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC)),
			finish:   timePtr(time.Date(2026, 2, 13, 10, 0, 30, 0, time.UTC)),
			expected: "30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &model_v2.Job{
				CommonMeta: model_v2.CommonMeta{
					Name:      "test-job",
					Namespace: "default",
					CreatedAt: time.Now(),
				},
				StartTime:  tt.start,
				FinishTime: tt.finish,
			}
			detail := JobDetail(src)
			if detail.Duration != tt.expected {
				t.Errorf("Duration = %q, want %q", detail.Duration, tt.expected)
			}
		})
	}
}

func TestJobDetail_NilTimes(t *testing.T) {
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

	detail := JobDetail(src)

	if detail.StartTime != "" {
		t.Errorf("StartTime = %q, want empty for nil", detail.StartTime)
	}
	if detail.FinishTime != "" {
		t.Errorf("FinishTime = %q, want empty for nil", detail.FinishTime)
	}
	if detail.Duration != "" {
		t.Errorf("Duration = %q, want empty for nil times", detail.Duration)
	}
}

func TestJobDetail_Status(t *testing.T) {
	tests := []struct {
		name     string
		active   int32
		failed   int32
		complete bool
		expected string
	}{
		{"complete", 0, 0, true, "Complete"},
		{"running", 1, 0, false, "Running"},
		{"failed", 0, 2, false, "Failed"},
		{"complete with failures", 0, 1, true, "Complete"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &model_v2.Job{
				CommonMeta: model_v2.CommonMeta{
					Name:      "test",
					Namespace: "default",
					CreatedAt: time.Now(),
				},
				Active:   tt.active,
				Failed:   tt.failed,
				Complete: tt.complete,
			}
			detail := JobDetail(src)
			if detail.Status != tt.expected {
				t.Errorf("Status = %q, want %q", detail.Status, tt.expected)
			}
		})
	}
}

func TestJobDetail_PodTemplate(t *testing.T) {
	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			Name:      "test-job",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Template: model_v2.PodTemplate{
			Containers: []model_v2.ContainerDetail{
				{Name: "worker", Image: "busybox:latest"},
			},
		},
	}

	detail := JobDetail(src)

	if detail.Template == nil {
		t.Error("Template should not be nil when containers exist")
	}
}

func TestJobDetail_PodTemplate_Empty(t *testing.T) {
	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			Name:      "test-job",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
	}

	detail := JobDetail(src)

	if detail.Template != nil {
		t.Error("Template should be nil when no containers")
	}
}

func TestJobDetail_Conditions(t *testing.T) {
	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			Name:      "test-job",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Complete: true,
		Conditions: []model_v2.WorkloadCondition{
			{Type: "Complete", Status: "True", Reason: "BackoffLimitExceeded"},
		},
	}

	detail := JobDetail(src)

	if detail.Conditions == nil {
		t.Error("Conditions should not be nil when conditions exist")
	}
}

func TestJobDetail_SpecFields(t *testing.T) {
	completions := int32(3)
	parallelism := int32(2)
	backoff := int32(6)

	src := &model_v2.Job{
		CommonMeta: model_v2.CommonMeta{
			Name:      "test-job",
			Namespace: "default",
			CreatedAt: time.Now(),
		},
		Completions:  &completions,
		Parallelism:  &parallelism,
		BackoffLimit: &backoff,
	}

	detail := JobDetail(src)

	if detail.Completions == nil || *detail.Completions != 3 {
		t.Errorf("Completions = %v, want 3", detail.Completions)
	}
	if detail.Parallelism == nil || *detail.Parallelism != 2 {
		t.Errorf("Parallelism = %v, want 2", detail.Parallelism)
	}
	if detail.BackoffLimit == nil || *detail.BackoffLimit != 6 {
		t.Errorf("BackoffLimit = %v, want 6", detail.BackoffLimit)
	}
}

// 辅助函数
func timePtr(t time.Time) *time.Time {
	return &t
}

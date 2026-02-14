package convert

import (
	"testing"

	"AtlHyper/model_v2"
)

func TestLimitRangeItem_FieldMapping(t *testing.T) {
	src := &model_v2.LimitRange{
		Name:      "default-limits",
		Namespace: "production",
		Items: []model_v2.LimitRangeItem{
			{
				Type:           "Container",
				Max:            map[string]string{"cpu": "4", "memory": "8Gi"},
				Min:            map[string]string{"cpu": "100m", "memory": "128Mi"},
				Default:        map[string]string{"cpu": "500m", "memory": "512Mi"},
				DefaultRequest: map[string]string{"cpu": "200m", "memory": "256Mi"},
			},
		},
		CreatedAt: "2025-12-01T00:00:00Z",
		Age:       "75d",
	}

	item := LimitRangeItem(src)

	if item.Name != "default-limits" {
		t.Errorf("Name = %q, want %q", item.Name, "default-limits")
	}
	if item.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "production")
	}
	if len(item.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(item.Items))
	}
	if item.Items[0].Type != "Container" {
		t.Errorf("Items[0].Type = %q, want %q", item.Items[0].Type, "Container")
	}
	if item.Items[0].Max["cpu"] != "4" {
		t.Errorf("Items[0].Max[cpu] = %q, want %q", item.Items[0].Max["cpu"], "4")
	}
	if item.Items[0].Default["memory"] != "512Mi" {
		t.Errorf("Items[0].Default[memory] = %q, want %q", item.Items[0].Default["memory"], "512Mi")
	}
	if item.Age != "75d" {
		t.Errorf("Age = %q, want %q", item.Age, "75d")
	}
}

func TestLimitRangeItems_NilInput(t *testing.T) {
	result := LimitRangeItems(nil)
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestLimitRangeItems_EmptyInput(t *testing.T) {
	result := LimitRangeItems([]model_v2.LimitRange{})
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestLimitRangeDetail_FieldMapping(t *testing.T) {
	src := &model_v2.LimitRange{
		Name:      "default-limits",
		Namespace: "production",
		Items: []model_v2.LimitRangeItem{
			{
				Type:    "Container",
				Max:     map[string]string{"cpu": "4"},
				Default: map[string]string{"cpu": "500m"},
			},
		},
		CreatedAt:   "2025-12-01T00:00:00Z",
		Age:         "75d",
		Labels:      map[string]string{"env": "prod"},
		Annotations: map[string]string{"managed": "terraform"},
	}

	detail := LimitRangeDetail(src)

	if detail.Name != "default-limits" {
		t.Errorf("Name = %q, want %q", detail.Name, "default-limits")
	}
	if len(detail.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(detail.Items))
	}
	if detail.Items[0].Type != "Container" {
		t.Errorf("Items[0].Type = %q, want %q", detail.Items[0].Type, "Container")
	}
	if detail.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %q, want %q", detail.Labels["env"], "prod")
	}
	if detail.Annotations["managed"] != "terraform" {
		t.Errorf("Annotations[managed] = %q, want %q", detail.Annotations["managed"], "terraform")
	}
}

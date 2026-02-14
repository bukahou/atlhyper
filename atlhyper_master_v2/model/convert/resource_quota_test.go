package convert

import (
	"testing"

	"AtlHyper/model_v2"
)

func TestResourceQuotaItem_FieldMapping(t *testing.T) {
	src := &model_v2.ResourceQuota{
		Name:      "compute-quota",
		Namespace: "production",
		Scopes:    []string{"NotTerminating"},
		Hard:      map[string]string{"requests.cpu": "10", "pods": "50"},
		Used:      map[string]string{"requests.cpu": "6.5", "pods": "32"},
		CreatedAt: "2025-12-01T00:00:00Z",
		Age:       "75d",
	}

	item := ResourceQuotaItem(src)

	if item.Name != "compute-quota" {
		t.Errorf("Name = %q, want %q", item.Name, "compute-quota")
	}
	if item.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "production")
	}
	if len(item.Scopes) != 1 || item.Scopes[0] != "NotTerminating" {
		t.Errorf("Scopes = %v, want [NotTerminating]", item.Scopes)
	}
	if item.Hard["requests.cpu"] != "10" {
		t.Errorf("Hard[requests.cpu] = %q, want %q", item.Hard["requests.cpu"], "10")
	}
	if item.Used["pods"] != "32" {
		t.Errorf("Used[pods] = %q, want %q", item.Used["pods"], "32")
	}
	if item.Age != "75d" {
		t.Errorf("Age = %q, want %q", item.Age, "75d")
	}
}

func TestResourceQuotaItems_NilInput(t *testing.T) {
	result := ResourceQuotaItems(nil)
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestResourceQuotaItems_EmptyInput(t *testing.T) {
	result := ResourceQuotaItems([]model_v2.ResourceQuota{})
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

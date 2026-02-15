package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestStatefulSetItem_FieldMapping(t *testing.T) {
	src := &model_v2.StatefulSet{
		Summary: model_v2.StatefulSetSummary{
			Name:        "mysql",
			Namespace:   "db",
			Replicas:    3,
			Ready:       3,
			Current:     3,
			Updated:     3,
			Available:   3,
			CreatedAt:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			Age:         "60d",
			ServiceName: "mysql-headless",
		},
	}

	item := StatefulSetItem(src)

	if item.Name != "mysql" {
		t.Errorf("Name = %q, want %q", item.Name, "mysql")
	}
	if item.Namespace != "db" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "db")
	}
	if item.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", item.Replicas)
	}
	if item.Ready != 3 {
		t.Errorf("Ready = %d, want 3", item.Ready)
	}
	if item.ServiceName != "mysql-headless" {
		t.Errorf("ServiceName = %q, want %q", item.ServiceName, "mysql-headless")
	}
	if item.CreatedAt != "2024-03-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want RFC3339", item.CreatedAt)
	}
	if item.Age != "60d" {
		t.Errorf("Age = %q, want %q", item.Age, "60d")
	}
}

func TestStatefulSetItems_NilReturnsEmpty(t *testing.T) {
	result := StatefulSetItems(nil)
	if result == nil {
		t.Error("StatefulSetItems(nil) should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestStatefulSetItems_Multiple(t *testing.T) {
	src := []model_v2.StatefulSet{
		{Summary: model_v2.StatefulSetSummary{Name: "a", CreatedAt: time.Now()}},
		{Summary: model_v2.StatefulSetSummary{Name: "b", CreatedAt: time.Now()}},
	}
	result := StatefulSetItems(src)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].Name != "a" || result[1].Name != "b" {
		t.Errorf("names = [%q, %q], want [a, b]", result[0].Name, result[1].Name)
	}
}

func TestStatefulSetDetail_FieldMapping(t *testing.T) {
	src := &model_v2.StatefulSet{
		Summary: model_v2.StatefulSetSummary{
			Name:        "mysql",
			Namespace:   "db",
			Replicas:    3,
			Ready:       3,
			Current:     3,
			Updated:     3,
			Available:   3,
			CreatedAt:   time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			Age:         "60d",
			ServiceName: "mysql-headless",
			Selector:    "app=mysql",
		},
		Labels:      map[string]string{"app": "mysql"},
		Annotations: map[string]string{"version": "8.0"},
	}

	d := StatefulSetDetail(src)

	if d.Name != "mysql" {
		t.Errorf("Name = %q, want %q", d.Name, "mysql")
	}
	if d.Replicas != 3 {
		t.Errorf("Replicas = %d, want 3", d.Replicas)
	}
	if d.ServiceName != "mysql-headless" {
		t.Errorf("ServiceName = %q, want %q", d.ServiceName, "mysql-headless")
	}
	if d.Age != "60d" {
		t.Errorf("Age = %q, want %q", d.Age, "60d")
	}
	if d.CreatedAt != "2024-03-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want RFC3339 format", d.CreatedAt)
	}
	if len(d.Labels) != 1 {
		t.Errorf("Labels count = %d, want 1", len(d.Labels))
	}
}

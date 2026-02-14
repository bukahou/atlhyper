package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

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

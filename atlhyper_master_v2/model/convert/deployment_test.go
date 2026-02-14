package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestDeploymentItem_FieldMapping(t *testing.T) {
	src := &model_v2.Deployment{
		Summary: model_v2.DeploymentSummary{
			Name:      "web",
			Namespace: "default",
			Replicas:  3,
			Ready:     2,
			CreatedAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		Template: model_v2.PodTemplate{
			Containers: []model_v2.ContainerDetail{
				{Image: "nginx:1.25"},
				{Image: "sidecar:latest"},
			},
		},
		Labels:      map[string]string{"app": "web", "env": "prod"},
		Annotations: map[string]string{"note": "test"},
	}

	item := DeploymentItem(src)

	if item.Name != "web" {
		t.Errorf("Name = %q, want %q", item.Name, "web")
	}
	if item.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "default")
	}
	if item.Image != "nginx:1.25" {
		t.Errorf("Image = %q, want first container image %q", item.Image, "nginx:1.25")
	}
	if item.Replicas != "2/3" {
		t.Errorf("Replicas = %q, want %q", item.Replicas, "2/3")
	}
	if item.LabelCount != 2 {
		t.Errorf("LabelCount = %d, want 2", item.LabelCount)
	}
	if item.AnnoCount != 1 {
		t.Errorf("AnnoCount = %d, want 1", item.AnnoCount)
	}
}

func TestDeploymentItem_NoContainers(t *testing.T) {
	src := &model_v2.Deployment{
		Summary: model_v2.DeploymentSummary{
			Name:      "empty",
			Namespace: "test",
		},
		Template: model_v2.PodTemplate{},
	}

	item := DeploymentItem(src)
	if item.Image != "" {
		t.Errorf("Image = %q, want empty for no containers", item.Image)
	}
}

func TestDeploymentDetail_FieldMapping(t *testing.T) {
	src := &model_v2.Deployment{
		Summary: model_v2.DeploymentSummary{
			Name:      "api",
			Namespace: "prod",
			Strategy:  "RollingUpdate",
			Replicas:  5,
			Updated:   5,
			Ready:     5,
			Available: 5,
			Paused:    true,
			Selector:  "app=api",
			CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Age:       "30d",
		},
		Labels: map[string]string{"app": "api"},
	}

	d := DeploymentDetail(src)

	if d.Strategy != "RollingUpdate" {
		t.Errorf("Strategy = %q, want %q", d.Strategy, "RollingUpdate")
	}
	if d.Replicas != 5 {
		t.Errorf("Replicas = %d, want 5", d.Replicas)
	}
	if !d.Paused {
		t.Error("Paused = false, want true")
	}
	if d.Age != "30d" {
		t.Errorf("Age = %q, want %q", d.Age, "30d")
	}
}

func TestDeploymentItems_NilInput(t *testing.T) {
	result := DeploymentItems(nil)
	if result == nil {
		t.Error("DeploymentItems(nil) should return non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

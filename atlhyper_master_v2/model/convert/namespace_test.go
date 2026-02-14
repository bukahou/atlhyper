package convert

import (
	"testing"

	"AtlHyper/model_v2"
)

func TestNamespaceItem_FieldMapping(t *testing.T) {
	src := &model_v2.Namespace{
		Summary: model_v2.NamespaceSummary{
			Name:      "production",
			CreatedAt: "2024-01-01T00:00:00Z",
		},
		Status: model_v2.NamespaceStatus{Phase: "Active"},
		Resources: model_v2.NamespaceResources{
			Pods: 42,
		},
		Labels:      map[string]string{"env": "prod", "team": "backend"},
		Annotations: map[string]string{"description": "prod namespace"},
	}

	item := NamespaceItem(src)

	if item.Name != "production" {
		t.Errorf("Name = %q, want %q", item.Name, "production")
	}
	if item.Status != "Active" {
		t.Errorf("Status = %q, want %q", item.Status, "Active")
	}
	if item.PodCount != 42 {
		t.Errorf("PodCount = %d, want 42", item.PodCount)
	}
	if item.LabelCount != 2 {
		t.Errorf("LabelCount = %d, want 2", item.LabelCount)
	}
	if item.AnnotationCount != 1 {
		t.Errorf("AnnotationCount = %d, want 1", item.AnnotationCount)
	}
}

func TestNamespaceDetail_FieldMapping(t *testing.T) {
	src := &model_v2.Namespace{
		Summary: model_v2.NamespaceSummary{
			Name:      "staging",
			CreatedAt: "2024-03-15T00:00:00Z",
			Age:       "120d",
		},
		Status: model_v2.NamespaceStatus{Phase: "Active"},
		Resources: model_v2.NamespaceResources{
			Pods:        20,
			PodsRunning: 18,
			PodsPending: 1,
			PodsFailed:  1,
			Deployments: 5,
			Services:    8,
			Ingresses:   3,
			ConfigMaps:  10,
			Secrets:     4,
			PVCs:        2,
		},
		Labels: map[string]string{"env": "staging"},
	}

	d := NamespaceDetail(src)

	if d.Phase != "Active" {
		t.Errorf("Phase = %q, want %q", d.Phase, "Active")
	}
	if d.Age != "120d" {
		t.Errorf("Age = %q, want %q", d.Age, "120d")
	}
	if d.Pods != 20 {
		t.Errorf("Pods = %d, want 20", d.Pods)
	}
	if d.PodsRunning != 18 {
		t.Errorf("PodsRunning = %d, want 18", d.PodsRunning)
	}
	if d.Deployments != 5 {
		t.Errorf("Deployments = %d, want 5", d.Deployments)
	}
	if d.PersistentVolumeClaims != 2 {
		t.Errorf("PVCs = %d, want 2", d.PersistentVolumeClaims)
	}
}

func TestNamespaceItems_NilInput(t *testing.T) {
	result := NamespaceItems(nil)
	if result == nil {
		t.Error("NamespaceItems(nil) should return non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

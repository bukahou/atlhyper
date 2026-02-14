package convert

import (
	"testing"

	"AtlHyper/model_v2"
)

func TestServiceAccountItem_FieldMapping(t *testing.T) {
	automount := true
	src := &model_v2.ServiceAccount{
		Name:                         "app-deployer",
		Namespace:                    "production",
		SecretsCount:                 2,
		ImagePullSecretsCount:        1,
		AutomountServiceAccountToken: &automount,
		CreatedAt:                    "2025-12-01T00:00:00Z",
		Age:                          "75d",
	}

	item := ServiceAccountItem(src)

	if item.Name != "app-deployer" {
		t.Errorf("Name = %q, want %q", item.Name, "app-deployer")
	}
	if item.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "production")
	}
	if item.SecretsCount != 2 {
		t.Errorf("SecretsCount = %d, want 2", item.SecretsCount)
	}
	if item.ImagePullSecretsCount != 1 {
		t.Errorf("ImagePullSecretsCount = %d, want 1", item.ImagePullSecretsCount)
	}
	if item.AutomountServiceAccountToken == nil || !*item.AutomountServiceAccountToken {
		t.Error("AutomountServiceAccountToken should be true")
	}
	if item.CreatedAt != "2025-12-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want %q", item.CreatedAt, "2025-12-01T00:00:00Z")
	}
	if item.Age != "75d" {
		t.Errorf("Age = %q, want %q", item.Age, "75d")
	}
}

func TestServiceAccountItem_NilAutomount(t *testing.T) {
	src := &model_v2.ServiceAccount{
		Name:                         "default",
		Namespace:                    "default",
		AutomountServiceAccountToken: nil,
		CreatedAt:                    "2025-06-01T00:00:00Z",
		Age:                          "258d",
	}

	item := ServiceAccountItem(src)

	if item.AutomountServiceAccountToken != nil {
		t.Error("AutomountServiceAccountToken should be nil")
	}
}

func TestServiceAccountItems_NilInput(t *testing.T) {
	result := ServiceAccountItems(nil)
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestServiceAccountItems_EmptyInput(t *testing.T) {
	result := ServiceAccountItems([]model_v2.ServiceAccount{})
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestServiceAccountDetail_SecretNames(t *testing.T) {
	automount := true
	src := &model_v2.ServiceAccount{
		Name:                         "app-deployer",
		Namespace:                    "production",
		SecretsCount:                 2,
		ImagePullSecretsCount:        1,
		AutomountServiceAccountToken: &automount,
		SecretNames:                  []string{"token-abc", "tls-cert"},
		ImagePullSecretNames:         []string{"registry-creds"},
		CreatedAt:                    "2025-12-01T00:00:00Z",
		Age:                          "75d",
		Labels:                       map[string]string{"team": "platform"},
		Annotations:                  map[string]string{"managed-by": "helm"},
	}

	detail := ServiceAccountDetail(src)

	if detail.Name != "app-deployer" {
		t.Errorf("Name = %q, want %q", detail.Name, "app-deployer")
	}
	if len(detail.SecretNames) != 2 {
		t.Errorf("SecretNames length = %d, want 2", len(detail.SecretNames))
	}
	if detail.SecretNames[0] != "token-abc" {
		t.Errorf("SecretNames[0] = %q, want %q", detail.SecretNames[0], "token-abc")
	}
	if len(detail.ImagePullSecretNames) != 1 || detail.ImagePullSecretNames[0] != "registry-creds" {
		t.Errorf("ImagePullSecretNames = %v, want [registry-creds]", detail.ImagePullSecretNames)
	}
	if detail.Labels["team"] != "platform" {
		t.Errorf("Labels[team] = %q, want %q", detail.Labels["team"], "platform")
	}
	if detail.Annotations["managed-by"] != "helm" {
		t.Errorf("Annotations[managed-by] = %q, want %q", detail.Annotations["managed-by"], "helm")
	}
}

package convert

import (
	"testing"

	"AtlHyper/model_v2"
)

func TestNetworkPolicyItem_FieldMapping(t *testing.T) {
	src := &model_v2.NetworkPolicy{
		Name:             "deny-all-ingress",
		Namespace:        "production",
		PodSelector:      "{}",
		PolicyTypes:      []string{"Ingress"},
		IngressRuleCount: 0,
		EgressRuleCount:  0,
		CreatedAt:        "2025-12-01T00:00:00Z",
		Age:              "75d",
	}

	item := NetworkPolicyItem(src)

	if item.Name != "deny-all-ingress" {
		t.Errorf("Name = %q, want %q", item.Name, "deny-all-ingress")
	}
	if item.Namespace != "production" {
		t.Errorf("Namespace = %q, want %q", item.Namespace, "production")
	}
	if item.PodSelector != "{}" {
		t.Errorf("PodSelector = %q, want %q", item.PodSelector, "{}")
	}
	if len(item.PolicyTypes) != 1 || item.PolicyTypes[0] != "Ingress" {
		t.Errorf("PolicyTypes = %v, want [Ingress]", item.PolicyTypes)
	}
	if item.IngressRuleCount != 0 {
		t.Errorf("IngressRuleCount = %d, want 0", item.IngressRuleCount)
	}
	if item.EgressRuleCount != 0 {
		t.Errorf("EgressRuleCount = %d, want 0", item.EgressRuleCount)
	}
	if item.CreatedAt != "2025-12-01T00:00:00Z" {
		t.Errorf("CreatedAt = %q, want %q", item.CreatedAt, "2025-12-01T00:00:00Z")
	}
	if item.Age != "75d" {
		t.Errorf("Age = %q, want %q", item.Age, "75d")
	}
}

func TestNetworkPolicyItems_NilInput(t *testing.T) {
	result := NetworkPolicyItems(nil)
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestNetworkPolicyItems_EmptyInput(t *testing.T) {
	result := NetworkPolicyItems([]model_v2.NetworkPolicy{})
	if result == nil {
		t.Error("should return empty slice, not nil")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

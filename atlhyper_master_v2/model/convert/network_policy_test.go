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

func TestNetworkPolicyDetail_Rules(t *testing.T) {
	src := &model_v2.NetworkPolicy{
		Name:             "allow-web",
		Namespace:        "production",
		PodSelector:      `{"matchLabels":{"app":"web"}}`,
		PolicyTypes:      []string{"Ingress", "Egress"},
		IngressRuleCount: 1,
		EgressRuleCount:  1,
		IngressRules: []model_v2.NetworkPolicyRule{
			{
				Peers: []model_v2.NetworkPolicyPeer{
					{Type: "podSelector", Selector: `{"matchLabels":{"role":"frontend"}}`},
				},
				Ports: []model_v2.NetworkPolicyPort{
					{Protocol: "TCP", Port: "80"},
				},
			},
		},
		EgressRules: []model_v2.NetworkPolicyRule{
			{
				Peers: []model_v2.NetworkPolicyPeer{
					{Type: "ipBlock", CIDR: "10.0.0.0/8", Except: []string{"10.0.1.0/24"}},
				},
			},
		},
		CreatedAt:   "2025-12-01T00:00:00Z",
		Age:         "75d",
		Labels:      map[string]string{"env": "prod"},
		Annotations: map[string]string{"note": "test"},
	}

	detail := NetworkPolicyDetail(src)

	if detail.Name != "allow-web" {
		t.Errorf("Name = %q, want %q", detail.Name, "allow-web")
	}
	if detail.IngressRules == nil {
		t.Error("IngressRules should not be nil")
	}
	if detail.EgressRules == nil {
		t.Error("EgressRules should not be nil")
	}
	if detail.Labels["env"] != "prod" {
		t.Errorf("Labels[env] = %q, want %q", detail.Labels["env"], "prod")
	}
	if detail.Annotations["note"] != "test" {
		t.Errorf("Annotations[note] = %q, want %q", detail.Annotations["note"], "test")
	}
}

func TestNetworkPolicyDetail_EmptyRules(t *testing.T) {
	src := &model_v2.NetworkPolicy{
		Name:      "deny-all",
		Namespace: "default",
		CreatedAt: "2025-12-01T00:00:00Z",
		Age:       "75d",
	}

	detail := NetworkPolicyDetail(src)

	if detail.IngressRules != nil {
		t.Error("IngressRules should be nil when no rules")
	}
	if detail.EgressRules != nil {
		t.Error("EgressRules should be nil when no rules")
	}
}

package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestServiceItem_PortFormatting(t *testing.T) {
	src := &model_v2.Service{
		Summary: model_v2.ServiceSummary{
			Name:      "web",
			Namespace: "default",
			Type:      "NodePort",
			ClusterIP: "10.96.0.1",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Ports: []model_v2.ServicePort{
			{Port: 80, NodePort: 30080, Protocol: "TCP", TargetPort: "8080"},
			{Port: 443, Protocol: "TCP", TargetPort: "8443"},
		},
		Selector: map[string]string{"app": "web"},
	}

	item := ServiceItem(src)

	if item.Ports != "80:30080/TCP→8080, 443/TCP→8443" {
		t.Errorf("Ports = %q, unexpected format", item.Ports)
	}
	if item.Protocol != "TCP" {
		t.Errorf("Protocol = %q, want %q", item.Protocol, "TCP")
	}
	if item.Type != "NodePort" {
		t.Errorf("Type = %q, want %q", item.Type, "NodePort")
	}
}

func TestServiceItem_SamePortTarget(t *testing.T) {
	src := &model_v2.Service{
		Summary: model_v2.ServiceSummary{
			Name:      "simple",
			Namespace: "default",
			Type:      "ClusterIP",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Ports: []model_v2.ServicePort{
			{Port: 80, Protocol: "TCP", TargetPort: "80"},
		},
	}

	item := ServiceItem(src)
	// Port == TargetPort → no arrow
	if item.Ports != "80/TCP" {
		t.Errorf("Ports = %q, want %q (no arrow when port==targetPort)", item.Ports, "80/TCP")
	}
}

func TestServiceItem_NoPorts(t *testing.T) {
	src := &model_v2.Service{
		Summary: model_v2.ServiceSummary{
			Name:      "headless",
			Namespace: "default",
			Type:      "ClusterIP",
			ClusterIP: "None",
			CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	item := ServiceItem(src)
	if item.Ports != "" {
		t.Errorf("Ports = %q, want empty", item.Ports)
	}
}

func TestServiceDetail_FieldMapping(t *testing.T) {
	src := &model_v2.Service{
		Summary: model_v2.ServiceSummary{
			Name:      "api",
			Namespace: "prod",
			Type:      "LoadBalancer",
			CreatedAt: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Age:       "30d",
			Badges:    []string{"external"},
		},
		Ports: []model_v2.ServicePort{
			{Port: 443, Protocol: "TCP", TargetPort: "8443", AppProtocol: "https"},
		},
		Selector: map[string]string{"app": "api"},
		Network: model_v2.ServiceNetwork{
			ClusterIPs:            []string{"10.96.0.5"},
			ExternalTrafficPolicy: "Local",
		},
		Spec: model_v2.ServiceSpec{
			SessionAffinity: "ClientIP",
		},
		Backends: &model_v2.ServiceBackends{
			Summary: model_v2.BackendSummary{Ready: 3, Total: 3},
		},
	}

	d := ServiceDetail(src)

	if d.Type != "LoadBalancer" {
		t.Errorf("Type = %q, want %q", d.Type, "LoadBalancer")
	}
	if len(d.Ports) != 1 || d.Ports[0].AppProtocol != "https" {
		t.Errorf("Ports not mapped correctly")
	}
	if d.SessionAffinity != "ClientIP" {
		t.Errorf("SessionAffinity = %q, want %q", d.SessionAffinity, "ClientIP")
	}
	if d.ExternalTrafficPolicy != "Local" {
		t.Errorf("ExternalTrafficPolicy = %q, want %q", d.ExternalTrafficPolicy, "Local")
	}
	if d.Backends == nil {
		t.Error("Backends should not be nil")
	}
}

func TestServiceItems_NilInput(t *testing.T) {
	result := ServiceItems(nil)
	if result == nil {
		t.Error("ServiceItems(nil) should return non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

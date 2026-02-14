package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v2"
)

func TestIngressItems_RowExpansion(t *testing.T) {
	src := []model_v2.Ingress{
		{
			Summary: model_v2.IngressSummary{
				Name:      "web-ingress",
				Namespace: "default",
				CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			Spec: model_v2.IngressSpec{
				Rules: []model_v2.IngressRule{
					{
						Host: "example.com",
						Paths: []model_v2.IngressPath{
							{
								Path:     "/api",
								PathType: "Prefix",
								Backend: &model_v2.IngressBackend{
									Service: &model_v2.IngressServiceBackend{
										Name:       "api-svc",
										PortNumber: 80,
									},
								},
							},
							{
								Path:     "/web",
								PathType: "Prefix",
								Backend: &model_v2.IngressBackend{
									Service: &model_v2.IngressServiceBackend{
										Name:     "web-svc",
										PortName: "http",
									},
								},
							},
						},
					},
					{
						Host: "admin.example.com",
						Paths: []model_v2.IngressPath{
							{
								Path:     "/",
								PathType: "Prefix",
								Backend: &model_v2.IngressBackend{
									Service: &model_v2.IngressServiceBackend{
										Name:       "admin-svc",
										PortNumber: 8080,
									},
								},
							},
						},
					},
				},
				TLS: []model_v2.IngressTLS{
					{Hosts: []string{"example.com"}},
				},
			},
		},
	}

	rows := IngressItems(src)

	// 1 Ingress × (2 paths + 1 path) = 3 rows
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(rows))
	}

	// Row 0: example.com /api → api-svc:80, TLS=true
	if rows[0].Host != "example.com" {
		t.Errorf("rows[0].Host = %q, want %q", rows[0].Host, "example.com")
	}
	if rows[0].Path != "/api" {
		t.Errorf("rows[0].Path = %q, want %q", rows[0].Path, "/api")
	}
	if rows[0].ServiceName != "api-svc" {
		t.Errorf("rows[0].ServiceName = %q, want %q", rows[0].ServiceName, "api-svc")
	}
	if rows[0].ServicePort != "80" {
		t.Errorf("rows[0].ServicePort = %q, want %q", rows[0].ServicePort, "80")
	}
	if !rows[0].TLS {
		t.Error("rows[0].TLS = false, want true (example.com has TLS)")
	}

	// Row 1: example.com /web → web-svc:http
	if rows[1].ServicePort != "http" {
		t.Errorf("rows[1].ServicePort = %q, want %q (port name)", rows[1].ServicePort, "http")
	}

	// Row 2: admin.example.com / → admin-svc:8080, TLS=false
	if rows[2].Host != "admin.example.com" {
		t.Errorf("rows[2].Host = %q, want %q", rows[2].Host, "admin.example.com")
	}
	if rows[2].TLS {
		t.Error("rows[2].TLS = true, want false (admin.example.com has no TLS)")
	}
}

func TestIngressItems_NoRules(t *testing.T) {
	src := []model_v2.Ingress{
		{
			Summary: model_v2.IngressSummary{
				Name:       "fallback",
				Namespace:  "default",
				CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				TLSEnabled: true,
			},
			Spec: model_v2.IngressSpec{
				DefaultBackend: &model_v2.IngressBackend{
					Service: &model_v2.IngressServiceBackend{
						Name:       "default-svc",
						PortNumber: 80,
					},
				},
			},
		},
	}

	rows := IngressItems(src)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].Host != "*" {
		t.Errorf("Host = %q, want %q", rows[0].Host, "*")
	}
	if rows[0].ServiceName != "default-svc" {
		t.Errorf("ServiceName = %q, want %q", rows[0].ServiceName, "default-svc")
	}
}

func TestIngressItems_NilInput(t *testing.T) {
	result := IngressItems(nil)
	if result == nil {
		t.Error("IngressItems(nil) should return non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("len = %d, want 0", len(result))
	}
}

func TestIngressDetail_FieldMapping(t *testing.T) {
	src := &model_v2.Ingress{
		Summary: model_v2.IngressSummary{
			Name:         "main",
			Namespace:    "prod",
			IngressClass: "traefik",
			CreatedAt:    time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			Age:          "30d",
			Hosts:        []string{"example.com"},
			TLSEnabled:   true,
		},
		Status: model_v2.IngressStatus{
			LoadBalancer: []string{"1.2.3.4"},
		},
		Annotations: map[string]string{"traefik.ingress.kubernetes.io/router.tls": "true"},
	}

	d := IngressDetail(src)

	if d.Class != "traefik" {
		t.Errorf("Class = %q, want %q", d.Class, "traefik")
	}
	if !d.TLSEnabled {
		t.Error("TLSEnabled = false, want true")
	}
	if len(d.LoadBalancer) != 1 || d.LoadBalancer[0] != "1.2.3.4" {
		t.Errorf("LoadBalancer = %v, want [1.2.3.4]", d.LoadBalancer)
	}
	if len(d.Annotations) != 1 {
		t.Errorf("Annotations count = %d, want 1", len(d.Annotations))
	}
}

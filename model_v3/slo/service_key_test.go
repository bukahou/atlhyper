package slo

import "testing"

func TestServiceKey(t *testing.T) {
	cases := []struct{ ns, svc, key string }{
		{"geass", "geass-web", "geass/geass-web"},
		{"", "orphan", "orphan"},
		{"ns", "", ""},
	}
	for _, c := range cases {
		if got := ServiceKey(c.ns, c.svc); got != c.key {
			t.Errorf("ServiceKey(%q, %q) = %q, want %q", c.ns, c.svc, got, c.key)
		}
		if c.key == "" {
			continue
		}
		if ns, svc := SplitServiceKey(c.key); ns != c.ns || svc != c.svc {
			t.Errorf("SplitServiceKey(%q) = %q, %q", c.key, ns, svc)
		}
	}
}

func TestIngressSLO_MatchesHost(t *testing.T) {
	ing := &IngressSLO{ServiceKey: "shop/api", DisplayName: "api.example.com"}
	for host, want := range map[string]bool{"api.example.com": true, "shop/api": true, "other.example.com": false, "": false} {
		if got := ing.MatchesHost(host); got != want {
			t.Errorf("MatchesHost(%q) = %v, want %v", host, got, want)
		}
	}
}

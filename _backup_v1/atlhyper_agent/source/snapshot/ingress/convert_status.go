// internal/readonly/ingress/convert_status.go
package ingress

import (
	networkingv1 "k8s.io/api/networking/v1"
)

func summarizeLB(items []networkingv1.IngressLoadBalancerIngress) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, in := range items {
		if in.IP != "" {
			out = append(out, in.IP)
			continue
		}
		if in.Hostname != "" {
			out = append(out, in.Hostname)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

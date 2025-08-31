package pod

import (
	modelpod "AtlHyper/model/pod"

	corev1 "k8s.io/api/core/v1"
)

func buildNetwork(p *corev1.Pod) modelpod.Network {
	return modelpod.Network{
		PodIP:              p.Status.PodIP,
		PodIPs:             extractPodIPs(p.Status.PodIPs),
		HostIP:             p.Status.HostIP,
		HostNetwork:        p.Spec.HostNetwork,
		DNSPolicy:          string(p.Spec.DNSPolicy),
		ServiceAccountName: p.Spec.ServiceAccountName,
	}
}

func extractPodIPs(ips []corev1.PodIP) []string {
	if len(ips) == 0 {
		return nil
	}
	out := make([]string, 0, len(ips))
	for _, ip := range ips {
		out = append(out, ip.IP)
	}
	return out
}

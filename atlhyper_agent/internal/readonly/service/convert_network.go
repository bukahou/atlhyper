package service

import (
	modelsvc "AtlHyper/model/service"

	corev1 "k8s.io/api/core/v1"
)

func buildNetwork(svc *corev1.Service) modelsvc.ServiceNetwork {
	return modelsvc.ServiceNetwork{
		ClusterIPs:            append([]string(nil), svc.Spec.ClusterIPs...),
		ExternalIPs:           append([]string(nil), svc.Spec.ExternalIPs...),
		LoadBalancerIngress:   extractLBIngress(svc),
		IPFamilies:            toStrSlice(svc.Spec.IPFamilies),
		IPFamilyPolicy:        ipFamilyPolicyPtrValue(svc.Spec.IPFamilyPolicy),      
		ExternalTrafficPolicy: string(svc.Spec.ExternalTrafficPolicy),
		InternalTrafficPolicy: internalTrafficPolicyPtrValue(svc.Spec.InternalTrafficPolicy), 
	}
}


func extractLBIngress(svc *corev1.Service) []string {
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer || svc.Status.LoadBalancer.Ingress == nil {
		return nil
	}
	out := make([]string, 0, len(svc.Status.LoadBalancer.Ingress))
	for _, in := range svc.Status.LoadBalancer.Ingress {
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

func isHeadless(svc *corev1.Service) bool {
	// Headless 的关键信号：spec.ClusterIP == "None"（或 ClusterIPs[0] == "None"）
	if svc.Spec.ClusterIP == "None" {
		return true
	}
	if len(svc.Spec.ClusterIPs) > 0 && svc.Spec.ClusterIPs[0] == "None" {
		return true
	}
	return false
}

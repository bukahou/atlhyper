package service

import (
	modelsvc "AtlHyper/model/k8s"

	corev1 "k8s.io/api/core/v1"
)

// buildSkeleton —— 基于 corev1.Service 构建“骨架”（不含 backends）
func buildSkeleton(svc *corev1.Service) modelsvc.Service {
	created := svc.CreationTimestamp.Time

	s := modelsvc.Service{
		Summary: modelsvc.ServiceSummary{
			Name:        svc.Name,
			Namespace:   svc.Namespace,
			Type:        string(svc.Spec.Type),
			CreatedAt:   created,
			Age:         fmtAge(created),
			PortsCount:  len(svc.Spec.Ports),
			HasSelector: len(svc.Spec.Selector) > 0,
			Badges:      deriveBadges(svc),
			ClusterIP:   firstClusterIPForSummary(svc),
		},
		Spec:     buildSpec(svc),
		Ports:    mapServicePorts(svc.Spec.Ports),
		Selector: svc.Spec.Selector,
		Network:  buildNetwork(svc),
	}

	// ExternalName 便捷填充（summary 与 spec 已覆盖；再保障一下）
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		s.Summary.ExternalName = svc.Spec.ExternalName
	}

	return s
}

// attachBackends —— 在骨架上填充 backends
func attachBackends(dst *modelsvc.Service, be modelsvc.ServiceBackends) {
	dst.Backends = &be
}

func buildSpec(svc *corev1.Service) modelsvc.ServiceSpec {
	var saTimeout *int32
	if svc.Spec.SessionAffinityConfig != nil && svc.Spec.SessionAffinityConfig.ClientIP != nil {
		saTimeout = svc.Spec.SessionAffinityConfig.ClientIP.TimeoutSeconds
	}

	return modelsvc.ServiceSpec{
		Type:                          string(svc.Spec.Type),
		SessionAffinity:               string(svc.Spec.SessionAffinity),
		SessionAffinityTimeoutSeconds: saTimeout,
		ExternalTrafficPolicy:         string(svc.Spec.ExternalTrafficPolicy),
		InternalTrafficPolicy:         internalTrafficPolicyPtrValue(svc.Spec.InternalTrafficPolicy), 
		IPFamilies:                    toStrSlice(svc.Spec.IPFamilies),
		IPFamilyPolicy:                ipFamilyPolicyPtrValue(svc.Spec.IPFamilyPolicy),
		ClusterIPs:                    append([]string(nil), svc.Spec.ClusterIPs...),
		ExternalIPs:                   append([]string(nil), svc.Spec.ExternalIPs...),
		LoadBalancerClass:             stringPtrValue(svc.Spec.LoadBalancerClass),
		LoadBalancerSourceRanges:      append([]string(nil), svc.Spec.LoadBalancerSourceRanges...),
		PublishNotReadyAddresses:      svc.Spec.PublishNotReadyAddresses,
		AllocateLoadBalancerNodePorts: svc.Spec.AllocateLoadBalancerNodePorts,
		HealthCheckNodePort:           svc.Spec.HealthCheckNodePort,
		ExternalName:                  svc.Spec.ExternalName,
	}
}


func deriveBadges(svc *corev1.Service) []string {
	var out []string
	if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
		out = append(out, "LB")
	}
	if isHeadless(svc) {
		out = append(out, "Headless")
	}
	if len(svc.Spec.Selector) == 0 && svc.Spec.Type != corev1.ServiceTypeExternalName {
		out = append(out, "NoSelector")
	}
	if svc.Spec.Type == corev1.ServiceTypeExternalName {
		out = append(out, "ExternalName")
	}
	return out
}

// firstClusterIPForSummary —— summary.clusterIP 的便捷值：优先 spec.ClusterIP，其次 spec.ClusterIPs[0]；Headless 则为 "None"
func firstClusterIPForSummary(svc *corev1.Service) string {
	if isHeadless(svc) {
		return "None"
	}
	if svc.Spec.ClusterIP != "" {
		return svc.Spec.ClusterIP
	}
	if len(svc.Spec.ClusterIPs) > 0 {
		return svc.Spec.ClusterIPs[0]
	}
	return ""
}

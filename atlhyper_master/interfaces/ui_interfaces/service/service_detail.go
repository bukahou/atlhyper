// atlhyper_master/interfaces/ui_interfaces/service/service_detail.go
package service

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/service"
)

// GetServiceDetail —— 根据 clusterID + namespace + serviceName 返回扁平详情
func GetServiceDetail(ctx context.Context, clusterID, namespace, name string) (*ServiceDetailDTO, error) {
	list, err := datasource.GetServiceListLatest(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get service list failed: %w", err)
	}
	for _, s := range list {
		if s.Summary.Namespace == namespace && s.Summary.Name == name {
			dto := fromModelToDetail(s)
			return &dto, nil
		}
	}
	return nil, fmt.Errorf("service not found: %s/%s (cluster=%s)", namespace, name, clusterID)
}

func fromModelToDetail(s mod.Service) ServiceDetailDTO {
	dto := ServiceDetailDTO{
		// 基本
		Name:      s.Summary.Name,
		Namespace: s.Summary.Namespace,
		Type:      pickType(s),
		CreatedAt: s.Summary.CreatedAt,
		Age:       s.Summary.Age,

		// 选择器 & 端口
		Selector: s.Selector,
		Ports:    make([]ServicePortDTO, 0, len(s.Ports)),

		// 网络
		ClusterIPs:          s.Network.ClusterIPs,
		ExternalIPs:         s.Network.ExternalIPs,
		LoadBalancerIngress: s.Network.LoadBalancerIngress,

		// 重要 spec
		SessionAffinity:               s.Spec.SessionAffinity,
		SessionAffinityTimeoutSeconds: s.Spec.SessionAffinityTimeoutSeconds,
		ExternalTrafficPolicy:         s.Spec.ExternalTrafficPolicy,
		InternalTrafficPolicy:         s.Spec.InternalTrafficPolicy,
		IPFamilies:                    s.Spec.IPFamilies,
		IPFamilyPolicy:                s.Spec.IPFamilyPolicy,
		LoadBalancerClass:             s.Spec.LoadBalancerClass,
		LoadBalancerSourceRanges:      s.Spec.LoadBalancerSourceRanges,
		AllocateLoadBalancerNodePorts: s.Spec.AllocateLoadBalancerNodePorts,
		HealthCheckNodePort:           s.Spec.HealthCheckNodePort,
		ExternalName:                  s.Spec.ExternalName,

		// 徽标
		Badges: inferBadges(s),
	}

	// 端口
	for _, p := range s.Ports {
		dto.Ports = append(dto.Ports, ServicePortDTO{
			Name:        p.Name,
			Protocol:    p.Protocol,
			Port:        p.Port,
			TargetPort:  p.TargetPort,
			NodePort:    p.NodePort,
			AppProtocol: p.AppProtocol,
		})
	}

	// 端点
	if s.Backends != nil {
		b := &BackendsDTO{
			Ready:   s.Backends.Summary.Ready,
			NotReady:s.Backends.Summary.NotReady,
			Total:   s.Backends.Summary.Total,
			Slices:  s.Backends.Summary.Slices,
			Updated: s.Backends.Summary.Updated,
		}
		for _, ep := range s.Backends.Ports {
			b.Ports = append(b.Ports, EndpointPortDTO{
				Name: ep.Name, Port: ep.Port, Protocol: ep.Protocol, AppProtocol: ep.AppProtocol,
			})
		}
		for _, e := range s.Backends.Endpoints {
			var ref *K8sRefDTO
			if e.TargetRef != nil {
				ref = &K8sRefDTO{
					Kind: e.TargetRef.Kind, Namespace: e.TargetRef.Namespace,
					Name: e.TargetRef.Name, UID: e.TargetRef.UID,
				}
			}
			b.Endpoints = append(b.Endpoints, BackendEndpointDTO{
				Address: e.Address, Ready: e.Ready, NodeName: e.NodeName, Zone: e.Zone, TargetRef: ref,
			})
		}
		dto.Backends = b
	}

	return dto
}

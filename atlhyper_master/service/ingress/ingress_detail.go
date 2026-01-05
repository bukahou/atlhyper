// atlhyper_master/service/ingress/ingress_detail.go
package ingress

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/model/ui"
	"AtlHyper/atlhyper_master/repository"
	mod "AtlHyper/model/k8s"
)

// BuildIngressDetail —— 根据 clusterID + namespace + name 返回单个 Ingress 详情
func BuildIngressDetail(ctx context.Context, clusterID, namespace, name string) (*ui.IngressDetailDTO, error) {
	list, err := repository.GetIngressListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	for _, in := range list {
		if in.Summary.Namespace == namespace && in.Summary.Name == name {
			dto := fromModelToDetail(in)
			return &dto, nil
		}
	}
	return nil, fmt.Errorf("ingress not found: %s/%s (cluster=%s)", namespace, name, clusterID)
}

func fromModelToDetail(in mod.Ingress) ui.IngressDetailDTO {
	dto := ui.IngressDetailDTO{
		Name:         in.Summary.Name,
		Namespace:    in.Summary.Namespace,
		Class:        in.Summary.Class,
		Controller:   in.Summary.Controller,
		Hosts:        in.Summary.Hosts,
		TLSEnabled:   in.Summary.TLSEnabled,
		LoadBalancer: in.Summary.LoadBalancer,
		CreatedAt:    in.Summary.CreatedAt,
		Age:          in.Summary.Age,
		Annotations:  in.Annotations,
		Status: ui.IngressStatusDTO{
			LoadBalancer: in.Status.LoadBalancer,
		},
	}

	// Spec
	spec := ui.IngressSpecDTO{
		IngressClassName:         in.Spec.IngressClassName,
		LoadBalancerSourceRanges: in.Spec.LoadBalancerSourceRanges,
	}
	// default backend
	if in.Spec.DefaultBackend != nil {
		spec.DefaultBackend = &ui.IngressBackendRef{
			Type: in.Spec.DefaultBackend.Type,
		}
		if in.Spec.DefaultBackend.Service != nil {
			spec.DefaultBackend.Service = &ui.IngressServiceBackend{
				Name:       in.Spec.DefaultBackend.Service.Name,
				PortName:   in.Spec.DefaultBackend.Service.PortName,
				PortNumber: in.Spec.DefaultBackend.Service.PortNumber,
			}
		}
		if in.Spec.DefaultBackend.Resource != nil {
			spec.DefaultBackend.Resource = &ui.IngressObjectRef{
				APIGroup:  in.Spec.DefaultBackend.Resource.APIGroup,
				Kind:      in.Spec.DefaultBackend.Resource.Kind,
				Name:      in.Spec.DefaultBackend.Resource.Name,
				Namespace: in.Spec.DefaultBackend.Resource.Namespace,
			}
		}
	}
	// rules
	if len(in.Spec.Rules) > 0 {
		spec.Rules = make([]ui.IngressRuleDTO, 0, len(in.Spec.Rules))
		for _, r := range in.Spec.Rules {
			paths := make([]ui.IngressHTTPPath, 0, len(r.Paths))
			for _, p := range r.Paths {
				path := ui.IngressHTTPPath{
					Path:     p.Path,
					PathType: p.PathType,
					Backend: ui.IngressBackendRef{
						Type: p.Backend.Type,
					},
				}
				if p.Backend.Service != nil {
					path.Backend.Service = &ui.IngressServiceBackend{
						Name:       p.Backend.Service.Name,
						PortName:   p.Backend.Service.PortName,
						PortNumber: p.Backend.Service.PortNumber,
					}
				}
				if p.Backend.Resource != nil {
					path.Backend.Resource = &ui.IngressObjectRef{
						APIGroup:  p.Backend.Resource.APIGroup,
						Kind:      p.Backend.Resource.Kind,
						Name:      p.Backend.Resource.Name,
						Namespace: p.Backend.Resource.Namespace,
					}
				}
				paths = append(paths, path)
			}
			spec.Rules = append(spec.Rules, ui.IngressRuleDTO{
				Host:  r.Host,
				Paths: paths,
			})
		}
	}
	// tls
	if len(in.Spec.TLS) > 0 {
		spec.TLS = make([]ui.IngressTLSDTO, 0, len(in.Spec.TLS))
		for _, t := range in.Spec.TLS {
			spec.TLS = append(spec.TLS, ui.IngressTLSDTO{
				SecretName: t.SecretName,
				Hosts:      t.Hosts,
			})
		}
	}

	dto.Spec = spec
	return dto
}

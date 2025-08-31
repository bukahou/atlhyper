// internal/readonly/ingress/convert_backends.go
package ingress

import (
	modelingr "AtlHyper/model/ingress"

	networkingv1 "k8s.io/api/networking/v1"
)

// 规则
func mapRules(rules []networkingv1.IngressRule, ns string) []modelingr.Rule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]modelingr.Rule, 0, len(rules))
	for _, r := range rules {
		var paths []modelingr.HTTPPath
		if r.HTTP != nil {
			for _, p := range r.HTTP.Paths {
				paths = append(paths, modelingr.HTTPPath{
					Path:     p.Path,
					PathType: pathTypeToStr(p.PathType),
					Backend:  mapBackend(&p.Backend, ns), // ✅ 返回值类型，直接赋值
				})
			}
		}
		out = append(out, modelingr.Rule{
			Host:  r.Host,
			Paths: paths,
		})
	}
	return out
}

func pathTypeToStr(pt *networkingv1.PathType) string {
	if pt == nil {
		return ""
	}
	return string(*pt)
}

// 后端（值类型返回）
func mapBackend(b *networkingv1.IngressBackend, ns string) modelingr.BackendRef {
	var ref modelingr.BackendRef // 零值：Type="", Service/Resource 为 nil
	if b == nil {
		return ref
	}

	// Service backend
	if b.Service != nil {
		ref.Type = "Service"
		ref.Service = &modelingr.ServiceBackend{
			Name: b.Service.Name,
		}
		// 端口：Name 优先，否则 Number
		if b.Service.Port.Name != "" {
			ref.Service.PortName = b.Service.Port.Name
		} else if b.Service.Port.Number != 0 {
			ref.Service.PortNumber = b.Service.Port.Number
		}
		return ref
	}

	// Resource backend（TypedLocalObjectReference，通常与 Ingress 同命名空间）
	if b.Resource != nil {
		ref.Type = "Resource"
		apiGroup := ""
		if b.Resource.APIGroup != nil {
			apiGroup = *b.Resource.APIGroup
		}
		ref.Resource = &modelingr.ObjectRef{
			APIGroup:  apiGroup,
			Kind:      b.Resource.Kind,
			Name:      b.Resource.Name,
			Namespace: ns,
		}
		return ref
	}

	// 其它情况保持零值
	return ref
}

// TLS
func mapTLS(items []networkingv1.IngressTLS) []modelingr.IngressTLS {
	if len(items) == 0 {
		return nil
	}
	out := make([]modelingr.IngressTLS, 0, len(items))
	for _, t := range items {
		out = append(out, modelingr.IngressTLS{
			SecretName: t.SecretName,
			Hosts:      append([]string(nil), t.Hosts...),
		})
	}
	return out
}

// 从规则提取 hosts（去重）
func hostsFromRules(rules []networkingv1.IngressRule) []string {
	if len(rules) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if r.Host == "" {
			continue
		}
		if _, ok := seen[r.Host]; ok {
			continue
		}
		seen[r.Host] = struct{}{}
		out = append(out, r.Host)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}



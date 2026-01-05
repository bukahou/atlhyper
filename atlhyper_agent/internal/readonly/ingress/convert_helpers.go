// internal/readonly/ingress/convert_helpers.go
package ingress

import (
	"strings"

	modelingr "AtlHyper/model/k8s"

	networkingv1 "k8s.io/api/networking/v1"
)

// classNameOf —— 优先 spec.ingressClassName，回退注解 kubernetes.io/ingress.class
func classNameOf(in *networkingv1.Ingress) string {
	if in.Spec.IngressClassName != nil && *in.Spec.IngressClassName != "" {
		return *in.Spec.IngressClassName
	}
	if v := in.Annotations["kubernetes.io/ingress.class"]; v != "" {
		return v
	}
	return ""
}

func pickAnnotations(ann map[string]string) map[string]string {
	if len(ann) == 0 {
		return nil
	}
	keys := []string{
		"kubernetes.io/ingress.class",
		"nginx.ingress.kubernetes.io/rewrite-target",
		"nginx.ingress.kubernetes.io/ssl-redirect",
		"nginx.ingress.kubernetes.io/force-ssl-redirect",
		"alb.ingress.kubernetes.io/scheme",
		"alb.ingress.kubernetes.io/listen-ports",
		"traefik.ingress.kubernetes.io/router.entrypoints",
	}
	out := map[string]string{}
	for _, k := range keys {
		if v, ok := ann[k]; ok && v != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// 从常见控制器注解中提取来源网段白名单（逗号分隔）
func lbSrcRangesFromAnnotations(ann map[string]string) []string {
	if len(ann) == 0 {
		return nil
	}
	candidates := []string{
		// nginx
		"nginx.ingress.kubernetes.io/whitelist-source-range",
		// ALB
		"alb.ingress.kubernetes.io/inbound-cidrs",
		// 其它控制器可继续补充
	}
	for _, k := range candidates {
		if v, ok := ann[k]; ok && strings.TrimSpace(v) != "" {
			parts := strings.Split(v, ",")
			out := make([]string, 0, len(parts))
			for _, p := range parts {
				if s := strings.TrimSpace(p); s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

// 把值类型 BackendRef 转成 *BackendRef；若为“空 backend”则返回 nil
func backendPtr(b modelingr.BackendRef) *modelingr.BackendRef {
	if b.Type == "" && b.Service == nil && b.Resource == nil {
		return nil
	}
	return &b
}

// internal/readonly/ingress/convert_model.go
package ingress

import (
	"fmt"
	"time"

	modelingr "AtlHyper/model/k8s"

	networkingv1 "k8s.io/api/networking/v1"
)

func buildModel(in *networkingv1.Ingress, classCtl map[string]string) modelingr.Ingress {
	hosts := hostsFromRules(in.Spec.Rules)
	lb := summarizeLB(in.Status.LoadBalancer.Ingress)

	cls := classNameOf(in)
	controller := ""
	if cls != "" {
		controller = classCtl[cls] // 可能拿不到就留空
	}

	created := in.CreationTimestamp.Time

	// 从常见注解里提取“来源网段”白名单（不同控制器有不同键）
	lbSrc := lbSrcRangesFromAnnotations(in.Annotations)

	// DefaultBackend：mapBackend 返回值类型，这里转为 *BackendRef（空则返回 nil）
	defb := backendPtr(mapBackend(in.Spec.DefaultBackend, in.Namespace))

	return modelingr.Ingress{
		Summary: modelingr.IngressSummary{
			Name:         in.Name,
			Namespace:    in.Namespace,
			Class:        cls,
			Controller:   controller,
			Hosts:        hosts,
			TLSEnabled:   len(in.Spec.TLS) > 0,
			CreatedAt:    created,
			Age:          fmtAge(created),
			LoadBalancer: lb,
		},
		Spec: modelingr.IngressSpec{
			IngressClassName:         cls,
			LoadBalancerSourceRanges: lbSrc,                // ✅ 来自注解，而不是 in.Spec
			DefaultBackend:           defb,                 // ✅ 值→指针
			Rules:                    mapRules(in.Spec.Rules, in.Namespace),
			TLS:                      mapTLS(in.Spec.TLS),
		},
		Status: modelingr.IngressStatus{
			LoadBalancer: lb,
		},
		Annotations: pickAnnotations(in.Annotations),
	}
}

// fmtAge —— 与其他资源统一的“简洁时长”
func fmtAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	day := d / (24 * time.Hour)
	d -= day * 24 * time.Hour
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	switch {
	case day > 0:
		return fmt.Sprintf("%dd%dh", day, h)
	case h > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

// atlhyper_master_v2/model/convert/ingress.go
// model_v2.Ingress → model.IngressItem / model.IngressDetail 转换函数
// IngressItems 做行展开：1 个 Ingress 含 N 个 host×path → N 行 IngressItem
package convert

import (
	"fmt"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// IngressItems 展开多个 Ingress 为行列表（1 Ingress → N 行）
func IngressItems(src []model_v2.Ingress) []model.IngressItem {
	if src == nil {
		return []model.IngressItem{}
	}
	var rows []model.IngressItem
	for i := range src {
		rows = append(rows, expandIngress(&src[i])...)
	}
	if rows == nil {
		return []model.IngressItem{}
	}
	return rows
}

// expandIngress 将单个 Ingress 展开为多行
func expandIngress(src *model_v2.Ingress) []model.IngressItem {
	createdAt := src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00")

	// 检查每个 host 是否有 TLS
	tlsHosts := make(map[string]bool)
	for _, t := range src.Spec.TLS {
		for _, h := range t.Hosts {
			tlsHosts[h] = true
		}
	}

	var rows []model.IngressItem
	for _, rule := range src.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}
		hasTLS := tlsHosts[rule.Host]

		for _, p := range rule.Paths {
			svcName, svcPort := extractBackend(p.Backend)
			rows = append(rows, model.IngressItem{
				Name:        src.Summary.Name,
				Namespace:   src.Summary.Namespace,
				Host:        host,
				Path:        p.Path,
				ServiceName: svcName,
				ServicePort: svcPort,
				TLS:         hasTLS,
				CreatedAt:   createdAt,
			})
		}

		// 无 paths 的 rule 也要展示
		if len(rule.Paths) == 0 {
			rows = append(rows, model.IngressItem{
				Name:      src.Summary.Name,
				Namespace: src.Summary.Namespace,
				Host:      host,
				Path:      "/",
				TLS:       hasTLS,
				CreatedAt: createdAt,
			})
		}
	}

	// 无 rules 时，用 defaultBackend
	if len(src.Spec.Rules) == 0 {
		svcName, svcPort := extractBackend(src.Spec.DefaultBackend)
		rows = append(rows, model.IngressItem{
			Name:        src.Summary.Name,
			Namespace:   src.Summary.Namespace,
			Host:        "*",
			Path:        "/",
			ServiceName: svcName,
			ServicePort: svcPort,
			TLS:         src.Summary.TLSEnabled,
			CreatedAt:   createdAt,
		})
	}

	return rows
}

// extractBackend 从 IngressBackend 提取 service name 和 port
func extractBackend(b *model_v2.IngressBackend) (string, string) {
	if b == nil || b.Service == nil {
		return "", ""
	}
	port := b.Service.PortName
	if port == "" && b.Service.PortNumber > 0 {
		port = fmt.Sprintf("%d", b.Service.PortNumber)
	}
	return b.Service.Name, port
}

// IngressDetail 转换为详情（扁平 + 嵌套 spec/status）
func IngressDetail(src *model_v2.Ingress) model.IngressDetail {
	return model.IngressDetail{
		Name:      src.Summary.Name,
		Namespace: src.Summary.Namespace,
		Class:     src.Summary.IngressClass,
		CreatedAt: src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:       src.Summary.Age,

		Hosts:        src.Summary.Hosts,
		TLSEnabled:   src.Summary.TLSEnabled,
		LoadBalancer: src.Status.LoadBalancer,

		Spec:   src.Spec,
		Status: src.Status,

		Annotations: src.Annotations,
	}
}

// ui_interfaces/ingress/overview.go
package ingress

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/ingress"
)

// BuildIngressOverview —— 拉取全集群 Ingress 并构建概览 DTO
func BuildIngressOverview(ctx context.Context, clusterID string) (*IngressOverviewDTO, error) {
	list, err := datasource.GetIngressListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	dto := fromModelToOverview(list)
	return &dto, nil
}

func fromModelToOverview(items []mod.Ingress) IngressOverviewDTO {
	rows := make([]IngressRowSimple, 0, 64)

	hostSet := make(map[string]struct{})
	totalTLS := 0
	totalPaths := 0

	for _, in := range items {
		// 统计 hosts
		for _, h := range in.Summary.Hosts {
			if h == "" {
				continue
			}
			hostSet[h] = struct{}{}
		}
		// 统计 TLS 条目
		totalTLS += len(in.Spec.TLS)

		// 展平成表格行：每条 Path 一行
		tlsJoined := joinHostsFromTLS(in.Spec.TLS)
		created := in.Summary.CreatedAt

		if len(in.Spec.Rules) == 0 {
			// 没有 rules：只展示 defaultBackend（若存在）
			rows = append(rows, IngressRowSimple{
				Name:        in.Summary.Name,
				Namespace:   in.Summary.Namespace,
				Host:        "", // all
				Path:        "/",
				ServiceName: backendServiceName(in.Spec.DefaultBackend),
				ServicePort: backendServicePortString(in.Spec.DefaultBackend),
				TLS:         tlsJoined,
				CreatedAt:   created,
			})
			continue
		}

		for _, r := range in.Spec.Rules {
			host := r.Host
			if host != "" {
				hostSet[host] = struct{}{}
			}
			for _, p := range r.Paths {
				path := strings.TrimSpace(p.Path)
				if path == "" {
					path = "/"
				}
				rows = append(rows, IngressRowSimple{
					Name:        in.Summary.Name,
					Namespace:   in.Summary.Namespace,
					Host:        host,
					Path:        path,
					ServiceName: backendServiceName(&p.Backend),
					ServicePort: backendServicePortString(&p.Backend),
					TLS:         tlsJoined,
					CreatedAt:   created,
				})
				totalPaths++
			}
		}
	}

	return IngressOverviewDTO{
		Cards: OverviewCards{
			TotalIngresses: len(items),
			UsedHosts:      len(hostSet),
			TLSCerts:       totalTLS,
			TotalPaths:     totalPaths,
		},
		Rows: rows,
	}
}

// atlhyper_master/interfaces/ui_interfaces/namespace/namespace_detail.go
package namespace

import (
	"context"
	"fmt"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/namespace"
)

// BuildNamespaceDetail —— 根据 clusterID + namespaceName 返回单个详情
func BuildNamespaceDetail(ctx context.Context, clusterID, name string) (*NamespaceDetailDTO, error) {
	list, err := datasource.GetNamespaceListLatest(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	for _, ns := range list {
		if ns.Summary.Name == name {
			dto := fromModelToDetail(ns)
			return &dto, nil
		}
	}
	return nil, fmt.Errorf("namespace not found: %s (cluster=%s)", name, clusterID)
}

func fromModelToDetail(ns mod.Namespace) NamespaceDetailDTO {
	out := NamespaceDetailDTO{
		// 基本
		Name:      ns.Summary.Name,
		Phase:     ns.Summary.Phase,
		CreatedAt: ns.Summary.CreatedAt,
		Age:       ns.Summary.Age,

		// 概览
		Labels:          ns.Summary.Labels,
		Annotations:     ns.Summary.Annotations,
		LabelCount:      mapLen(ns.Summary.Labels),
		AnnotationCount: mapLen(ns.Summary.Annotations),

		// 计数
		Pods:            ns.Counts.Pods,
		PodsRunning:     ns.Counts.PodsRunning,
		PodsPending:     ns.Counts.PodsPending,
		PodsFailed:      ns.Counts.PodsFailed,
		PodsSucceeded:   ns.Counts.PodsSucceeded,
		Deployments:     ns.Counts.Deployments,
		StatefulSets:    ns.Counts.StatefulSets,
		DaemonSets:      ns.Counts.DaemonSets,
		Jobs:            ns.Counts.Jobs,
		CronJobs:        ns.Counts.CronJobs,
		Services:        ns.Counts.Services,
		Ingresses:       ns.Counts.Ingresses,
		ConfigMaps:      ns.Counts.ConfigMaps,
		Secrets:         ns.Counts.Secrets,
		PVCs:            ns.Counts.PVCs,
		NetworkPolicies: ns.Counts.NetworkPolicies,
		ServiceAccounts: ns.Counts.ServiceAccounts,

		Badges: ns.Badges,
	}

	// 配额
	if len(ns.Quotas) > 0 {
		out.Quotas = make([]ResourceQuotaDTO, 0, len(ns.Quotas))
		for _, q := range ns.Quotas {
			out.Quotas = append(out.Quotas, ResourceQuotaDTO{
				Name:   q.Name,
				Scopes: q.Scopes,
				Hard:   q.Hard,
				Used:   q.Used,
			})
		}
	}

	// LimitRanges
	if len(ns.LimitRanges) > 0 {
		out.LimitRanges = make([]LimitRangeDTO, 0, len(ns.LimitRanges))
		for _, lr := range ns.LimitRanges {
			itemDTOs := make([]LimitRangeItemDTO, 0, len(lr.Items))
			for _, it := range lr.Items {
				itemDTOs = append(itemDTOs, LimitRangeItemDTO{
					Type:                 it.Type,
					Max:                  it.Max,
					Min:                  it.Min,
					Default:              it.Default,
					DefaultRequest:       it.DefaultRequest,
					MaxLimitRequestRatio: it.MaxLimitRequestRatio,
				})
			}
			out.LimitRanges = append(out.LimitRanges, LimitRangeDTO{
				Name:  lr.Name,
				Items: itemDTOs,
			})
		}
	}

	// 指标
	if ns.Metrics != nil {
		out.Metrics = &NamespaceMetricsDTO{
			CPU: ResourceAggDTO{
				Usage:     ns.Metrics.CPU.Usage,
				Requests:  ns.Metrics.CPU.Requests,
				Limits:    ns.Metrics.CPU.Limits,
				UtilPct:   ns.Metrics.CPU.UtilPct,
				UtilBasis: ns.Metrics.CPU.UtilBasis,
				QuotaHard: ns.Metrics.CPU.QuotaHard,
				QuotaUsed: ns.Metrics.CPU.QuotaUsed,
			},
			Memory: ResourceAggDTO{
				Usage:     ns.Metrics.Memory.Usage,
				Requests:  ns.Metrics.Memory.Requests,
				Limits:    ns.Metrics.Memory.Limits,
				UtilPct:   ns.Metrics.Memory.UtilPct,
				UtilBasis: ns.Metrics.Memory.UtilBasis,
				QuotaHard: ns.Metrics.Memory.QuotaHard,
				QuotaUsed: ns.Metrics.Memory.QuotaUsed,
			},
		}
	}

	return out
}

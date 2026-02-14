// atlhyper_master_v2/model/convert/namespace.go
// model_v2.Namespace → model.NamespaceItem / model.NamespaceDetail 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// NamespaceItem 转换为列表项（扁平）
func NamespaceItem(src *model_v2.Namespace) model.NamespaceItem {
	return model.NamespaceItem{
		Name:            src.Summary.Name,
		Status:          src.Status.Phase,
		PodCount:        src.Resources.Pods,
		LabelCount:      len(src.Labels),
		AnnotationCount: len(src.Annotations),
		CreatedAt:       src.Summary.CreatedAt,
	}
}

// NamespaceItems 转换多个 Namespace 为列表项
func NamespaceItems(src []model_v2.Namespace) []model.NamespaceItem {
	if src == nil {
		return []model.NamespaceItem{}
	}
	result := make([]model.NamespaceItem, len(src))
	for i := range src {
		result[i] = NamespaceItem(&src[i])
	}
	return result
}

// NamespaceDetail 转换为详情（扁平）
func NamespaceDetail(src *model_v2.Namespace) model.NamespaceDetail {
	return model.NamespaceDetail{
		Name:      src.Summary.Name,
		Phase:     src.Status.Phase,
		CreatedAt: src.Summary.CreatedAt,
		Age:       src.Summary.Age,

		Labels:          src.Labels,
		Annotations:     src.Annotations,
		LabelCount:      len(src.Labels),
		AnnotationCount: len(src.Annotations),

		Pods:          src.Resources.Pods,
		PodsRunning:   src.Resources.PodsRunning,
		PodsPending:   src.Resources.PodsPending,
		PodsFailed:    src.Resources.PodsFailed,
		PodsSucceeded: src.Resources.PodsSucceeded,

		Deployments:  src.Resources.Deployments,
		StatefulSets: src.Resources.StatefulSets,
		DaemonSets:   src.Resources.DaemonSets,
		Jobs:         src.Resources.Jobs,
		CronJobs:     src.Resources.CronJobs,

		Services:        src.Resources.Services,
		Ingresses:       src.Resources.Ingresses,
		NetworkPolicies: src.Resources.NetworkPolicies,

		ConfigMaps:             src.Resources.ConfigMaps,
		Secrets:                src.Resources.Secrets,
		PersistentVolumeClaims: src.Resources.PVCs,
		ServiceAccounts:        src.Resources.ServiceAccounts,

		Quotas:      src.Quotas,
		LimitRanges: src.LimitRanges,
	}
}

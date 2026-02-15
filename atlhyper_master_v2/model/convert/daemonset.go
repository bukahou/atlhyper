// atlhyper_master_v2/model/convert/daemonset.go
// model_v2.DaemonSet → model.DaemonSetDetail 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// DaemonSetItem 转换为列表项（扁平）
func DaemonSetItem(src *model_v2.DaemonSet) model.DaemonSetItem {
	return model.DaemonSetItem{
		Name:         src.Summary.Name,
		Namespace:    src.Summary.Namespace,
		Desired:      src.Summary.DesiredNumberScheduled,
		Current:      src.Summary.CurrentNumberScheduled,
		Ready:        src.Summary.NumberReady,
		Available:    src.Summary.NumberAvailable,
		Misscheduled: src.Summary.NumberMisscheduled,
		CreatedAt:    src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:          src.Summary.Age,
	}
}

// DaemonSetItems 转换多个 DaemonSet 为列表项
func DaemonSetItems(src []model_v2.DaemonSet) []model.DaemonSetItem {
	if src == nil {
		return []model.DaemonSetItem{}
	}
	result := make([]model.DaemonSetItem, len(src))
	for i := range src {
		result[i] = DaemonSetItem(&src[i])
	}
	return result
}

// DaemonSetDetail 转换为详情（扁平顶层 + 嵌套子结构）
func DaemonSetDetail(src *model_v2.DaemonSet) model.DaemonSetDetail {
	return model.DaemonSetDetail{
		Name:             src.Summary.Name,
		Namespace:        src.Summary.Namespace,
		Desired:          src.Summary.DesiredNumberScheduled,
		Current:          src.Summary.CurrentNumberScheduled,
		Ready:            src.Summary.NumberReady,
		Available:        src.Summary.NumberAvailable,
		Unavailable:      src.Summary.NumberUnavailable,
		Misscheduled:     src.Summary.NumberMisscheduled,
		UpdatedScheduled: src.Summary.UpdatedNumberScheduled,
		CreatedAt:        src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:              src.Summary.Age,
		Selector:         src.Summary.Selector,

		Spec:       src.Spec,
		Template:   src.Template,
		Status:     src.Status,
		Conditions: src.Status.Conditions,
		Rollout:    src.Rollout,

		Labels:      src.Labels,
		Annotations: src.Annotations,
	}
}

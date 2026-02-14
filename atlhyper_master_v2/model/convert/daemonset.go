// atlhyper_master_v2/model/convert/daemonset.go
// model_v2.DaemonSet → model.DaemonSetDetail 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

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

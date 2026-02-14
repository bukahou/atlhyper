// atlhyper_master_v2/model/convert/statefulset.go
// model_v2.StatefulSet → model.StatefulSetDetail 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// StatefulSetDetail 转换为详情（扁平顶层 + 嵌套子结构）
func StatefulSetDetail(src *model_v2.StatefulSet) model.StatefulSetDetail {
	return model.StatefulSetDetail{
		Name:        src.Summary.Name,
		Namespace:   src.Summary.Namespace,
		Replicas:    src.Summary.Replicas,
		Ready:       src.Summary.Ready,
		Current:     src.Summary.Current,
		Updated:     src.Summary.Updated,
		Available:   src.Summary.Available,
		CreatedAt:   src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:         src.Summary.Age,
		ServiceName: src.Summary.ServiceName,
		Selector:    src.Summary.Selector,

		Spec:       src.Spec,
		Template:   src.Template,
		Status:     src.Status,
		Conditions: src.Status.Conditions,
		Rollout:    src.Rollout,

		Labels:      src.Labels,
		Annotations: src.Annotations,
	}
}

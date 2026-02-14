// atlhyper_master_v2/model/convert/deployment.go
// model_v2.Deployment → model.DeploymentItem / model.DeploymentDetail 转换函数
package convert

import (
	"fmt"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// DeploymentItem 转换为列表项（扁平）
func DeploymentItem(src *model_v2.Deployment) model.DeploymentItem {
	image := ""
	if len(src.Template.Containers) > 0 {
		image = src.Template.Containers[0].Image
	}

	return model.DeploymentItem{
		Name:       src.Summary.Name,
		Namespace:  src.Summary.Namespace,
		Image:      image,
		Replicas:   fmt.Sprintf("%d/%d", src.Summary.Ready, src.Summary.Replicas),
		LabelCount: len(src.Labels),
		AnnoCount:  len(src.Annotations),
		CreatedAt:  src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// DeploymentItems 转换多个 Deployment 为列表项
func DeploymentItems(src []model_v2.Deployment) []model.DeploymentItem {
	if src == nil {
		return []model.DeploymentItem{}
	}
	result := make([]model.DeploymentItem, len(src))
	for i := range src {
		result[i] = DeploymentItem(&src[i])
	}
	return result
}

// DeploymentDetail 转换为详情（扁平顶层 + 嵌套子结构）
func DeploymentDetail(src *model_v2.Deployment) model.DeploymentDetail {
	return model.DeploymentDetail{
		Name:        src.Summary.Name,
		Namespace:   src.Summary.Namespace,
		Strategy:    src.Summary.Strategy,
		Replicas:    src.Summary.Replicas,
		Updated:     src.Summary.Updated,
		Ready:       src.Summary.Ready,
		Available:   src.Summary.Available,
		Unavailable: src.Summary.Unavailable,
		Paused:      src.Summary.Paused,
		Selector:    src.Summary.Selector,
		CreatedAt:   src.Summary.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Age:         src.Summary.Age,

		Spec:        src.Spec,
		Template:    src.Template,
		Status:      src.Status,
		Conditions:  src.Status.Conditions,
		Rollout:     src.Rollout,
		ReplicaSets: src.ReplicaSets,

		Labels:      src.Labels,
		Annotations: src.Annotations,
	}
}

// atlhyper_master_v2/model/convert/job.go
// model_v2.Job → model.JobItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// JobItem 转换为列表项
func JobItem(src *model_v2.Job) model.JobItem {
	return model.JobItem{
		Name:       src.Name,
		Namespace:  src.Namespace,
		Active:     src.Active,
		Succeeded:  src.Succeeded,
		Failed:     src.Failed,
		Complete:   src.Complete,
		StartTime:  formatTimePtr(src.StartTime),
		FinishTime: formatTimePtr(src.FinishTime),
		CreatedAt:  src.CreatedAt.Format(timeFormat),
		Age:        formatAge(src.CreatedAt),
	}
}

// JobItems 转换多个 Job 为列表项
func JobItems(src []model_v2.Job) []model.JobItem {
	if src == nil {
		return []model.JobItem{}
	}
	result := make([]model.JobItem, len(src))
	for i := range src {
		result[i] = JobItem(&src[i])
	}
	return result
}

// JobDetail 转换为详情
func JobDetail(src *model_v2.Job) model.JobDetail {
	return model.JobDetail{
		Name:      src.Name,
		Namespace: src.Namespace,
		UID:       src.UID,
		OwnerKind: src.OwnerKind,
		OwnerName: src.OwnerName,
		CreatedAt: src.CreatedAt.Format(timeFormat),
		Age:       formatAge(src.CreatedAt),

		Status:    jobStatus(src),
		Active:    src.Active,
		Succeeded: src.Succeeded,
		Failed:    src.Failed,

		StartTime:  formatTimePtr(src.StartTime),
		FinishTime: formatTimePtr(src.FinishTime),
		Duration:   formatDuration(src.StartTime, src.FinishTime),

		Labels: src.Labels,
	}
}

// jobStatus 计算 Job 状态文本
func jobStatus(src *model_v2.Job) string {
	if src.Complete {
		return "Complete"
	}
	if src.Active > 0 {
		return "Running"
	}
	if src.Failed > 0 {
		return "Failed"
	}
	return "Complete"
}

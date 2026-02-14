// atlhyper_master_v2/model/convert/cronjob.go
// model_v2.CronJob → model.CronJobItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// CronJobItem 转换为列表项
func CronJobItem(src *model_v2.CronJob) model.CronJobItem {
	return model.CronJobItem{
		Name:               src.Name,
		Namespace:          src.Namespace,
		Schedule:           src.Schedule,
		Suspend:            src.Suspend,
		ActiveJobs:         src.ActiveJobs,
		LastScheduleTime:   formatTimePtr(src.LastScheduleTime),
		LastSuccessfulTime: formatTimePtr(src.LastSuccessfulTime),
		CreatedAt:          src.CreatedAt.Format(timeFormat),
		Age:                formatAge(src.CreatedAt),
	}
}

// CronJobItems 转换多个 CronJob 为列表项
func CronJobItems(src []model_v2.CronJob) []model.CronJobItem {
	if src == nil {
		return []model.CronJobItem{}
	}
	result := make([]model.CronJobItem, len(src))
	for i := range src {
		result[i] = CronJobItem(&src[i])
	}
	return result
}

// CronJobDetail 转换为详情
func CronJobDetail(src *model_v2.CronJob) model.CronJobDetail {
	return model.CronJobDetail{
		Name:      src.Name,
		Namespace: src.Namespace,
		UID:       src.UID,
		OwnerKind: src.OwnerKind,
		OwnerName: src.OwnerName,
		CreatedAt: src.CreatedAt.Format(timeFormat),
		Age:       formatAge(src.CreatedAt),

		Schedule:   src.Schedule,
		Suspend:    src.Suspend,
		ActiveJobs: src.ActiveJobs,

		LastScheduleTime:   formatTimePtr(src.LastScheduleTime),
		LastSuccessfulTime: formatTimePtr(src.LastSuccessfulTime),
		LastScheduleAgo:    formatTimeAgo(src.LastScheduleTime),
		LastSuccessAgo:     formatTimeAgo(src.LastSuccessfulTime),

		Labels: src.Labels,
	}
}

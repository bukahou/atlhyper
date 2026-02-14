// atlhyper_master_v2/model/convert/limit_range.go
// model_v2.LimitRange → model.LimitRangeItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// LimitRangeItem 转换为列表项
func LimitRangeItem(src *model_v2.LimitRange) model.LimitRangeItem {
	items := make([]model.LimitRangeItemEntry, len(src.Items))
	for i, entry := range src.Items {
		items[i] = model.LimitRangeItemEntry{
			Type:                 entry.Type,
			Default:              entry.Default,
			DefaultRequest:       entry.DefaultRequest,
			Max:                  entry.Max,
			Min:                  entry.Min,
			MaxLimitRequestRatio: entry.MaxLimitRequestRatio,
		}
	}

	return model.LimitRangeItem{
		Name:      src.Name,
		Namespace: src.Namespace,
		Items:     items,
		CreatedAt: src.CreatedAt,
		Age:       src.Age,
	}
}

// LimitRangeItems 转换多个 LimitRange 为列表项
func LimitRangeItems(src []model_v2.LimitRange) []model.LimitRangeItem {
	if src == nil {
		return []model.LimitRangeItem{}
	}
	result := make([]model.LimitRangeItem, len(src))
	for i := range src {
		result[i] = LimitRangeItem(&src[i])
	}
	return result
}

// LimitRangeDetail 转换为详情
func LimitRangeDetail(src *model_v2.LimitRange) model.LimitRangeDetail {
	items := make([]model.LimitRangeItemEntry, len(src.Items))
	for i, entry := range src.Items {
		items[i] = model.LimitRangeItemEntry{
			Type:                 entry.Type,
			Default:              entry.Default,
			DefaultRequest:       entry.DefaultRequest,
			Max:                  entry.Max,
			Min:                  entry.Min,
			MaxLimitRequestRatio: entry.MaxLimitRequestRatio,
		}
	}

	return model.LimitRangeDetail{
		Name:        src.Name,
		Namespace:   src.Namespace,
		Items:       items,
		CreatedAt:   src.CreatedAt,
		Age:         src.Age,
		Labels:      src.Labels,
		Annotations: src.Annotations,
	}
}

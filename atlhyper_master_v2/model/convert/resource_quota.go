// atlhyper_master_v2/model/convert/resource_quota.go
// model_v2.ResourceQuota → model.ResourceQuotaItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// ResourceQuotaItem 转换为列表项
func ResourceQuotaItem(src *model_v2.ResourceQuota) model.ResourceQuotaItem {
	return model.ResourceQuotaItem{
		Name:      src.Name,
		Namespace: src.Namespace,
		Scopes:    src.Scopes,
		Hard:      src.Hard,
		Used:      src.Used,
		CreatedAt: src.CreatedAt,
		Age:       src.Age,
	}
}

// ResourceQuotaItems 转换多个 ResourceQuota 为列表项
func ResourceQuotaItems(src []model_v2.ResourceQuota) []model.ResourceQuotaItem {
	if src == nil {
		return []model.ResourceQuotaItem{}
	}
	result := make([]model.ResourceQuotaItem, len(src))
	for i := range src {
		result[i] = ResourceQuotaItem(&src[i])
	}
	return result
}

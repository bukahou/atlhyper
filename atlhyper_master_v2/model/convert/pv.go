// atlhyper_master_v2/model/convert/pv.go
// model_v2.PersistentVolume → model.PVItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// PVItem 转换为列表项
func PVItem(src *model_v2.PersistentVolume) model.PVItem {
	return model.PVItem{
		Name:          src.Name,
		Capacity:      src.Capacity,
		Phase:         src.Phase,
		StorageClass:  src.StorageClass,
		AccessModes:   src.AccessModes,
		ReclaimPolicy: src.ReclaimPolicy,
		CreatedAt:     src.CreatedAt.Format(timeFormat),
		Age:           formatAge(src.CreatedAt),
	}
}

// PVDetail 转换为详情
func PVDetail(src *model_v2.PersistentVolume) model.PVDetail {
	return model.PVDetail{
		Name:             src.Name,
		UID:              src.UID,
		Capacity:         src.Capacity,
		Phase:            src.Phase,
		StorageClass:     src.StorageClass,
		AccessModes:      src.AccessModes,
		ReclaimPolicy:    src.ReclaimPolicy,
		VolumeSourceType: src.VolumeSourceType,
		ClaimRefName:     src.ClaimRefName,
		ClaimRefNS:       src.ClaimRefNS,
		CreatedAt:        src.CreatedAt.Format(timeFormat),
		Age:              formatAge(src.CreatedAt),
		Labels:           src.Labels,
	}
}

// PVItems 转换多个 PersistentVolume 为列表项
func PVItems(src []model_v2.PersistentVolume) []model.PVItem {
	if src == nil {
		return []model.PVItem{}
	}
	result := make([]model.PVItem, len(src))
	for i := range src {
		result[i] = PVItem(&src[i])
	}
	return result
}

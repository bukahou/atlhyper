// atlhyper_master_v2/model/convert/pvc.go
// model_v2.PersistentVolumeClaim → model.PVCItem 转换函数
package convert

import (
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v2"
)

// PVCItem 转换为列表项
func PVCItem(src *model_v2.PersistentVolumeClaim) model.PVCItem {
	return model.PVCItem{
		Name:              src.Name,
		Namespace:         src.Namespace,
		Phase:             src.Phase,
		VolumeName:        src.VolumeName,
		StorageClass:      src.StorageClass,
		AccessModes:       src.AccessModes,
		RequestedCapacity: src.RequestedCapacity,
		ActualCapacity:    src.ActualCapacity,
		CreatedAt:         src.CreatedAt.Format(timeFormat),
		Age:               formatAge(src.CreatedAt),
	}
}

// PVCDetail 转换为详情
func PVCDetail(src *model_v2.PersistentVolumeClaim) model.PVCDetail {
	return model.PVCDetail{
		Name:              src.Name,
		Namespace:         src.Namespace,
		UID:               src.UID,
		Phase:             src.Phase,
		VolumeName:        src.VolumeName,
		StorageClass:      src.StorageClass,
		AccessModes:       src.AccessModes,
		RequestedCapacity: src.RequestedCapacity,
		ActualCapacity:    src.ActualCapacity,
		CreatedAt:         src.CreatedAt.Format(timeFormat),
		Age:               formatAge(src.CreatedAt),
		Labels:            src.Labels,
	}
}

// PVCItems 转换多个 PersistentVolumeClaim 为列表项
func PVCItems(src []model_v2.PersistentVolumeClaim) []model.PVCItem {
	if src == nil {
		return []model.PVCItem{}
	}
	result := make([]model.PVCItem, len(src))
	for i := range src {
		result[i] = PVCItem(&src[i])
	}
	return result
}

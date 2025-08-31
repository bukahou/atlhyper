// ui_interfaces/configmap/detail.go
package configmap

import (
	"context"
	"fmt"
	"sort"

	"AtlHyper/atlhyper_master/interfaces/datasource"
	mod "AtlHyper/model/configmap"
)

// BuildConfigMapListFullByNamespace —— 指定 Namespace 下的 ConfigMap 列表（完整体：含 Data/Binary）
func BuildConfigMapListFullByNamespace(ctx context.Context, clusterID, namespace string) ([]ConfigMapDTO, error) {
	if namespace == "" {
		return nil, fmt.Errorf("namespace required")
	}

	// 如果后端已有按 ns 拉取的接口，优先使用：
	// list, err := datasource.GetConfigMapListByNamespaceLatest(ctx, clusterID, namespace)
	// 这里先复用全量 + 过滤
	list, err := datasource.GetConfigMapListLatest(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get configmap list failed: %w", err)
	}

	// 先筛，再预分配
	filtered := make([]mod.ConfigMap, 0, len(list))
	for _, cm := range list {
		if cm.Summary.Namespace == namespace {
			filtered = append(filtered, cm)
		}
	}

	// 稳定排序：按名称（也可改为按创建时间/大小等）
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Summary.Name < filtered[j].Summary.Name
	})

	// 转 DTO（完整体 withData=true）
	dtos := make([]ConfigMapDTO, 0, len(filtered))
	for _, cm := range filtered {
		dtos = append(dtos, fromModel(cm, true))
	}
	return dtos, nil
}

// fromModel —— 转换内部模型 → DTO
func fromModel(cm mod.ConfigMap, withData bool) ConfigMapDTO {
	dto := ConfigMapDTO{
		Name:                 cm.Summary.Name,
		Namespace:            cm.Summary.Namespace,
		CreatedAt:            cm.Summary.CreatedAt,
		Age:                  cm.Summary.Age,
		Immutable:            cm.Summary.Immutable,
		Labels:               cm.Summary.Labels,
		Annotations:          cm.Summary.Annotations,
		Keys:                 cm.Summary.Keys,
		BinaryKeys:           cm.Summary.BinaryKeys,
		TotalSizeBytes:       cm.Summary.TotalSizeBytes,
		BinaryTotalSizeBytes: cm.Summary.BinaryTotalSizeBytes,
	}

	if withData {
		// 预分配，避免多次扩容
		if n := len(cm.Data); n > 0 {
			dto.Data = make([]DataEntryDTO, 0, n)
			for _, d := range cm.Data {
				dto.Data = append(dto.Data, DataEntryDTO{
					Key:       d.Key,
					Size:      d.Size,
					Preview:   d.Preview,
					Truncated: d.Truncated,
				})
			}
		}
		if m := len(cm.Binary); m > 0 {
			dto.Binary = make([]BinaryEntryDTO, 0, m)
			for _, b := range cm.Binary {
				dto.Binary = append(dto.Binary, BinaryEntryDTO{
					Key:  b.Key,
					Size: b.Size,
				})
			}
		}
	}

	return dto
}

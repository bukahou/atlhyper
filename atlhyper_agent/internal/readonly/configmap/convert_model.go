package configmap

import (
	"sort"

	modelcm "AtlHyper/model/configmap"

	corev1 "k8s.io/api/core/v1"
)

const (
	// 单键最大预览字节（UTF-8），超过会截断并置 Truncated=true
	previewLimitBytes = 2 * 1024 // 2 KiB，可按需调大/调小
)

// buildModel —— 基于 corev1.ConfigMap 构建轻量模型（含 Summary / Data 预览 / Binary 尺寸）
func buildModel(cm *corev1.ConfigMap) modelcm.ConfigMap {
	// 1) Data / Binary 的映射（带预览/大小）
	dataEntries, totalDataSize := mapDataEntries(cm.Data, previewLimitBytes)
	binEntries, totalBinSize := mapBinaryEntries(cm.BinaryData)

	// 2) Summary
	created := cm.CreationTimestamp.Time
	immutable := cm.Immutable != nil && *cm.Immutable

	sum := modelcm.ConfigMapSummary{
		Name:                 cm.Name,
		Namespace:            cm.Namespace,
		Labels:               cm.Labels,
		Annotations:          pickAnnotations(cm.Annotations),
		CreatedAt:            created,
		Age:                  fmtAge(created),
		ResourceVersion:      cm.ResourceVersion,
		Immutable:            immutable,
		Keys:                 len(dataEntries),
		BinaryKeys:           len(binEntries),
		TotalSizeBytes:       totalDataSize,
		BinaryTotalSizeBytes: totalBinSize,
	}

	return modelcm.ConfigMap{
		Summary: sum,
		Data:    dataEntries,
		Binary:  binEntries,
	}
}

// mapDataEntries —— 把 .data 映射为 DataEntry（带 Size / Preview / Truncated）
func mapDataEntries(m map[string]string, limit int) ([]modelcm.DataEntry, int64) {
	if len(m) == 0 {
		return nil, 0
	}
	out := make([]modelcm.DataEntry, 0, len(m))
	var total int64

	// 为稳定输出排序 key
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		size := utf8ByteLen(v)
		total += int64(size)

		prev, trunc := previewUTF8(v, limit)
		out = append(out, modelcm.DataEntry{
			Key:       k,
			Size:      size,
			Preview:   prev,
			Truncated: trunc,
		})
	}
	return out, total
}

// mapBinaryEntries —— 把 .binaryData 映射为 BinaryEntry（只报告大小；可选哈希）
func mapBinaryEntries(m map[string][]byte) ([]modelcm.BinaryEntry, int64) {
	if len(m) == 0 {
		return nil, 0
	}
	out := make([]modelcm.BinaryEntry, 0, len(m))
	var total int64

	// 稳定排序
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		size := len(m[k])
		total += int64(size)
		out = append(out, modelcm.BinaryEntry{
			Key:  k,
			Size: size,
			// 如需哈希：在此处计算并赋值
			// SHA256: hex.EncodeToString(sum256(m[k])),
		})
	}
	return out, total
}

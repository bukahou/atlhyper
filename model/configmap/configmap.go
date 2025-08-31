// model/configmap/configmap.go
package configmap

import "time"

// ConfigMap 是对 k8s/core/v1 ConfigMap 的精简投影。
// - 体积控制：默认不原样携带大 Value；用 DataEntry/BinaryEntry 提供 key、大小、可选预览/哈希。
// - 可观测性：Summary 中提供键数、总字节数、是否 immutable 等元信息。
type ConfigMap struct {
	Summary ConfigMapSummary `json:"summary"`
	// Data：字符串键值（来自 .data）。通常只放“预览 + 大小”，避免上报体积过大。
	Data []DataEntry `json:"data,omitempty"`
	// Binary：二进制键值（来自 .binaryData）。不直接携带原始字节，建议只放大小/哈希。
	Binary []BinaryEntry `json:"binary,omitempty"`
}

// ConfigMapSummary 描述基础元信息与聚合统计。
type ConfigMapSummary struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Labels     map[string]string `json:"labels,omitempty"`
	// 可按需挑选常见注解（例如管理类注解），避免把所有注解原样带上导致体积膨胀
	Annotations     map[string]string `json:"annotations,omitempty"`
	CreatedAt       time.Time         `json:"createdAt,omitempty"`
	Age             string            `json:"age,omitempty"` // 友好时长，如 "5d3h"
	ResourceVersion string            `json:"resourceVersion,omitempty"`
	Immutable       bool              `json:"immutable,omitempty"` // 对应 .immutable

	// 统计信息（用于快速展示/列表聚合）
	Keys                int   `json:"keys"`                // .data 的键数量
	BinaryKeys          int   `json:"binaryKeys"`          // .binaryData 的键数量
	TotalSizeBytes      int64 `json:"totalSizeBytes"`      // 估算 .data 全部值的总字节数（UTF-8 字节）
	BinaryTotalSizeBytes int64 `json:"binaryTotalSizeBytes"` // 估算 .binaryData 全部值的总字节数
}

// DataEntry 描述一个字符串键（.data[key]）。
// - 为了控制体积，建议 readonly 层仅填充 Preview（最多 N 字节）与 Size；超限时置 Truncated=true。
type DataEntry struct {
	Key       string `json:"key"`
	Size      int    `json:"size"`                 // 原始值的字节长度（UTF-8）
	Preview   string `json:"preview,omitempty"`    // 预览片段（可能是全量或截断后的前缀）
	Truncated bool   `json:"truncated,omitempty"`  // 预览是否被截断
	// 可选：为变更对比/缓存命中加入哈希（例如 sha256），按需开启
	// SHA256 string `json:"sha256,omitempty"`
}

// BinaryEntry 描述一个二进制键（.binaryData[key]）。
// - 不直接携带原始字节；只提供大小与可选哈希，避免上报体积与敏感数据风险。
type BinaryEntry struct {
	Key   string `json:"key"`
	Size  int    `json:"size"`              // 原始字节长度
	// 可选：按需开启内容哈希，方便主端比对是否变化
	// SHA256 string `json:"sha256,omitempty"`
}

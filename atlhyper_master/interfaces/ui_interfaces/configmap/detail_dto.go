// ui_interfaces/configmap/dto.go
package configmap

import "time"

// ConfigMapDTO —— 对外传输对象，列表/详情共用
type ConfigMapDTO struct {
	// 基本信息
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	CreatedAt  time.Time         `json:"createdAt,omitempty"`
	Age        string            `json:"age,omitempty"`
	Immutable  bool              `json:"immutable,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Annotations map[string]string`json:"annotations,omitempty"`

	// 统计信息
	Keys                int   `json:"keys"`
	BinaryKeys          int   `json:"binaryKeys"`
	TotalSizeBytes      int64 `json:"totalSizeBytes"`
	BinaryTotalSizeBytes int64`json:"binaryTotalSizeBytes"`

	// 数据内容（详情时展示）
	Data   []DataEntryDTO   `json:"data,omitempty"`
	Binary []BinaryEntryDTO `json:"binary,omitempty"`
}

// DataEntryDTO —— 字符串键值（.data）
type DataEntryDTO struct {
	Key       string `json:"key"`
	Size      int    `json:"size"`
	Preview   string `json:"preview,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// BinaryEntryDTO —— 二进制键值（.binaryData）
type BinaryEntryDTO struct {
	Key  string `json:"key"`
	Size int    `json:"size"`
}

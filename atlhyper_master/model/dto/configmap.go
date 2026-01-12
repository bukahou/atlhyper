// atlhyper_master/dto/ui/configmap.go
// ConfigMap UI DTOs
package dto

import "time"

// ConfigMapDTO - 列表/详情共用
type ConfigMapDTO struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	Age         string            `json:"age,omitempty"`
	Immutable   bool              `json:"immutable,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Keys                 int   `json:"keys"`
	BinaryKeys           int   `json:"binaryKeys"`
	TotalSizeBytes       int64 `json:"totalSizeBytes"`
	BinaryTotalSizeBytes int64 `json:"binaryTotalSizeBytes"`

	Data   []ConfigMapDataEntry   `json:"data,omitempty"`
	Binary []ConfigMapBinaryEntry `json:"binary,omitempty"`
}

type ConfigMapDataEntry struct {
	Key       string `json:"key"`
	Size      int    `json:"size"`
	Preview   string `json:"preview,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

type ConfigMapBinaryEntry struct {
	Key  string `json:"key"`
	Size int    `json:"size"`
}

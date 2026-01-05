// model/k8s/configmap.go
// ConfigMap 资源模型
package k8s

import "time"

// ConfigMap 是对 k8s/core/v1 ConfigMap 的精简投影
type ConfigMap struct {
	Summary ConfigMapSummary `json:"summary"`
	Data    []DataEntry      `json:"data,omitempty"`
	Binary  []BinaryEntry    `json:"binary,omitempty"`
}

// ConfigMapSummary 描述基础元信息与聚合统计
type ConfigMapSummary struct {
	Name                 string            `json:"name"`
	Namespace            string            `json:"namespace"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
	CreatedAt            time.Time         `json:"createdAt,omitempty"`
	Age                  string            `json:"age,omitempty"`
	ResourceVersion      string            `json:"resourceVersion,omitempty"`
	Immutable            bool              `json:"immutable,omitempty"`
	Keys                 int               `json:"keys"`
	BinaryKeys           int               `json:"binaryKeys"`
	TotalSizeBytes       int64             `json:"totalSizeBytes"`
	BinaryTotalSizeBytes int64             `json:"binaryTotalSizeBytes"`
}

// DataEntry 描述一个字符串键
type DataEntry struct {
	Key       string `json:"key"`
	Size      int    `json:"size"`
	Preview   string `json:"preview,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// BinaryEntry 描述一个二进制键
type BinaryEntry struct {
	Key  string `json:"key"`
	Size int    `json:"size"`
}

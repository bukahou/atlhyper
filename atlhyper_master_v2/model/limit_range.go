// atlhyper_master_v2/model/limit_range.go
// LimitRange Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// LimitRangeItem LimitRange 列表项
type LimitRangeItem struct {
	Name      string                `json:"name"`
	Namespace string                `json:"namespace"`
	Items     []LimitRangeItemEntry `json:"items"`
	CreatedAt string                `json:"createdAt"`
	Age       string                `json:"age"`
}

// LimitRangeItemEntry LimitRange 限制项
type LimitRangeItemEntry struct {
	Type                 string            `json:"type"`
	Default              map[string]string `json:"default,omitempty"`
	DefaultRequest       map[string]string `json:"defaultRequest,omitempty"`
	Max                  map[string]string `json:"max,omitempty"`
	Min                  map[string]string `json:"min,omitempty"`
	MaxLimitRequestRatio map[string]string `json:"maxLimitRequestRatio,omitempty"`
}

// LimitRangeDetail LimitRange 详情
type LimitRangeDetail struct {
	Name      string                `json:"name"`
	Namespace string                `json:"namespace"`
	Items     []LimitRangeItemEntry `json:"items"`
	CreatedAt string                `json:"createdAt"`
	Age       string                `json:"age"`

	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

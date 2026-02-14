// atlhyper_master_v2/model/resource_quota.go
// ResourceQuota Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// ResourceQuotaItem ResourceQuota 列表项
type ResourceQuotaItem struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Scopes    []string          `json:"scopes,omitempty"`
	Hard      map[string]string `json:"hard,omitempty"`
	Used      map[string]string `json:"used,omitempty"`
	CreatedAt string            `json:"createdAt"`
	Age       string            `json:"age"`
}

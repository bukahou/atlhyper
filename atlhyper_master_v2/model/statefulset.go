// atlhyper_master_v2/model/statefulset.go
// StatefulSet Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// StatefulSetDetail StatefulSet 详情（扁平顶层 + 嵌套子结构）
type StatefulSetDetail struct {
	// 基本信息（扁平）
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Replicas    int32  `json:"replicas"`
	Ready       int32  `json:"ready"`
	Current     int32  `json:"current"`
	Updated     int32  `json:"updated"`
	Available   int32  `json:"available"`
	CreatedAt   string `json:"createdAt"`
	Age         string `json:"age,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	Selector    string `json:"selector,omitempty"`

	// 嵌套结构（使用 interface{} 保持前端灵活性）
	Spec       interface{} `json:"spec,omitempty"`
	Template   interface{} `json:"template,omitempty"`
	Status     interface{} `json:"status,omitempty"`
	Conditions interface{} `json:"conditions,omitempty"`
	Rollout    interface{} `json:"rollout,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

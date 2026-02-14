// atlhyper_master_v2/model/daemonset.go
// DaemonSet Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// DaemonSetDetail DaemonSet 详情（扁平顶层 + 嵌套子结构）
type DaemonSetDetail struct {
	// 基本信息（扁平）
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	Desired          int32  `json:"desired"`
	Current          int32  `json:"current"`
	Ready            int32  `json:"ready"`
	Available        int32  `json:"available"`
	Unavailable      int32  `json:"unavailable,omitempty"`
	Misscheduled     int32  `json:"misscheduled,omitempty"`
	UpdatedScheduled int32  `json:"updatedScheduled,omitempty"`
	CreatedAt        string `json:"createdAt"`
	Age              string `json:"age,omitempty"`
	Selector         string `json:"selector,omitempty"`

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

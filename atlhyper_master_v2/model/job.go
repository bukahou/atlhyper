// atlhyper_master_v2/model/job.go
// Job Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// JobItem Job 列表项
type JobItem struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Active     int32  `json:"active"`
	Succeeded  int32  `json:"succeeded"`
	Failed     int32  `json:"failed"`
	Complete   bool   `json:"complete"`
	StartTime  string `json:"startTime"`
	FinishTime string `json:"finishTime"`
	CreatedAt  string `json:"createdAt"`
	Age        string `json:"age"`
}

// JobDetail Job 详情
type JobDetail struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
	OwnerKind string `json:"ownerKind,omitempty"`
	OwnerName string `json:"ownerName,omitempty"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`

	// 状态
	Status    string `json:"status"`
	Active    int32  `json:"active"`
	Succeeded int32  `json:"succeeded"`
	Failed    int32  `json:"failed"`

	// 规格
	Completions  *int32 `json:"completions,omitempty"`
	Parallelism  *int32 `json:"parallelism,omitempty"`
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// 时间
	StartTime  string `json:"startTime"`
	FinishTime string `json:"finishTime"`
	Duration   string `json:"duration"`

	// Pod 模板与条件
	Template   interface{} `json:"template,omitempty"`
	Conditions interface{} `json:"conditions,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

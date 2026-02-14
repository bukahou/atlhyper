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

// atlhyper_master_v2/model/cronjob.go
// CronJob Web API 响应类型（camelCase JSON tag，扁平结构）
package model

// CronJobItem CronJob 列表项
type CronJobItem struct {
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	Schedule           string `json:"schedule"`
	Suspend            bool   `json:"suspend"`
	ActiveJobs         int32  `json:"activeJobs"`
	LastScheduleTime   string `json:"lastScheduleTime"`
	LastSuccessfulTime string `json:"lastSuccessfulTime"`
	CreatedAt          string `json:"createdAt"`
	Age                string `json:"age"`
}

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

// CronJobDetail CronJob 详情
type CronJobDetail struct {
	// 基本信息
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
	OwnerKind string `json:"ownerKind,omitempty"`
	OwnerName string `json:"ownerName,omitempty"`
	CreatedAt string `json:"createdAt"`
	Age       string `json:"age"`

	// 调度配置
	Schedule          string `json:"schedule"`
	Suspend           bool   `json:"suspend"`
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty"`
	ActiveJobs        int32  `json:"activeJobs"`

	// 历史保留
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *int32 `json:"failedJobsHistoryLimit,omitempty"`

	// 时间
	LastScheduleTime   string `json:"lastScheduleTime"`
	LastSuccessfulTime string `json:"lastSuccessfulTime"`
	LastScheduleAgo    string `json:"lastScheduleAgo"`
	LastSuccessAgo     string `json:"lastSuccessAgo"`

	// Pod 模板
	Template interface{} `json:"template,omitempty"`

	// 元数据
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

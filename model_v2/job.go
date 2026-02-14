package model_v2

import "time"

// ============================================================
// Job 模型
// ============================================================

// Job K8s Job 资源模型
//
// Job 创建一个或多个 Pod，并确保指定数量的 Pod 成功终止。
type Job struct {
	CommonMeta

	// 状态
	Active    int32 `json:"active"`    // 正在运行的 Pod 数
	Succeeded int32 `json:"succeeded"` // 成功的 Pod 数
	Failed    int32 `json:"failed"`    // 失败的 Pod 数
	Complete  bool  `json:"complete"`  // 是否完成

	// 规格
	Completions  *int32 `json:"completions,omitempty"`  // 期望完成数
	Parallelism  *int32 `json:"parallelism,omitempty"`  // 并行数
	BackoffLimit *int32 `json:"backoff_limit,omitempty"` // 重试次数上限

	// Pod 模板与条件
	Template   PodTemplate         `json:"template"`             // Pod 模板
	Conditions []WorkloadCondition `json:"conditions,omitempty"` // 状态条件

	// 时间
	StartTime  *time.Time `json:"start_time,omitempty"`  // 开始时间
	FinishTime *time.Time `json:"finish_time,omitempty"` // 完成时间
}

// IsRunning 判断 Job 是否正在运行
func (j *Job) IsRunning() bool {
	return j.Active > 0
}

// IsComplete 判断 Job 是否完成
func (j *Job) IsComplete() bool {
	return j.Complete
}

// IsFailed 判断 Job 是否失败
func (j *Job) IsFailed() bool {
	return j.Failed > 0 && j.Active == 0 && !j.Complete
}

// ============================================================
// CronJob 模型
// ============================================================

// CronJob K8s CronJob 资源模型
//
// CronJob 按照预定的时间表创建 Job。
type CronJob struct {
	CommonMeta

	// 调度配置
	Schedule          string `json:"schedule"`                        // Cron 表达式
	Suspend           bool   `json:"suspend"`                         // 是否暂停
	ConcurrencyPolicy string `json:"concurrency_policy,omitempty"`    // Allow, Forbid, Replace

	// 历史保留
	SuccessfulJobsHistoryLimit *int32 `json:"successful_jobs_history_limit,omitempty"`
	FailedJobsHistoryLimit     *int32 `json:"failed_jobs_history_limit,omitempty"`

	// Pod 模板
	Template PodTemplate `json:"template"` // 从 JobTemplate.Spec.Template 提取

	// 状态
	ActiveJobs int32 `json:"active_jobs"` // 当前活跃的 Job 数

	// 时间
	LastScheduleTime   *time.Time `json:"last_schedule_time,omitempty"`   // 上次调度时间
	LastSuccessfulTime *time.Time `json:"last_successful_time,omitempty"` // 上次成功时间
}

// IsSuspended 判断是否暂停
func (c *CronJob) IsSuspended() bool {
	return c.Suspend
}

// HasActiveJobs 判断是否有活跃的 Job
func (c *CronJob) HasActiveJobs() bool {
	return c.ActiveJobs > 0
}

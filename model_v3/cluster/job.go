package cluster

import (
	"time"

	model_v3 "AtlHyper/model_v3"
)

// Job K8s Job 资源模型
type Job struct {
	model_v3.CommonMeta
	Active       int32               `json:"active"`
	Succeeded    int32               `json:"succeeded"`
	Failed       int32               `json:"failed"`
	Complete     bool                `json:"complete"`
	Completions  *int32              `json:"completions,omitempty"`
	Parallelism  *int32              `json:"parallelism,omitempty"`
	BackoffLimit *int32              `json:"backoffLimit,omitempty"`
	Template     PodTemplate         `json:"template"`
	Conditions   []WorkloadCondition `json:"conditions,omitempty"`
	StartTime    *time.Time          `json:"startTime,omitempty"`
	FinishTime   *time.Time          `json:"finishTime,omitempty"`
}

func (j *Job) IsRunning() bool  { return j.Active > 0 }
func (j *Job) IsComplete() bool { return j.Complete }
func (j *Job) IsFailed() bool   { return j.Failed > 0 && j.Active == 0 && !j.Complete }

// CronJob K8s CronJob 资源模型
type CronJob struct {
	model_v3.CommonMeta
	Schedule                   string      `json:"schedule"`
	Suspend                    bool        `json:"suspend"`
	ConcurrencyPolicy          string      `json:"concurrencyPolicy,omitempty"`
	SuccessfulJobsHistoryLimit *int32      `json:"successfulJobsHistoryLimit,omitempty"`
	FailedJobsHistoryLimit     *int32      `json:"failedJobsHistoryLimit,omitempty"`
	Template                   PodTemplate `json:"template"`
	ActiveJobs                 int32       `json:"activeJobs"`
	LastScheduleTime           *time.Time  `json:"lastScheduleTime,omitempty"`
	LastSuccessfulTime         *time.Time  `json:"lastSuccessfulTime,omitempty"`
}

func (c *CronJob) IsSuspended() bool   { return c.Suspend }
func (c *CronJob) HasActiveJobs() bool { return c.ActiveJobs > 0 }

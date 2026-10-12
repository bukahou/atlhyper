// atlhyper_master_v2/aiops/enricher/analysis_jobs.go
// 深度分析状态跟踪（内存）
//
// 深度分析是「后台执行、前端轮询」：此前前端只能轮询报告列表，失败无记录、超过 60s 即放弃，
// 用户看到的只有「没反应」。这里按事件记录最近一次分析的状态与错误码，供状态接口查询。
// Master 重启即丢失 —— 此时进行中的分析本身也已中断，前端据「查不到」提示重试。
package enricher

import (
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
)

// 分析状态
const (
	JobRunning   = "running"
	JobSucceeded = "succeeded"
	JobFailed    = "failed"
)

// jobRetention 已结束记录的保留期（足够前端取到结果）
const jobRetention = 30 * time.Minute

// AnalysisJob 某事件最近一次深度分析的状态
type AnalysisJob struct {
	IncidentID string     `json:"incidentId"`
	State      string     `json:"state"`               // running / succeeded / failed
	ErrorCode  string     `json:"errorCode,omitempty"` // 失败时：ai.Code*
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type analysisJobs struct {
	mu   sync.Mutex
	jobs map[string]*AnalysisJob
	now  func() time.Time
}

func newAnalysisJobs(now func() time.Time) *analysisJobs {
	return &analysisJobs{jobs: map[string]*AnalysisJob{}, now: now}
}

// start 登记开始；同一事件已在进行中则返回 false（不重复启动）
func (a *analysisJobs) start(incidentID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pruneLocked()
	if j, ok := a.jobs[incidentID]; ok && j.State == JobRunning {
		return false
	}
	a.jobs[incidentID] = &AnalysisJob{IncidentID: incidentID, State: JobRunning, StartedAt: a.now()}
	return true
}

// finish 登记结束：err 为 nil → succeeded，否则 failed + 错误码
func (a *analysisJobs) finish(incidentID string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	j, ok := a.jobs[incidentID]
	if !ok {
		return
	}
	t := a.now()
	j.FinishedAt = &t
	if err != nil {
		j.State, j.ErrorCode = JobFailed, ai.ErrorCode(err)
		return
	}
	j.State, j.ErrorCode = JobSucceeded, ""
}

// get 返回状态副本
func (a *analysisJobs) get(incidentID string) (AnalysisJob, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	j, ok := a.jobs[incidentID]
	if !ok {
		return AnalysisJob{}, false
	}
	return *j, true
}

func (a *analysisJobs) pruneLocked() {
	cutoff := a.now().Add(-jobRetention)
	for id, j := range a.jobs {
		if j.FinishedAt != nil && j.FinishedAt.Before(cutoff) {
			delete(a.jobs, id)
		}
	}
}

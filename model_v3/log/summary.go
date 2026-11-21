// model_v3/log/summary.go
// 日志统计摘要类型
package log

import "time"

// Summary 日志统计摘要（5 分钟窗口）
type Summary struct {
	TotalEntries   int64            `json:"totalEntries"`
	SeverityCounts map[string]int64 `json:"severityCounts"` // {"ERROR": 10, "WARN": 50, ...}
	TopServices    []ServiceCount   `json:"topServices"`    // Top 10 服务按日志量排序
	LatestAt       time.Time        `json:"latestAt"`       // 最新一条日志时间

	// ServiceErrors 按服务的 ERROR 日志数：时间窗与「什么算 ERROR」同 SeverityCounts["ERROR"]，
	// 按错误数从多到少，只含错误数最多的若干个服务（条数上限由 Agent 的查询定），没有错误的服务不在其中。
	// 旧版 Agent 不带此字段，解出来是 nil
	ServiceErrors []ServiceErrorCount `json:"serviceErrors"`
}

// ServiceCount 服务日志计数
type ServiceCount struct {
	Service string `json:"service"`
	Count   int64  `json:"count"`
}

// ServiceErrorCount 一个服务的 ERROR 日志数。服务按「命名空间 + 服务名」区分，
// 命名空间与 APM 服务同一取法，主控据此对上同一个服务实体
type ServiceErrorCount struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Count     int64  `json:"count"`
}

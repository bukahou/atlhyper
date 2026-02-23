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
}

// ServiceCount 服务日志计数
type ServiceCount struct {
	Service string `json:"service"`
	Count   int64  `json:"count"`
}

package types

import "time"

// 📦 通用结构体：用于统一异常日志事件表示
type LogEvent struct {
	Timestamp  time.Time
	Kind       string // Pod / Node / ...
	Namespace  string
	Name       string
	Node       string // ✅ 表示异常所属节点
	ReasonCode string
	Category   string
	Severity   string
	Message    string
}

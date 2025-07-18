package model

import "time"

// EventLog 表结构
type EventLog struct {
	Category   string
	EventTime  string
	Kind       string
	Message    string
	Name       string
	Namespace  string
	Node       string
	Reason     string
	Severity   string
	Time       string
}

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


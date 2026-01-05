// model/transport/event.go
// 诊断事件模型（Agent ↔ Master）
package transport

import "time"

// LogEvent Agent 上报的原始数据结构（Store 内储存的事件结构）
type LogEvent struct {
	Timestamp  time.Time
	Kind       string // Pod / Node / ...
	Namespace  string
	Name       string
	Node       string // 表示异常所属节点
	ReasonCode string
	Category   string
	Severity   string
	Message    string
}

// EventLog Master 中 DB 落盘的事件结构
type EventLog struct {
	ClusterID string
	Category  string
	EventTime string
	Kind      string
	Message   string
	Name      string
	Namespace string
	Node      string
	Reason    string
	Severity  string
	Time      string
}

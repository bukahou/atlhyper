// model/event/eventlog.go
package model

import "time"

//agent上报的原始数据结构(Store内储存的事件结构)
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

//Master中DB落盘的事件结构
type EventLog struct {
	ClusterID  string
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
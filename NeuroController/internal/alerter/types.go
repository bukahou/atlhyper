package alerter

import "time"

//
// ========= Pod 相关告警状态 =========
//

type PodStatus struct {
	PodName    string    // Pod 名称
	reasonCode string    // 具体 reason 字段（可选扩展："ReadinessProbeFailed"、"NodeLost"）
	Message    string    // 原始异常信息（用于日志或邮件）
	Timestamp  time.Time // 首次出现异常时间
	LastSeen   time.Time // 最后一次收到该异常时间（用于判断是否恢复）
}

type DeploymentHealthState struct {
	Namespace     string
	Name          string
	ExpectedCount int
	UnreadyPods   map[string]PodStatus
	FirstObserved time.Time
	Confirmed     bool
}

//
// ========= Node 相关告警状态 =========未实装
//

type NodeHealthState struct {
	Name          string
	LastSeenTime  time.Time
	FirstNotReady time.Time
	Confirmed     bool
}

//
// ========= Endpoint 相关（可扩展）=========未实装
//

type EndpointState struct {
	Name        string
	Namespace   string
	LastNoReady time.Time
	Confirmed   bool
}

//
// ========= 全局限频记录 =========未实装
//

var LastAlertTime = make(map[string]time.Time) // key 可为 deployment:xxx / node:xxx

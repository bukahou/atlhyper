package model_v2

import "time"

// ============================================================
// Event 模型
// ============================================================

// Event K8s Event 资源模型
//
// Event 记录集群中发生的事件，用于问题诊断和审计。
//
// Web 展示用途：
//   - 事件列表：Type, Reason, Message, InvolvedObject, Age
//   - 资源详情页：关联事件列表
//   - 告警统计：Warning 事件数量和趋势
type Event struct {
	CommonMeta

	// 事件类型
	Type   string `json:"type"`   // Normal, Warning
	Reason string `json:"reason"` // 事件原因码（如 Created, Scheduled, FailedScheduling）

	// 事件内容
	Message string `json:"message"` // 事件详细信息

	// 事件来源
	Source string `json:"source,omitempty"` // 事件来源组件（如 kubelet, scheduler）

	// 关联对象
	InvolvedObject ResourceRef `json:"involved_object"` // 关联的资源对象

	// 计数
	Count int32 `json:"count"` // 事件发生次数

	// 时间
	FirstTimestamp time.Time `json:"first_timestamp"` // 首次发生时间
	LastTimestamp  time.Time `json:"last_timestamp"`  // 最后发生时间
}

// IsWarning 判断是否是 Warning 类型
func (e *Event) IsWarning() bool {
	return e.Type == "Warning"
}

// IsNormal 判断是否是 Normal 类型
func (e *Event) IsNormal() bool {
	return e.Type == "Normal"
}

// IsCritical 判断是否是严重事件
//
// 某些 Warning 事件表示严重问题：
//   - Failed, FailedScheduling: 调度失败
//   - OOMKilled: 内存溢出被杀
//   - BackOff, CrashLoopBackOff: 容器反复崩溃
func (e *Event) IsCritical() bool {
	if e.Type != "Warning" {
		return false
	}

	criticalReasons := []string{
		"Failed",
		"FailedScheduling",
		"FailedMount",
		"FailedAttachVolume",
		"OOMKilled",
		"BackOff",
		"CrashLoopBackOff",
		"Unhealthy",
		"NodeNotReady",
	}

	for _, reason := range criticalReasons {
		if e.Reason == reason {
			return true
		}
	}
	return false
}

// GetSeverity 获取事件严重程度
//
// 返回值：
//   - "critical": 严重事件
//   - "warning": 警告事件
//   - "info": 普通事件
func (e *Event) GetSeverity() string {
	if e.IsCritical() {
		return "critical"
	}
	if e.IsWarning() {
		return "warning"
	}
	return "info"
}

// MatchesResource 判断事件是否关联指定资源
func (e *Event) MatchesResource(kind, namespace, name string) bool {
	return e.InvolvedObject.Kind == kind &&
		e.InvolvedObject.Namespace == namespace &&
		e.InvolvedObject.Name == name
}

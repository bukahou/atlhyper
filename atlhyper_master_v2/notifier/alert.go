// atlhyper_master_v2/notifier/alert.go
// 告警类型定义（保留兼容）
package notifier

// Severity 告警级别
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Source 告警来源
type Source string

const (
	SourceAgentHeartbeat Source = "agent_heartbeat"
	SourceK8sEvent       Source = "k8s_event"
	SourceManual         Source = "manual"
)

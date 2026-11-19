// atlhyper_master_v2/model/event.go
// Event Web API 响应类型（camelCase JSON tag）
package model

// EventLog 事件日志项
type EventLog struct {
	ClusterID string `json:"clusterId"`
	Category  string `json:"category"`
	EventTime string `json:"eventTime"`
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Node      string `json:"node"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity"` // EventSeverity* 之一，前端只按它上色
	Time      string `json:"time"`
}

// 事件严重程度（由后端判定，见 convert.eventSeverity）
const (
	EventSeverityCritical = "critical" // Warning 里会直接导致故障的原因（OOM、CrashLoop、调度失败……）
	EventSeverityWarning  = "warning"
	EventSeverityError    = "error"
	EventSeverityInfo     = "info"
)

// EventCards 事件统计卡片（随事件列表一起返回，前端不自己数）
type EventCards struct {
	TotalAlerts     int `json:"totalAlerts"` // critical + warning + error
	TotalEvents     int `json:"totalEvents"`
	Critical        int `json:"critical"`
	Warning         int `json:"warning"` // 不含 critical
	Info            int `json:"info"`
	Error           int `json:"error"`
	CategoriesCount int `json:"categoriesCount"`
	KindsCount      int `json:"kindsCount"`
}

// EventListResponse 事件列表响应
type EventListResponse struct {
	Events []EventLog `json:"events"`
	Cards  EventCards `json:"cards"` // 本次返回的事件的统计
	Total  int        `json:"total"`
	Source string     `json:"source"`
	Limit  int        `json:"limit,omitempty"`
	Offset int        `json:"offset,omitempty"`
}

// atlhyper_master_v2/notifier/types.go
// 通知系统数据模型
package notifier

import (
	"fmt"
	"time"
)

// ============================================================
// 告警模型
// ============================================================

// Alert 告警信息
type Alert struct {
	ID        string            // 唯一标识 (UUID)
	Title     string            // 告警标题
	Message   string            // 详细消息
	Severity  string            // critical / warning / info
	Source    string            // agent_heartbeat / k8s_event / manual
	ClusterID string            // 集群 ID
	Resource  string            // 资源标识 (Pod/default/nginx-xxx)
	Reason    string            // 原因代码 (CrashLoopBackOff)
	Fields    map[string]string // 扩展字段
	Timestamp time.Time         // 发生时间
}

// DedupKey 生成去重 Key
func (a *Alert) DedupKey() string {
	return fmt.Sprintf("%s|%s|%s|%s",
		a.ClusterID, a.Resource, a.Reason, a.Severity)
}

// AlertSummary 聚合后的告警摘要
type AlertSummary struct {
	Total       int            // 告警总数
	BySeverity  map[string]int // 按级别统计: critical/warning/info -> count
	Clusters    []string       // 涉及集群 (去重)
	Namespaces  []string       // 涉及命名空间 (去重)
	Alerts      []*Alert       // 告警列表 (最多 MaxAlertsInMsg 条)
	HasMore     bool           // 是否有更多
	MoreCount   int            // 省略条数
	GeneratedAt time.Time      // 生成时间
}

// ============================================================
// 消息模型
// ============================================================

// Message 通知消息
type Message struct {
	Title    string            // 标题
	Content  string            // 内容
	Severity string            // 严重程度: info / warning / critical
	Fields   map[string]string // 额外字段
}

// Result 发送结果
type Result struct {
	Success bool
	Error   string
}

// ============================================================
// 常量定义
// ============================================================

// 严重级别
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// 告警来源
const (
	SourceAgentHeartbeat = "agent_heartbeat"
	SourceK8sEvent       = "k8s_event"
	SourceManual         = "manual"
)

// atlhyper_master_v2/notifier/alert/types.go
// 统一告警对象：事件、Agent 心跳、K8s Warning 事件都先变成 Alert，再进入同一条通知管道
package alert

import "time"

// Source 告警来源
type Source string

const (
	SourceIncident  Source = "incident"  // AIOps 事件
	SourceHeartbeat Source = "heartbeat" // Agent 心跳
	SourceK8sEvent  Source = "k8s_event" // K8s Warning 事件（旧，默认关）
)

// Status 告警状态
type Status string

const (
	StatusFiring   Status = "firing"
	StatusResolved Status = "resolved"
)

// Kind 告警最近一次变化的种类。
//
// 前五种是告警自身的变化；KindRepeat 只出现在送达记录里（对持续中的告警再提醒一次），不会写进告警。
type Kind string

const (
	KindFiring    Kind = "firing"    // 触发
	KindEscalated Kind = "escalated" // 严重度升高
	KindRecurred  Kind = "recurred"  // 恢复后复发
	KindFlapping  Kind = "flapping"  // 警告级反复到阈值
	KindResolved  Kind = "resolved"  // 恢复
	KindRepeat    Kind = "repeat"    // 重复提醒
)

// Opens 这种变化是否代表「告警开始了」。
// 开始类的通知还没发出去时，后到的升级不改写它：收件人看到的第一条必须是「开始」
func (k Kind) Opens() bool {
	return k == KindFiring || k == KindRecurred || k == KindFlapping
}

// 标签键：供路由规则与静默匹配
const (
	LabelNamespace   = "namespace"
	LabelSubject     = "subject"     // 主体键，如 shop/deployment/api
	LabelSubjectType = "subjectType" // deployment / statefulset / node ...
)

// 详情键：供消息模板与接口展示
const (
	DetailIncidentID     = "incidentId"
	DetailRootCause      = "rootCause"      // 领头实体
	DetailMemberCount    = "memberCount"    // 本轮有风险的成员数
	DetailMembers        = "members"        // 成员实体键，逗号分隔（过多时截断）
	DetailPeakRisk       = "peakRisk"       // 峰值风险，0–100
	DetailRecurrence     = "recurrence"     // 复发次数
	DetailOccurrences24h = "occurrences24h" // 近 24 小时发生次数
	DetailLastHeartbeat  = "lastHeartbeat"  // 最后一次心跳（RFC 3339）
	DetailEventKind      = "eventKind"      // K8s 事件涉及的资源类型
	DetailEventName      = "eventName"      // K8s 事件涉及的资源名
	DetailEventReason    = "eventReason"    // K8s 事件的 Reason
	DetailEventCount     = "eventCount"     // K8s 事件累计次数
)

// Alert 统一告警对象
type Alert struct {
	Fingerprint string // 稳定标识：同一个问题始终是同一个值
	Source      Source // incident / heartbeat / k8s_event
	ClusterID   string
	Severity    Severity // low / medium / high / critical（与 AIOps 的四档一致）
	Status      Status   // firing / resolved
	Kind        Kind     // 最近一次变化：firing / escalated / recurred / flapping / resolved
	Title       string
	Summary     string
	Labels      map[string]string // namespace、subject、subjectType，供规则与静默匹配
	Details     map[string]string // 根因实体、成员、峰值风险、复发次数、事件号等，供模板使用
	StartsAt    time.Time
	EndsAt      *time.Time
	UpdatedAt   time.Time

	// Quiet 这次变化不发通知。只在提交给管道的那一刻有意义，不落库。
	// 用于「结束了但并不是真的恢复」：事件失联关闭、K8s 事件长时间没有再出现
	Quiet bool
}

// Label 取标签值；没有时返回空串
func (a *Alert) Label(key string) string { return a.Labels[key] }

// Detail 取详情值；没有时返回空串
func (a *Alert) Detail(key string) string { return a.Details[key] }

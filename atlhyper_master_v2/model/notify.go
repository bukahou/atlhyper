// atlhyper_master_v2/model/notify.go
// 通知设置页的响应模型：策略、当前告警、送达记录、接收人、路由规则、静默
//
// 时间一律是 RFC 3339 字符串（UTC），「没有」时为空串；列表恒为数组（不为 null）
package model

// NotifyPolicy 全局通知策略
type NotifyPolicy struct {
	IncidentAlertsEnabled bool    `json:"incidentAlertsEnabled"`
	MinSeverity           string  `json:"minSeverity"`           // 低于此严重度的告警不通知（默认策略下）
	NotifyResolved        bool    `json:"notifyResolved"`        // 是否发恢复通知
	WarningFlapThreshold  int     `json:"warningFlapThreshold"`  // 警告级事件 24 小时内发生几次才通知；0 = 不通知
	GroupWaitSeconds      int     `json:"groupWaitSeconds"`      // 分组等待
	GroupIntervalSeconds  int     `json:"groupIntervalSeconds"`  // 分组间隔
	RepeatIntervalSeconds int     `json:"repeatIntervalSeconds"` // 重复提醒间隔
	K8sEventAlertsEnabled bool    `json:"k8sEventAlertsEnabled"` // 旧的逐条 K8s 事件告警
	DefaultReceiverIDs    []int64 `json:"defaultReceiverIds"`    // 没有规则命中时发给谁；空 = 全部已启用的接收人
}

// NotifyAlert 一条告警
type NotifyAlert struct {
	Fingerprint    string `json:"fingerprint"`
	Source         string `json:"source"` // incident / heartbeat / k8s_event
	ClusterID      string `json:"clusterId"`
	Severity       string `json:"severity"` // low / medium / high / critical
	Status         string `json:"status"`   // firing / resolved
	Kind           string `json:"kind"`     // 最近一次变化：firing / escalated / recurred / flapping / resolved
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Subject        string `json:"subject"`        // 主体键；心跳告警为空
	IncidentID     string `json:"incidentId"`     // 对应的事件号；不是事件告警时为空
	StartsAt       string `json:"startsAt"`       // RFC 3339
	EndsAt         string `json:"endsAt"`         // 持续中为空串
	LastNotifiedAt string `json:"lastNotifiedAt"` // 还没通知过为空串
	NotifyCount    int    `json:"notifyCount"`
	SuppressedBy   string `json:"suppressedBy"` // 被什么抑制（cluster-offline）；空 = 没有
	SilencedBy     int64  `json:"silencedBy"`   // 命中的静默 id；0 = 没有
}

// NotifyDelivery 一条送达记录
type NotifyDelivery struct {
	ID           int64  `json:"id"`
	BatchID      string `json:"batchId"`
	Fingerprint  string `json:"fingerprint"`
	AlertTitle   string `json:"alertTitle"`
	ReceiverID   int64  `json:"receiverId"`   // 没有接收人时为 0
	ReceiverName string `json:"receiverName"` // 没有接收人时为空串
	ReceiverType string `json:"receiverType"` // 没有接收人时为空串
	Kind         string `json:"kind"`         // firing / escalated / recurred / flapping / resolved / repeat
	Status       string `json:"status"`       // pending / success / failed / abandoned / no-receiver
	Attempts     int    `json:"attempts"`
	LastError    string `json:"lastError"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// NotifyReceiver 一个接收人。
//
// ⛔ 凭据（Webhook 地址、SMTP 密码）不在其中：只说明是否已配置
type NotifyReceiver struct {
	ID        int64              `json:"id"`
	Name      string             `json:"name"`
	Type      string             `json:"type"` // slack / email
	Enabled   bool               `json:"enabled"`
	Slack     *NotifySlackConfig `json:"slack,omitempty"`
	Email     *NotifyEmailConfig `json:"email,omitempty"`
	CreatedAt string             `json:"createdAt"`
	UpdatedAt string             `json:"updatedAt"`
}

// NotifySlackConfig Slack 接收人的可见配置
type NotifySlackConfig struct {
	WebhookConfigured bool `json:"webhookConfigured"`
}

// NotifyEmailConfig 邮件接收人的可见配置
type NotifyEmailConfig struct {
	SMTPHost           string   `json:"smtpHost"`
	SMTPPort           int      `json:"smtpPort"`
	SMTPUser           string   `json:"smtpUser"`
	SMTPTLS            bool     `json:"smtpTLS"`
	FromAddress        string   `json:"fromAddress"`
	ToAddresses        []string `json:"toAddresses"`
	PasswordConfigured bool     `json:"passwordConfigured"`
}

// NotifyRule 一条路由规则
type NotifyRule struct {
	ID                    int64    `json:"id"`
	Name                  string   `json:"name"`
	Position              int      `json:"position"`
	Enabled               bool     `json:"enabled"`
	ClusterIDs            []string `json:"clusterIds"` // 空 = 全部集群
	Sources               []string `json:"sources"`    // 空 = 全部来源
	MinSeverity           string   `json:"minSeverity"`
	ReceiverIDs           []int64  `json:"receiverIds"`           // 空 = 命中后不发给任何人
	GroupWaitSeconds      *int     `json:"groupWaitSeconds"`      // null = 用策略里的全局值
	GroupIntervalSeconds  *int     `json:"groupIntervalSeconds"`  // null = 用策略里的全局值
	RepeatIntervalSeconds *int     `json:"repeatIntervalSeconds"` // null = 用策略里的全局值
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`
}

// NotifyRuleMatchResult 试匹配的结果
type NotifyRuleMatchResult struct {
	MatchedRule      *NotifyRuleRef      `json:"matchedRule"`      // null = 走默认策略
	Receivers        []NotifyReceiverRef `json:"receivers"`        // 规则（或默认策略）指向的接收人，含已停用的
	BelowMinSeverity bool                `json:"belowMinSeverity"` // 默认策略下严重度不到 minSeverity，不会发送
}

// NotifyRuleRef 规则的简要信息
type NotifyRuleRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// NotifyReceiverRef 接收人的简要信息
type NotifyReceiverRef struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

// NotifySilence 一条静默
type NotifySilence struct {
	ID        int64  `json:"id"`
	ClusterID string `json:"clusterId"` // 空 = 全部集群
	Namespace string `json:"namespace"` // 空 = 全部命名空间
	Subject   string `json:"subject"`   // 空 = 全部主体
	StartsAt  string `json:"startsAt"`
	EndsAt    string `json:"endsAt"`
	Comment   string `json:"comment"`
	CreatedBy string `json:"createdBy"`
	CreatedAt string `json:"createdAt"`
	State     string `json:"state"` // pending（未开始）/ active / expired
}

// NotifyReceiverInUseError 要删除的接收人仍被引用
type NotifyReceiverInUseError struct {
	References []string // 引用它的对象的说法，如「路由规则「生产高严重度」」「默认策略」
}

func (e *NotifyReceiverInUseError) Error() string {
	text := "接收人仍被引用，不能删除："
	for index, reference := range e.References {
		if index > 0 {
			text += "、"
		}
		text += reference
	}
	return text
}

// atlhyper_master_v2/model/notify_request.go
// 通知设置页的请求模型与列表查询条件
package model

// UpdateNotifyPolicyRequest 保存通知策略。
//
// 整体替换：九个字段都必须给出。用指针是为了分辨「没传」与「传了 false / 0」——
// 少传一个开关不应被当成把它关掉
type UpdateNotifyPolicyRequest struct {
	IncidentAlertsEnabled *bool    `json:"incidentAlertsEnabled"`
	MinSeverity           *string  `json:"minSeverity"`
	NotifyResolved        *bool    `json:"notifyResolved"`
	WarningFlapThreshold  *int     `json:"warningFlapThreshold"`
	GroupWaitSeconds      *int     `json:"groupWaitSeconds"`
	GroupIntervalSeconds  *int     `json:"groupIntervalSeconds"`
	RepeatIntervalSeconds *int     `json:"repeatIntervalSeconds"`
	K8sEventAlertsEnabled *bool    `json:"k8sEventAlertsEnabled"`
	DefaultReceiverIDs    *[]int64 `json:"defaultReceiverIds"`
}

// NotifyAlertQuery 告警列表的查询条件
type NotifyAlertQuery struct {
	Status    string // firing / resolved；空 = 全部
	ClusterID string // 空 = 全部
	Limit     int
	Offset    int
}

// NotifyDeliveryQuery 送达记录的查询条件
type NotifyDeliveryQuery struct {
	Fingerprint string // 空 = 全部
	Status      string // 空 = 不限
	Limit       int
	Offset      int
}

// NotifyReceiverRequest 新建 / 修改接收人。
//
// 凭据（webhookUrl）在修改时留空 = 保持原值；修改时不带 slack / email 段 = 整段配置保持原值
type NotifyReceiverRequest struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`    // slack / email
	Enabled *bool             `json:"enabled"` // 没传：新建时为启用，修改时保持原值
	Slack   *NotifySlackInput `json:"slack"`
	Email   *NotifyEmailInput `json:"email"`
}

// NotifySlackInput Slack 接收人的配置
type NotifySlackInput struct {
	WebhookURL string `json:"webhookUrl"`
}

// NotifyEmailInput 邮件接收人的配置：只有收件人
type NotifyEmailInput struct {
	ToAddresses []string `json:"toAddresses"`
}

// UpdateNotifySMTPRequest 保存全局发件设置（整体替换）。password 留空 = 保持原值；第一次保存必须填写
type UpdateNotifySMTPRequest struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	Password    string `json:"password"`
	TLS         bool   `json:"tls"`
	FromAddress string `json:"fromAddress"` // 空 = 使用 user
}

// NotifyRuleRequest 新建 / 修改路由规则（不带 id 与 position）
type NotifyRuleRequest struct {
	Name                  string   `json:"name"`
	Enabled               *bool    `json:"enabled"` // 没传：新建时为启用，修改时保持原值
	ClusterIDs            []string `json:"clusterIds"`
	Sources               []string `json:"sources"`
	MinSeverity           string   `json:"minSeverity"` // 空 = low
	ReceiverIDs           []int64  `json:"receiverIds"`
	GroupWaitSeconds      *int     `json:"groupWaitSeconds"`
	GroupIntervalSeconds  *int     `json:"groupIntervalSeconds"`
	RepeatIntervalSeconds *int     `json:"repeatIntervalSeconds"`
}

// NotifyRuleOrderRequest 规则重排：ids 必须恰好是全部规则的一个排列
type NotifyRuleOrderRequest struct {
	IDs []int64 `json:"ids"`
}

// NotifyRuleMatchRequest 试匹配：这样一条告警会发给谁
type NotifyRuleMatchRequest struct {
	ClusterID string `json:"clusterId"`
	Source    string `json:"source"`
	Severity  string `json:"severity"`
}

// CreateNotifySilenceRequest 新建静默
type CreateNotifySilenceRequest struct {
	ClusterID string `json:"clusterId"`
	Namespace string `json:"namespace"`
	Subject   string `json:"subject"`
	StartsAt  string `json:"startsAt"` // RFC 3339；空 = 现在
	EndsAt    string `json:"endsAt"`   // RFC 3339
	Comment   string `json:"comment"`
}

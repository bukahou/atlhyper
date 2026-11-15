// atlhyper_master_v2/database/types_notify.go
// 通知模块的持久化模型：策略、告警、送达记录、接收人、路由规则、静默
package database

import "time"

// NotifyPolicy 全局通知策略（单行）
type NotifyPolicy struct {
	IncidentAlertsEnabled bool   // 事件是否产生告警
	MinSeverity           string // 默认策略下，低于此严重度的告警不通知
	NotifyResolved        bool   // 是否发恢复通知
	WarningFlapThreshold  int    // 警告级事件 24 小时内发生几次才通知；0 = 不通知
	GroupWaitSeconds      int
	GroupIntervalSeconds  int
	RepeatIntervalSeconds int
	K8sEventAlertsEnabled bool    // 旧的逐条 K8s 事件告警
	DefaultReceiverIDs    []int64 // 没有规则命中时发给谁；空 = 全部已启用的接收人
	UpdatedAt             time.Time
}

// DefaultNotifyPolicy 策略的出厂值；库里还没有策略行时即为生效值
func DefaultNotifyPolicy() *NotifyPolicy {
	return &NotifyPolicy{
		IncidentAlertsEnabled: true,
		MinSeverity:           "low",
		NotifyResolved:        true,
		WarningFlapThreshold:  3,
		GroupWaitSeconds:      30,
		GroupIntervalSeconds:  300,
		RepeatIntervalSeconds: 14400,
		K8sEventAlertsEnabled: false,
		DefaultReceiverIDs:    []int64{},
	}
}

// NotifyAlert 告警表的一行
type NotifyAlert struct {
	Fingerprint string
	Source      string
	ClusterID   string
	Severity    string
	Status      string // firing / resolved
	Kind        string // 最近一次变化
	Title       string
	Summary     string
	Labels      map[string]string
	Details     map[string]string
	StartsAt    time.Time
	EndsAt      *time.Time
	UpdatedAt   time.Time

	LastNotifiedAt *time.Time // 最近一次进入发送（含「没有接收人」）的时刻
	// LastNotifiedKind / LastNotifiedSeverity 本次持续期间最近一次已通知的变化与当时的严重度；
	// 空 = 这次持续还没有通知过任何人
	LastNotifiedKind     string
	LastNotifiedSeverity string
	NotifyCount          int    // 累计发出的通知数（不含「没有接收人」）
	SuppressedBy         string // 被什么抑制，如 cluster-offline；空 = 没有
	SilencedBy           int64  // 命中的静默 id；0 = 没有
}

// NotifyAlertQueryOpts 告警列表的查询条件
type NotifyAlertQueryOpts struct {
	Status    string // 空 = 全部
	ClusterID string // 空 = 全部
	Limit     int
	Offset    int
}

// 送达记录的状态
const (
	NotifyDeliveryPending    = "pending"     // 已建批次，尚未发出
	NotifyDeliverySuccess    = "success"     // 发送成功
	NotifyDeliveryFailed     = "failed"      // 发送失败，等待重试
	NotifyDeliveryAbandoned  = "abandoned"   // 重试用尽，或接收人已不可用
	NotifyDeliveryNoReceiver = "no-receiver" // 没有任何可用接收人
)

// NotifyDelivery 送达记录：一条告警发给一个接收人的一次通知
type NotifyDelivery struct {
	ID            int64
	BatchID       string // 同一条消息里的告警共用一个批次号
	Fingerprint   string
	AlertTitle    string
	ReceiverID    int64 // 0 = 没有接收人
	ReceiverName  string
	ReceiverType  string
	Kind          string // firing / escalated / recurred / flapping / resolved / repeat
	Status        string
	Attempts      int
	LastError     string
	NextAttemptAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NotifyDeliveryResult 一个批次的一次发送结果
type NotifyDeliveryResult struct {
	Status        string
	Attempts      int
	LastError     string
	NextAttemptAt *time.Time
	UpdatedAt     time.Time
}

// NotifyDeliveryQueryOpts 送达记录的查询条件
type NotifyDeliveryQueryOpts struct {
	Fingerprint string // 空 = 全部
	Status      string // 空 = 全部
	Limit       int
	Offset      int
}

// 接收人类型
const (
	NotifyReceiverSlack = "slack"
	NotifyReceiverEmail = "email"
)

// NotifyReceiver 接收人。
//
// Slack / Email 里有凭据（Webhook 地址、SMTP 密码）：整份配置加密后落库，仓库读出时解密。
// 两者按 Type 只有一个非 nil；配置解不开时两者都为 nil
type NotifyReceiver struct {
	ID        int64
	Name      string
	Type      string // slack / email
	Enabled   bool
	Slack     *SlackConfig
	Email     *EmailConfig
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SlackConfig Slack 配置
type SlackConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost     string   `json:"smtpHost"`
	SMTPPort     int      `json:"smtpPort"`
	SMTPUser     string   `json:"smtpUser"`
	SMTPPassword string   `json:"smtpPassword"`
	SMTPTLS      bool     `json:"smtpTLS"`
	FromAddress  string   `json:"fromAddress"`
	ToAddresses  []string `json:"toAddresses"`
}

// NotifyRule 路由规则
type NotifyRule struct {
	ID          int64
	Name        string
	Position    int // 匹配顺序，从小到大
	Enabled     bool
	ClusterIDs  []string // 空 = 全部集群
	Sources     []string // 空 = 全部来源
	MinSeverity string
	ReceiverIDs []int64
	// 三个时间参数为 nil 时用策略里的全局值
	GroupWaitSeconds      *int
	GroupIntervalSeconds  *int
	RepeatIntervalSeconds *int
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// NotifySilence 静默
type NotifySilence struct {
	ID        int64
	ClusterID string // 空 = 全部集群
	Namespace string // 空 = 全部命名空间
	Subject   string // 空 = 全部主体；填了则精确匹配主体键
	StartsAt  time.Time
	EndsAt    time.Time
	Comment   string
	CreatedBy string
	CreatedAt time.Time
}

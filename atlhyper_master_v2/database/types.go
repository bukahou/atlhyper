// atlhyper_master_v2/database/types.go
// Database 模型定义
// 包含: 所有数据库模型 struct
package database

import (
	"time"
)

// ==================== 模型定义 ====================

// AuditLog 审计日志
type AuditLog struct {
	ID           int64
	Timestamp    time.Time
	UserID       int64
	Username     string
	Role         int
	Source       string // web / api / ai
	Action       string
	Resource     string
	Method       string
	RequestBody  string
	StatusCode   int
	Success      bool
	ErrorMessage string
	IP           string
	UserAgent    string
	DurationMs   int64
}

// AuditQueryOpts 审计查询选项
type AuditQueryOpts struct {
	UserID int64
	Source string
	Action string
	Since  time.Time
	Until  time.Time
	Limit  int
	Offset int
}

// User 用户
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	DisplayName  string
	Email        string
	Role         int // 1=Viewer, 2=Operator, 3=Admin（数值越大权限越高）
	Status       int // 1=Active, 0=Disabled
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
	LastLoginIP  string
}

// ClusterEvent 集群事件（只存 Warning）
// DedupKey = MD5(ClusterID + InvolvedKind + InvolvedNamespace + InvolvedName + Reason)
type ClusterEvent struct {
	ID                int64
	DedupKey          string // 业务去重键
	ClusterID         string
	Namespace         string
	Name              string
	Type              string // Warning（只存 Warning）
	Reason            string
	Message           string
	SourceComponent   string
	SourceHost        string
	InvolvedKind      string
	InvolvedName      string
	InvolvedNamespace string
	FirstTimestamp    time.Time
	LastTimestamp     time.Time
	Count             int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EventQueryOpts 查询选项
type EventQueryOpts struct {
	Type   string    // 过滤类型: Normal / Warning
	Reason string    // 过滤原因
	Since  time.Time // 起始时间
	Until  time.Time // 结束时间
	Limit  int       // 限制条数
	Offset int       // 偏移量
}

// HourlyEventCount 每小时事件统计
type HourlyEventCount struct {
	Hour         string // 格式: 2006-01-02T15
	WarningCount int
	NormalCount  int
}

// HourlyKindCount 每小时按资源类型统计
type HourlyKindCount struct {
	Hour  string // 格式: 2006-01-02T15
	Kind  string // 资源类型: Pod, Node, Deployment 等
	Count int
}

// NotifyChannel 通知渠道
type NotifyChannel struct {
	ID        int64
	Type      string // slack / email（UNIQUE，一个类型一条记录）
	Name      string // 显示名称
	Enabled   bool   // 是否启用（默认 false）
	Config    string // JSON 配置
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SlackConfig Slack 配置
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

// EmailConfig Email 配置
type EmailConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	SMTPUser     string   `json:"smtp_user"`
	SMTPPassword string   `json:"smtp_password"`
	SMTPTLS      bool     `json:"smtp_tls"`
	FromAddress  string   `json:"from_address"`
	ToAddresses  []string `json:"to_addresses"`
}

// Cluster 集群信息
type Cluster struct {
	ID          int64
	ClusterUID  string // 集群 UID（来自 kube-system）
	Name        string // 显示名称
	Description string
	Environment string // prod / staging / dev
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CommandHistory 指令历史
type CommandHistory struct {
	ID              int64
	CommandID       string // 指令唯一 ID
	ClusterID       string
	Source          string // web / ai
	UserID          int64
	Action          string // scale / restart / delete_pod
	TargetKind      string
	TargetNamespace string
	TargetName      string
	Params          string // JSON
	Status          string // pending / running / success / failed / timeout
	Result          string // JSON
	ErrorMessage    string
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	DurationMs      int64
}

// CommandQueryOpts 命令查询选项
type CommandQueryOpts struct {
	ClusterID string // 集群 ID
	Source    string // web / ai
	Status    string // pending / running / success / failed / timeout
	Action    string // restart / scale / delete_pod / cordon / uncordon
	Search    string // 模糊搜索目标名称
	Limit     int
	Offset    int
}

// Setting 系统设置
type Setting struct {
	Key         string
	Value       string // JSON
	Description string
	UpdatedAt   time.Time
	UpdatedBy   int64
}

// AIConversation AI 对话
type AIConversation struct {
	ID           int64
	UserID       int64
	ClusterID    string
	Title        string
	MessageCount int
	// 累计统计
	TotalInputTokens  int64 // 累计输入 Token
	TotalOutputTokens int64 // 累计输出 Token
	TotalToolCalls    int   // 累计指令数
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// AIMessage AI 消息
type AIMessage struct {
	ID             int64
	ConversationID int64
	Role           string // user / assistant / tool
	Content        string
	ToolCalls      string // JSON: [{id, name, params, result}]
	CreatedAt      time.Time
}

// AIProvider AI 提供商配置
type AIProvider struct {
	ID          int64
	Name        string // 显示名称 (例: "Gemini本番", "OpenAI予備")
	Provider    string // gemini / openai / anthropic
	APIKey      string
	Model       string
	Description string // 说明・备注

	// 使用统计
	TotalRequests int64
	TotalTokens   int64
	TotalCost     float64
	LastUsedAt    *time.Time
	LastError     string
	LastErrorAt   *time.Time

	// 状态
	Status          string // unknown / healthy / quota_exceeded / auth_error / error
	StatusCheckedAt *time.Time

	// 审计
	CreatedAt time.Time
	CreatedBy int64
	UpdatedAt time.Time
	UpdatedBy int64
	DeletedAt *time.Time // 软删除
}

// AIActiveConfig 当前使用中的 AI 配置
type AIActiveConfig struct {
	ID          int64
	Enabled     bool   // AI 功能总开关
	ProviderID  *int64 // 当前使用的 AIProvider ID (NULL = 未设置)
	ToolTimeout int    // Tool 执行超时(秒)
	UpdatedAt   time.Time
	UpdatedBy   int64
}

// AIProviderModel 提供商支持的模型列表
type AIProviderModel struct {
	ID          int64
	Provider    string // gemini / openai / anthropic
	Model       string // 模型ID (例: gemini-2.0-flash)
	DisplayName string // 表示名 (例: Gemini 2.0 Flash)
	IsDefault   bool   // 是否为该提供商的默认模型
	SortOrder   int    // 显示顺序
	CreatedAt   time.Time
}

// ==================== SLO 模型定义 ====================

// SLOTarget SLO 目标配置
type SLOTarget struct {
	ID                 int64
	ClusterID          string
	Host               string
	IngressName        string
	IngressClass       string
	Namespace          string
	TLS                bool
	TimeRange          string // "1d", "7d", "30d"
	AvailabilityTarget float64
	P95LatencyTarget   int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SLORouteMapping IngressRoute 到域名/路径的映射
// Agent 采集 Traefik IngressRoute CRD 后上报，用于将 service 维度的指标转换为 domain/path 维度
type SLORouteMapping struct {
	ID          int64
	ClusterID   string
	Domain      string
	PathPrefix  string
	IngressName string
	Namespace   string
	TLS         bool
	ServiceKey  string // Traefik service 标识，如 namespace-service-port@kubernetes
	ServiceName string
	ServicePort int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ==================== AIOps 模型定义 ====================

// AIOpsBaselineState 基线状态数据库模型
type AIOpsBaselineState struct {
	EntityKey  string
	MetricName string
	EMA        float64
	Variance   float64
	Count      int64
	UpdatedAt  int64
}

// ==================== AIOps Incident 模型定义 ====================

// AIOpsIncident 事件数据库模型
type AIOpsIncident struct {
	ID         string
	ClusterID  string
	State      string
	Severity   string
	RootCause  string
	PeakRisk   float64
	StartedAt  time.Time
	ResolvedAt *time.Time
	DurationS  int64
	Recurrence int
	Summary    string
	CreatedAt  time.Time
}

// AIOpsIncidentEntity 受影响实体数据库模型
type AIOpsIncidentEntity struct {
	IncidentID string
	EntityKey  string
	EntityType string
	RLocal     float64
	RFinal     float64
	Role       string
}

// AIOpsIncidentTimeline 事件时间线数据库模型
type AIOpsIncidentTimeline struct {
	ID         int64
	IncidentID string
	Timestamp  time.Time
	EventType  string
	EntityKey  string
	Detail     string
}

// AIOpsIncidentQueryOpts 事件查询选项
type AIOpsIncidentQueryOpts struct {
	ClusterID string
	State     string
	Severity  string
	From      time.Time
	To        time.Time
	Limit     int
	Offset    int
}

// AIOpsIncidentStatsRaw Repository 层返回的统计原始数据
type AIOpsIncidentStatsRaw struct {
	TotalIncidents  int
	ActiveIncidents int
	MTTR            float64
	RecurringCount  int
	BySeverity      map[string]int
	ByState         map[string]int
}

// AIOpsRootCauseCount 根因统计
type AIOpsRootCauseCount struct {
	EntityKey string
	Count     int
}

// atlhyper_master_v2/database/interfaces.go
// Database 统一接口定义
// 包含: 模型定义 + Repository 接口 + Dialect 接口 + DB 结构体
package database

import (
	"context"
	"database/sql"
	"time"
)

// ==================== DB 结构体 ====================

// DB 数据库统一访问点
// 通过 New() 工厂函数创建，repo.Init() 注入 Repository 实例
type DB struct {
	Audit          AuditRepository
	User           UserRepository
	Event          ClusterEventRepository
	Notify         NotifyChannelRepository
	Cluster        ClusterRepository
	Command        CommandHistoryRepository
	Settings       SettingsRepository
	AIConversation AIConversationRepository
	AIMessage      AIMessageRepository
	AIProvider     AIProviderRepository
	AIActive       AIActiveConfigRepository
	AIModel        AIProviderModelRepository
	SLO SLORepository

	AIOpsBaseline AIOpsBaselineRepository
	AIOpsGraph     AIOpsGraphRepository
	AIOpsIncident  AIOpsIncidentRepository

	Conn *sql.DB // 导出供 repo 包使用
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.Conn.Close()
}

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

// ==================== NodeMetrics 模型定义 ====================


// ==================== Repository 接口 ====================

// AuditRepository 审计日志接口
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, opts AuditQueryOpts) ([]*AuditLog, error)
	Count(ctx context.Context, opts AuditQueryOpts) (int64, error)
}

// UserRepository 用户接口
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	UpdateLastLogin(ctx context.Context, id int64, ip string) error
}

// ClusterEventRepository Event 持久化接口
type ClusterEventRepository interface {
	// 写入
	Upsert(ctx context.Context, event *ClusterEvent) error
	UpsertBatch(ctx context.Context, events []*ClusterEvent) error

	// 查询
	ListByCluster(ctx context.Context, clusterID string, opts EventQueryOpts) ([]*ClusterEvent, error)
	ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*ClusterEvent, error)
	ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*ClusterEvent, error)

	// 告警服务专用
	GetLatestEventID(ctx context.Context) (int64, error)
	GetEventsSince(ctx context.Context, sinceID int64) ([]*ClusterEvent, error)

	// 清理
	DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error)
	DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error)

	// 统计
	CountByCluster(ctx context.Context, clusterID string) (int64, error)
	CountByHour(ctx context.Context, clusterID string, hours int) ([]HourlyEventCount, error)
	CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]HourlyKindCount, error)
}

// NotifyChannelRepository 通知渠道接口
type NotifyChannelRepository interface {
	Create(ctx context.Context, channel *NotifyChannel) error
	Update(ctx context.Context, channel *NotifyChannel) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*NotifyChannel, error)
	GetByType(ctx context.Context, channelType string) (*NotifyChannel, error)
	List(ctx context.Context) ([]*NotifyChannel, error)
	ListEnabled(ctx context.Context) ([]*NotifyChannel, error)
}

// ClusterRepository 集群接口
type ClusterRepository interface {
	Create(ctx context.Context, cluster *Cluster) error
	Update(ctx context.Context, cluster *Cluster) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Cluster, error)
	GetByUID(ctx context.Context, uid string) (*Cluster, error)
	List(ctx context.Context) ([]*Cluster, error)
}

// CommandHistoryRepository 指令历史接口
type CommandHistoryRepository interface {
	Create(ctx context.Context, cmd *CommandHistory) error
	Update(ctx context.Context, cmd *CommandHistory) error
	GetByCommandID(ctx context.Context, cmdID string) (*CommandHistory, error)
	ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*CommandHistory, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*CommandHistory, error)
	// 新增：带筛选条件的列表查询
	List(ctx context.Context, opts CommandQueryOpts) ([]*CommandHistory, error)
	Count(ctx context.Context, opts CommandQueryOpts) (int64, error)
}

// SettingsRepository 设置接口
type SettingsRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetByPrefix(ctx context.Context, prefix string) ([]*Setting, error)
	Set(ctx context.Context, setting *Setting) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]*Setting, error)
}

// AIConversationRepository AI 对话接口
type AIConversationRepository interface {
	Create(ctx context.Context, conv *AIConversation) error
	Update(ctx context.Context, conv *AIConversation) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*AIConversation, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*AIConversation, error)
}

// AIMessageRepository AI 消息接口
type AIMessageRepository interface {
	Create(ctx context.Context, msg *AIMessage) error
	ListByConversation(ctx context.Context, convID int64) ([]*AIMessage, error)
	DeleteByConversation(ctx context.Context, convID int64) error
}

// AIProviderRepository AI 提供商配置接口
type AIProviderRepository interface {
	Create(ctx context.Context, p *AIProvider) error
	Update(ctx context.Context, p *AIProvider) error
	Delete(ctx context.Context, id int64) error // 软删除
	GetByID(ctx context.Context, id int64) (*AIProvider, error)
	List(ctx context.Context) ([]*AIProvider, error) // deleted_at IS NULL

	// 统计更新
	IncrementUsage(ctx context.Context, id int64, requests, tokens int64, cost float64) error
	UpdateStatus(ctx context.Context, id int64, status, errorMsg string) error
}

// AIActiveConfigRepository 当前使用中的 AI 配置接口
type AIActiveConfigRepository interface {
	Get(ctx context.Context) (*AIActiveConfig, error)
	Update(ctx context.Context, cfg *AIActiveConfig) error
	SwitchProvider(ctx context.Context, providerID int64, updatedBy int64) error
	SetEnabled(ctx context.Context, enabled bool, updatedBy int64) error
}

// AIProviderModelRepository 提供商模型管理接口
type AIProviderModelRepository interface {
	Create(ctx context.Context, m *AIProviderModel) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*AIProviderModel, error)
	ListByProvider(ctx context.Context, provider string) ([]*AIProviderModel, error)
	ListAll(ctx context.Context) ([]*AIProviderModel, error)
	GetDefaultModel(ctx context.Context, provider string) (*AIProviderModel, error)
}

// SLORepository SLO 配置与路由映射访问接口
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse
type SLORepository interface {
	// Targets
	GetTargets(ctx context.Context, clusterID string) ([]*SLOTarget, error)
	GetTargetsByHost(ctx context.Context, clusterID, host string) ([]*SLOTarget, error)
	UpsertTarget(ctx context.Context, t *SLOTarget) error
	DeleteTarget(ctx context.Context, clusterID, host, timeRange string) error

	// Route Mapping
	UpsertRouteMapping(ctx context.Context, m *SLORouteMapping) error
	GetRouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*SLORouteMapping, error)
	GetRouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*SLORouteMapping, error)
	GetAllRouteMappings(ctx context.Context, clusterID string) ([]*SLORouteMapping, error)
	GetAllDomains(ctx context.Context, clusterID string) ([]string, error)
	DeleteRouteMapping(ctx context.Context, clusterID, serviceKey string) error
}


// ==================== Dialect 接口 ====================

// Dialect 数据库方言接口
// 每种数据库引擎 (SQLite, MySQL, PostgreSQL) 提供自己的实现
// Dialect 负责 SQL 生成和行扫描（各 DB 的类型映射不同）
type Dialect interface {
	Audit() AuditDialect
	User() UserDialect
	Event() EventDialect
	Notify() NotifyDialect
	Cluster() ClusterDialect
	Command() CommandDialect
	Settings() SettingsDialect
	AIConversation() AIConversationDialect
	AIMessage() AIMessageDialect
	AIProvider() AIProviderDialect
	AIActiveConfig() AIActiveConfigDialect
	AIProviderModel() AIProviderModelDialect
	SLO() SLODialect

	AIOpsBaseline() AIOpsBaselineDialect
	AIOpsGraph() AIOpsGraphDialect
	AIOpsIncident() AIOpsIncidentDialect
	Migrate(db *sql.DB) error
}

// AuditDialect 审计日志 SQL 方言
type AuditDialect interface {
	Insert(log *AuditLog) (query string, args []any)
	List(opts AuditQueryOpts) (query string, args []any)
	Count(opts AuditQueryOpts) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AuditLog, error)
}

// UserDialect 用户 SQL 方言
type UserDialect interface {
	Insert(user *User) (query string, args []any)
	Update(user *User) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByUsername(username string) (query string, args []any)
	SelectAll() (query string, args []any)
	UpdateLastLogin(id int64, ip string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*User, error)
}

// EventDialect 事件 SQL 方言
type EventDialect interface {
	Upsert(event *ClusterEvent) (query string, args []any)
	ListByCluster(clusterID string, opts EventQueryOpts) (query string, args []any)
	ListByInvolvedResource(clusterID, kind, namespace, name string) (query string, args []any)
	ListByType(clusterID, eventType string, since time.Time) (query string, args []any)
	DeleteBefore(clusterID string, before time.Time) (query string, args []any)
	CountByCluster(clusterID string) (query string, args []any)
	CountByHour(clusterID string, since time.Time) (query string, args []any)
	CountByHourAndKind(clusterID string, since time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*ClusterEvent, error)
	ScanHourlyCount(rows *sql.Rows) (*HourlyEventCount, error)
	ScanHourlyKindCount(rows *sql.Rows) (*HourlyKindCount, error)
}

// NotifyDialect 通知渠道 SQL 方言
type NotifyDialect interface {
	Insert(ch *NotifyChannel) (query string, args []any)
	Update(ch *NotifyChannel) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByType(channelType string) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectEnabled() (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifyChannel, error)
}

// ClusterDialect 集群 SQL 方言
type ClusterDialect interface {
	Insert(cluster *Cluster) (query string, args []any)
	Update(cluster *Cluster) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByUID(uid string) (query string, args []any)
	SelectAll() (query string, args []any)
	ScanRow(rows *sql.Rows) (*Cluster, error)
}

// CommandDialect 指令历史 SQL 方言
type CommandDialect interface {
	Insert(cmd *CommandHistory) (query string, args []any)
	Update(cmd *CommandHistory) (query string, args []any)
	SelectByCommandID(cmdID string) (query string, args []any)
	SelectByCluster(clusterID string, limit, offset int) (query string, args []any)
	SelectByUser(userID int64, limit, offset int) (query string, args []any)
	// 新增：带筛选条件的查询
	SelectWithOpts(opts CommandQueryOpts) (query string, args []any)
	CountWithOpts(opts CommandQueryOpts) (query string, args []any)
	ScanRow(rows *sql.Rows) (*CommandHistory, error)
}

// SettingsDialect 设置 SQL 方言
type SettingsDialect interface {
	SelectByKey(key string) (query string, args []any)
	SelectByPrefix(prefix string) (query string, args []any)
	Upsert(s *Setting) (query string, args []any)
	Delete(key string) (query string, args []any)
	SelectAll() (query string, args []any)
	ScanRow(rows *sql.Rows) (*Setting, error)
}

// AIConversationDialect AI 对话 SQL 方言
type AIConversationDialect interface {
	Insert(conv *AIConversation) (query string, args []any)
	Update(conv *AIConversation) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByUser(userID int64, limit, offset int) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIConversation, error)
}

// AIMessageDialect AI 消息 SQL 方言
type AIMessageDialect interface {
	Insert(msg *AIMessage) (query string, args []any)
	SelectByConversation(convID int64) (query string, args []any)
	DeleteByConversation(convID int64) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIMessage, error)
}

// AIProviderDialect AI 提供商 SQL 方言
type AIProviderDialect interface {
	Insert(p *AIProvider) (query string, args []any)
	Update(p *AIProvider) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectAll() (query string, args []any)
	IncrementUsage(id int64, requests, tokens int64, cost float64) (query string, args []any)
	UpdateStatus(id int64, status, errorMsg string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIProvider, error)
}

// AIActiveConfigDialect AI 当前配置 SQL 方言
type AIActiveConfigDialect interface {
	Select() (query string, args []any)
	Update(cfg *AIActiveConfig) (query string, args []any)
	SwitchProvider(providerID int64, updatedBy int64) (query string, args []any)
	SetEnabled(enabled bool, updatedBy int64) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIActiveConfig, error)
}

// AIProviderModelDialect 提供商模型 SQL 方言
type AIProviderModelDialect interface {
	Insert(m *AIProviderModel) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByProvider(provider string) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectDefault(provider string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIProviderModel, error)
}

// SLODialect SLO 配置与路由映射 SQL 方言
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse
type SLODialect interface {
	// Targets
	SelectTargets(clusterID string) (query string, args []any)
	SelectTargetsByHost(clusterID, host string) (query string, args []any)
	UpsertTarget(t *SLOTarget) (query string, args []any)
	DeleteTarget(clusterID, host, timeRange string) (query string, args []any)
	ScanTarget(rows *sql.Rows) (*SLOTarget, error)

	// Route Mapping
	UpsertRouteMapping(m *SLORouteMapping) (query string, args []any)
	SelectRouteMappingByServiceKey(clusterID, serviceKey string) (query string, args []any)
	SelectRouteMappingsByDomain(clusterID, domain string) (query string, args []any)
	SelectAllRouteMappings(clusterID string) (query string, args []any)
	SelectAllDomains(clusterID string) (query string, args []any)
	DeleteRouteMapping(clusterID, serviceKey string) (query string, args []any)
	ScanRouteMapping(rows *sql.Rows) (*SLORouteMapping, error)
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

// ==================== AIOps Repository 接口 ====================

// AIOpsBaselineRepository 基线状态数据访问接口
type AIOpsBaselineRepository interface {
	BatchUpsert(ctx context.Context, states []*AIOpsBaselineState) error
	ListAll(ctx context.Context) ([]*AIOpsBaselineState, error)
	ListByEntity(ctx context.Context, entityKey string) ([]*AIOpsBaselineState, error)
	DeleteByEntity(ctx context.Context, entityKey string) error
}

// AIOpsGraphRepository 依赖图快照数据访问接口
type AIOpsGraphRepository interface {
	Save(ctx context.Context, clusterID string, snapshot []byte) error
	Load(ctx context.Context, clusterID string) ([]byte, error)
	ListClusterIDs(ctx context.Context) ([]string, error)
}

// ==================== AIOps Dialect 接口 ====================

// AIOpsBaselineDialect 基线 SQL 方言
type AIOpsBaselineDialect interface {
	Upsert(state *AIOpsBaselineState) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectByEntity(entityKey string) (query string, args []any)
	DeleteByEntity(entityKey string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIOpsBaselineState, error)
}

// AIOpsGraphDialect 依赖图 SQL 方言
type AIOpsGraphDialect interface {
	Upsert(clusterID string, snapshot []byte) (query string, args []any)
	SelectByCluster(clusterID string) (query string, args []any)
	SelectAllClusterIDs() (query string, args []any)
	ScanSnapshot(rows *sql.Rows) (clusterID string, data []byte, err error)
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

// ==================== AIOps Incident Repository 接口 ====================

// AIOpsIncidentRepository 事件数据访问接口
type AIOpsIncidentRepository interface {
	CreateIncident(ctx context.Context, inc *AIOpsIncident) error
	GetByID(ctx context.Context, id string) (*AIOpsIncident, error)
	UpdateState(ctx context.Context, id, state, severity string) error
	Resolve(ctx context.Context, id string, resolvedAt time.Time) error
	UpdateRootCause(ctx context.Context, id, rootCause string) error
	UpdatePeakRisk(ctx context.Context, id string, peakRisk float64) error
	IncrementRecurrence(ctx context.Context, id string) error
	List(ctx context.Context, opts AIOpsIncidentQueryOpts) ([]*AIOpsIncident, error)
	Count(ctx context.Context, opts AIOpsIncidentQueryOpts) (int, error)
	AddEntity(ctx context.Context, entity *AIOpsIncidentEntity) error
	GetEntities(ctx context.Context, incidentID string) ([]*AIOpsIncidentEntity, error)
	AddTimeline(ctx context.Context, entry *AIOpsIncidentTimeline) error
	GetTimeline(ctx context.Context, incidentID string) ([]*AIOpsIncidentTimeline, error)
	GetIncidentStats(ctx context.Context, clusterID string, since time.Time) (*AIOpsIncidentStatsRaw, error)
	TopRootCauses(ctx context.Context, clusterID string, since time.Time, limit int) ([]AIOpsRootCauseCount, error)
	ListByEntity(ctx context.Context, entityKey string, since time.Time) ([]*AIOpsIncident, error)
}

// ==================== AIOps Incident Dialect 接口 ====================

// AIOpsIncidentDialect 事件 SQL 方言
type AIOpsIncidentDialect interface {
	InsertIncident(inc *AIOpsIncident) (string, []any)
	SelectByID(id string) (string, []any)
	UpdateState(id, state, severity string) (string, []any)
	Resolve(id string, resolvedAt time.Time) (string, []any)
	UpdateRootCause(id, rootCause string) (string, []any)
	UpdatePeakRisk(id string, peakRisk float64) (string, []any)
	IncrementRecurrence(id string) (string, []any)
	InsertEntity(entity *AIOpsIncidentEntity) (string, []any)
	SelectEntities(incidentID string) (string, []any)
	InsertTimeline(entry *AIOpsIncidentTimeline) (string, []any)
	SelectTimeline(incidentID string) (string, []any)
	ScanIncident(rows *sql.Rows) (*AIOpsIncident, error)
	ScanEntity(rows *sql.Rows) (*AIOpsIncidentEntity, error)
	ScanTimeline(rows *sql.Rows) (*AIOpsIncidentTimeline, error)
}

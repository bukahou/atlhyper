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
	Audit    AuditRepository
	User     UserRepository
	Event    ClusterEventRepository
	Notify   NotifyChannelRepository
	Cluster  ClusterRepository
	Command  CommandHistoryRepository
	Settings SettingsRepository

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

// Setting 系统设置
type Setting struct {
	Key         string
	Value       string // JSON
	Description string
	UpdatedAt   time.Time
	UpdatedBy   int64
}

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
}

// SettingsRepository 设置接口
type SettingsRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	Set(ctx context.Context, setting *Setting) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]*Setting, error)
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
	ScanRow(rows *sql.Rows) (*CommandHistory, error)
}

// SettingsDialect 设置 SQL 方言
type SettingsDialect interface {
	SelectByKey(key string) (query string, args []any)
	Upsert(s *Setting) (query string, args []any)
	Delete(key string) (query string, args []any)
	SelectAll() (query string, args []any)
	ScanRow(rows *sql.Rows) (*Setting, error)
}

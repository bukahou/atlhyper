// atlhyper_master_v2/database/interfaces_notify.go
// 通知模块的 Repository 接口与 SQL 方言接口
package database

import (
	"context"
	"database/sql"
	"time"
)

// ==================== Repository 接口 ====================

// NotifyPolicyRepository 全局通知策略（单行）
type NotifyPolicyRepository interface {
	// Get 取当前策略；库里还没有策略行时返回出厂值
	Get(ctx context.Context) (*NotifyPolicy, error)
	Save(ctx context.Context, policy *NotifyPolicy) error
}

// NotifyAlertRepository 告警
type NotifyAlertRepository interface {
	// Upsert 按指纹写入；时间戳由调用方给出（管道用自己的时钟）
	Upsert(ctx context.Context, alert *NotifyAlert) error
	// Get 不存在时返回 (nil, nil)
	Get(ctx context.Context, fingerprint string) (*NotifyAlert, error)
	ListFiring(ctx context.Context) ([]*NotifyAlert, error)
	List(ctx context.Context, opts NotifyAlertQueryOpts) ([]*NotifyAlert, error)
	Count(ctx context.Context, opts NotifyAlertQueryOpts) (int, error)
	// DeleteResolvedBefore 删除在给定时刻之前结束的告警
	DeleteResolvedBefore(ctx context.Context, before time.Time) (int64, error)
}

// NotifyDeliveryRepository 送达记录
type NotifyDeliveryRepository interface {
	// Create 写入一条记录并回填 ID；时间戳由调用方给出
	Create(ctx context.Context, delivery *NotifyDelivery) error
	// UpdateBatch 把一次发送的结果写到该批次的全部记录上
	UpdateBatch(ctx context.Context, batchID string, result NotifyDeliveryResult) error
	// ListUnfinished 还没有定论的记录（pending / failed），供重启后接着重试
	ListUnfinished(ctx context.Context) ([]*NotifyDelivery, error)
	List(ctx context.Context, opts NotifyDeliveryQueryOpts) ([]*NotifyDelivery, error)
	Count(ctx context.Context, opts NotifyDeliveryQueryOpts) (int, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}

// NotifyReceiverRepository 接收人。配置整体加密落库，读出时解密
type NotifyReceiverRepository interface {
	// Create 名称重复 → ErrNotifyReceiverNameTaken；成功后回填 ID 与时间
	Create(ctx context.Context, receiver *NotifyReceiver) error
	// Update 不存在 → ErrNotifyReceiverNotFound；名称重复 → ErrNotifyReceiverNameTaken
	Update(ctx context.Context, receiver *NotifyReceiver) error
	// Delete 不存在 → ErrNotifyReceiverNotFound
	Delete(ctx context.Context, id int64) error
	// GetByID 不存在时返回 (nil, nil)
	GetByID(ctx context.Context, id int64) (*NotifyReceiver, error)
	List(ctx context.Context) ([]*NotifyReceiver, error)
}

// NotifyRuleRepository 路由规则
type NotifyRuleRepository interface {
	// Create 新规则排在最后；成功后回填 ID、Position 与时间
	Create(ctx context.Context, rule *NotifyRule) error
	// Update 不改 Position；不存在 → ErrNotifyRuleNotFound
	Update(ctx context.Context, rule *NotifyRule) error
	// Delete 不存在 → ErrNotifyRuleNotFound
	Delete(ctx context.Context, id int64) error
	// GetByID 不存在时返回 (nil, nil)
	GetByID(ctx context.Context, id int64) (*NotifyRule, error)
	// List 按 Position 从小到大
	List(ctx context.Context) ([]*NotifyRule, error)
	// Reorder 按给定顺序重排（同一事务）。ids 必须恰好是全部规则的一个排列，否则 ErrNotifyRuleOrderMismatch
	Reorder(ctx context.Context, ids []int64) error
}

// NotifySilenceRepository 静默。静默不删除，只会过期；过期一段时间后由清理任务删除
type NotifySilenceRepository interface {
	Create(ctx context.Context, silence *NotifySilence) error
	// GetByID 不存在时返回 (nil, nil)
	GetByID(ctx context.Context, id int64) (*NotifySilence, error)
	// List 全部静默，开始时间新的在前
	List(ctx context.Context) ([]*NotifySilence, error)
	// SetEndsAt 改结束时间（提前结束）；不存在 → ErrNotifySilenceNotFound
	SetEndsAt(ctx context.Context, id int64, endsAt time.Time) error
	// DeleteEndedBefore 删除在给定时刻之前结束的静默
	DeleteEndedBefore(ctx context.Context, before time.Time) (int64, error)
}

// ==================== Dialect 接口 ====================

// NotifyPolicyDialect 通知策略 SQL 方言
type NotifyPolicyDialect interface {
	Select() (query string, args []any)
	Upsert(policy *NotifyPolicy, now time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifyPolicy, error)
}

// NotifyAlertDialect 告警 SQL 方言
type NotifyAlertDialect interface {
	Upsert(alert *NotifyAlert) (query string, args []any)
	SelectByFingerprint(fingerprint string) (query string, args []any)
	SelectFiring() (query string, args []any)
	List(opts NotifyAlertQueryOpts) (query string, args []any)
	Count(opts NotifyAlertQueryOpts) (query string, args []any)
	DeleteResolvedBefore(before time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifyAlert, error)
}

// NotifyDeliveryDialect 送达记录 SQL 方言
type NotifyDeliveryDialect interface {
	Insert(delivery *NotifyDelivery) (query string, args []any)
	UpdateBatch(batchID string, result NotifyDeliveryResult) (query string, args []any)
	SelectUnfinished() (query string, args []any)
	List(opts NotifyDeliveryQueryOpts) (query string, args []any)
	Count(opts NotifyDeliveryQueryOpts) (query string, args []any)
	DeleteBefore(before time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifyDelivery, error)
}

// NotifyReceiverDialect 接收人 SQL 方言。config 列存的是密文，加解密在仓库里做
type NotifyReceiverDialect interface {
	Insert(receiver *NotifyReceiver, encryptedConfig string, now time.Time) (query string, args []any)
	Update(receiver *NotifyReceiver, encryptedConfig string, now time.Time) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectAll() (query string, args []any)
	ScanRow(rows *sql.Rows) (receiver *NotifyReceiver, encryptedConfig string, err error)
	// IsDuplicateName 该错误是否由名称唯一约束引起
	IsDuplicateName(err error) bool
}

// NotifyRuleDialect 路由规则 SQL 方言
type NotifyRuleDialect interface {
	Insert(rule *NotifyRule, position int, now time.Time) (query string, args []any)
	Update(rule *NotifyRule, now time.Time) (query string, args []any)
	UpdatePosition(id int64, position int) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectMaxPosition() (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifyRule, error)
}

// NotifySilenceDialect 静默 SQL 方言
type NotifySilenceDialect interface {
	Insert(silence *NotifySilence, now time.Time) (query string, args []any)
	UpdateEndsAt(id int64, endsAt time.Time) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectAll() (query string, args []any)
	DeleteEndedBefore(before time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifySilence, error)
}

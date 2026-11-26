// atlhyper_master_v2/database/interfaces_service_account.go
// 服务账号与访问令牌：Repository 与 Dialect 接口
package database

import (
	"context"
	"database/sql"
	"time"
)

// ServiceAccountRepository 服务账号：users 表里 kind = service 的行。
//
// 每条语句都限定在服务账号上：拿人的 id 调用等同于不存在。
type ServiceAccountRepository interface {
	// Create 建服务账号（种类与口令占位由本仓库写入，调用方给的不作数），成功后回填 ID；
	// 账号名与任何已有用户名重复 → ErrUsernameTaken
	Create(ctx context.Context, account *User) error
	// List 全部服务账号，按账号名排序
	List(ctx context.Context) ([]*User, error)
	// GetByID 不存在或不是服务账号时返回 (nil, nil)
	GetByID(ctx context.Context, id int64) (*User, error)
	// Update 改显示名、等级与启停；不存在 → ErrServiceAccountNotFound
	Update(ctx context.Context, account *User) error
	// Delete 删除账号，令牌随外键一起删；不存在 → ErrServiceAccountNotFound
	Delete(ctx context.Context, id int64) error
}

// APITokenRepository 访问令牌：只存哈希与显示用前缀
type APITokenRepository interface {
	// CreateWithinLimit 该账号未吊销的令牌少于 limit 时写入（判断与写入是同一条语句），成功后回填 ID。
	// 已满 → ErrAPITokenLimitReached；账号不存在 → ErrServiceAccountNotFound
	CreateWithinLimit(ctx context.Context, token *APIToken, limit int) error
	// GetByHash 哈希为空或查不到时返回 (nil, nil)
	GetByHash(ctx context.Context, tokenHash string) (*APIToken, error)
	// List 全部令牌：先按账号，同一账号新签发的在前
	List(ctx context.Context) ([]*APIToken, error)
	// ListByUser 某个账号的令牌，新签发的在前
	ListByUser(ctx context.Context, userID int64) ([]*APIToken, error)
	// Revoke 吊销（只写 revoked_at，行保留；已吊销的保持第一次的时间）。该账号下没有这枚令牌 → ErrAPITokenNotFound
	Revoke(ctx context.Context, userID, tokenID int64, at time.Time) error
	// Touch 记下最近使用时间
	Touch(ctx context.Context, id int64, usedAt time.Time) error
}

// ServiceAccountDialect 服务账号 SQL 方言
type ServiceAccountDialect interface {
	Insert(account *User, now time.Time) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	Update(account *User, now time.Time) (query string, args []any)
	Delete(id int64) (query string, args []any)
	ScanRow(rows *sql.Rows) (*User, error)
	// IsDuplicateUsername 写入失败是否因为用户名已被占用
	IsDuplicateUsername(err error) bool
}

// APITokenDialect 访问令牌 SQL 方言
type APITokenDialect interface {
	InsertWithinLimit(token *APIToken, limit int) (query string, args []any)
	SelectByHash(tokenHash string) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectByUser(userID int64) (query string, args []any)
	Revoke(userID, tokenID int64, at time.Time) (query string, args []any)
	Touch(id int64, usedAt time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*APIToken, error)
	// IsUnknownAccount 写入失败是否因为令牌指向的账号不存在（外键）
	IsUnknownAccount(err error) bool
}

// atlhyper_master_v2/database/repository/audit.go
// AuditRepository 接口定义
package repository

import (
	"context"
	"time"
)

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

// AuditRepository 审计日志接口
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, opts AuditQueryOpts) ([]*AuditLog, error)
	Count(ctx context.Context, opts AuditQueryOpts) (int64, error)
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

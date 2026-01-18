// atlhyper_master_v2/database/repository/settings.go
// SettingsRepository 接口定义
package repository

import (
	"context"
	"time"
)

// Setting 系统设置
type Setting struct {
	Key         string
	Value       string // JSON
	Description string
	UpdatedAt   time.Time
	UpdatedBy   int64
}

// SettingsRepository 设置接口
type SettingsRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	Set(ctx context.Context, setting *Setting) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]*Setting, error)
}

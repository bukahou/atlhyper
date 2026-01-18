// atlhyper_master_v2/database/repository/user.go
// UserRepository 接口定义
package repository

import (
	"context"
	"time"
)

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

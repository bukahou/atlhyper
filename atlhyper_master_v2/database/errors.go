// atlhyper_master_v2/database/errors.go
// 数据库层错误定义
package database

import "errors"

var (
	// ErrIdentityExists 外部身份（issuer + subject）已关联到某个用户
	ErrIdentityExists = errors.New("外部身份已存在")
	// ErrUsernameTaken 用户名已被占用
	ErrUsernameTaken = errors.New("用户名已被占用")
)

// model/entity/user.go
// 用户实体（数据库模型）
package entity

import "time"

// User 用户实体
type User struct {
	ID           int
	Username     string
	PasswordHash string
	DisplayName  string
	Email        string
	Role         int
	CreatedAt    time.Time
	LastLogin    *time.Time
}

// service/user/dto.go
// 用户服务 DTO 定义
package user

import "time"

// UserDTO 用户信息 DTO（不含敏感信息）
type UserDTO struct {
	ID          int        `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"displayName"`
	Email       string     `json:"email"`
	Role        int        `json:"role"`
	CreatedAt   time.Time  `json:"createdAt"`
	LastLogin   *time.Time `json:"lastLogin,omitempty"`
}

// LoginResult 登录结果
type LoginResult struct {
	Token      string   `json:"token"`
	User       UserDTO  `json:"user"`
	ClusterIDs []string `json:"clusterIds"`
}

// RegisterReq 注册请求
type RegisterReq struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Role        int    `json:"role"`
}

// LoginReq 登录请求
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateRoleReq 更新角色请求
type UpdateRoleReq struct {
	ID   int `json:"id"`
	Role int `json:"role"`
}

// DeleteUserReq 删除用户请求
type DeleteUserReq struct {
	ID int `json:"id"`
}

// AuditLogDTO 审计日志 DTO
type AuditLogDTO struct {
	ID        int    `json:"id"`
	UserID    int    `json:"userId"`
	Username  string `json:"username"`
	Role      int    `json:"role"`
	Action    string `json:"action"`
	Success   bool   `json:"success"`
	IP        string `json:"ip"`
	Method    string `json:"method"`
	Status    int    `json:"status"`
	Timestamp string `json:"timestamp"`
}

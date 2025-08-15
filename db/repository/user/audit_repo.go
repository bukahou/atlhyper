package user

import (
	"NeuroController/db/utils"
	"context"
	"database/sql"
	"time"
)

// AuditRecord 审计记录结构体
type AuditRecord struct {
	UserID   int            // 用户 ID（未登录时传 0）
	Username string         // 用户名（未登录时传 "anonymous"）
	Role     int            // 角色（1=Viewer, 2=Operator, 3=Admin）
	Action   string         // 动作（如 pod.restart / auth.login）
	Success  bool           // 是否成功
	IP       sql.NullString // 客户端 IP
	Method   sql.NullString // HTTP 方法
	Status   sql.NullInt64  // HTTP 状态码
	TimeISO  string         // ISO 格式时间（可选，不传则使用 time.Now）
}

// Insert 插入一条审计记录
func InsertAuditLogs(ctx context.Context, r AuditRecord) error {
	// 统一时间处理
	ts := r.TimeISO
	if ts == "" {
		ts = time.Now().Format("2006-01-02T15:04:05Z07:00") // RFC3339，保留本地时区偏移
	}

	// SQLite 中 true/false 用 1/0 存
	successInt := 0
	if r.Success {
		successInt = 1
	}

	// 执行插入
	_, err := utils.DB.ExecContext(ctx, `
		INSERT INTO user_audit_logs
			(timestamp, user_id, username, role, action, success, ip, method, status)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		ts,
		r.UserID,
		r.Username,
		r.Role,
		r.Action,
		successInt,
		r.IP,
		r.Method,
		r.Status,
	)

	return err
}
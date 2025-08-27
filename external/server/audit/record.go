// external/audit/record.go
package audit

import (
	"NeuroController/db/repository/user"
	"context"
	"database/sql"

	"github.com/gin-gonic/gin"
)

// Record 统一审计入口：业务 handler / 登录 / 鉴权失败都调用它
// - action: 例如 "pod.restart" / "auth.login" / "auth.guard.forbidden"
// - success: true/false
// - status: HTTP 状态码（可选；传 <=0 则不写）
func Record(c *gin.Context, action string, success bool, status int) error {
	uid, uname, role := GetUserFromCtxSafe(c)

	var ip sql.NullString
	var method sql.NullString
	var st sql.NullInt64

	if c != nil {
		if cip := c.ClientIP(); cip != "" {
			ip = sql.NullString{String: cip, Valid: true}
		}
		if c.Request != nil && c.Request.Method != "" {
			method = sql.NullString{String: c.Request.Method, Valid: true}
		}
	}
	if status > 0 {
		st = sql.NullInt64{Int64: int64(status), Valid: true}
	}

	return user.InsertAuditLogs(context.Background(), user.AuditRecord{
		UserID:   uid,
		Username: uname,
		Role:     role,
		Action:   action,
		Success:  success,
		IP:       ip,
		Method:   method,
		Status:   st,
	})
}

// RecordLogin 登录专用（登录阶段未注入 JWT）
func RecordLogin(c *gin.Context, username string, success bool, status int) error {
	var ip sql.NullString
	var method sql.NullString
	var st sql.NullInt64

	if c != nil {
		if cip := c.ClientIP(); cip != "" {
			ip = sql.NullString{String: cip, Valid: true}
		}
		if c.Request != nil && c.Request.Method != "" {
			method = sql.NullString{String: c.Request.Method, Valid: true}
		}
	}
	if status > 0 {
		st = sql.NullInt64{Int64: int64(status), Valid: true}
	}

	// 登录阶段：user_id=0, role=1(Viewer) 作为占位
	return user.InsertAuditLogs(context.Background(), user.AuditRecord{
		UserID:   0,
		Username: safeUsername(username),
		Role:     1,
		Action:   "auth.login",
		Success:  success,
		IP:       ip,
		Method:   method,
		Status:   st,
	})
}

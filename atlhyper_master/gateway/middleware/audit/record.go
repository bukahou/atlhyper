// gateway/middleware/audit/record.go
package audit

import (
	"context"

	"AtlHyper/atlhyper_master/repository"

	"github.com/gin-gonic/gin"
)

// Record 统一审计入口：业务 handler / 登录 / 鉴权失败都调用它
// - action: 例如 "pod.restart" / "auth.login" / "auth.guard.forbidden"
// - success: true/false
// - status: HTTP 状态码（可选；传 <=0 则不写）
func Record(c *gin.Context, action string, success bool, status int) error {
	uid, uname, role := GetUserFromCtxSafe(c)

	ip := ""
	method := ""
	if c != nil {
		ip = c.ClientIP()
		if c.Request != nil {
			method = c.Request.Method
		}
	}

	return repository.Audit.Insert(context.Background(), &repository.AuditLog{
		UserID:   uid,
		Username: uname,
		Role:     role,
		Action:   action,
		Success:  success,
		IP:       ip,
		Method:   method,
		Status:   status,
	})
}

// RecordLogin 登录专用（登录阶段未注入 JWT）
func RecordLogin(c *gin.Context, username string, success bool, status int) error {
	ip := ""
	method := ""
	if c != nil {
		ip = c.ClientIP()
		if c.Request != nil {
			method = c.Request.Method
		}
	}

	// 登录阶段：user_id=0, role=1(Viewer) 作为占位
	return repository.Audit.Insert(context.Background(), &repository.AuditLog{
		UserID:   0,
		Username: safeUsername(username),
		Role:     1,
		Action:   "auth.login",
		Success:  success,
		IP:       ip,
		Method:   method,
		Status:   status,
	})
}

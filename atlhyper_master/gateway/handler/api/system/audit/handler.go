// gateway/handler/api/system/audit/handler.go
// 审计日志处理器
package audit

import (
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	userSvc "AtlHyper/atlhyper_master/service/db/user"

	"github.com/gin-gonic/gin"
)

// HandleGetAuditLogs 获取用户审计日志
// GET /uiapi/system/audit/list
func HandleGetAuditLogs(c *gin.Context) {
	logs, err := userSvc.GetAuditLogs(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, 50000, "查询用户审计日志失败")
		return
	}

	response.Success(c, "获取用户审计日志成功", logs)
}

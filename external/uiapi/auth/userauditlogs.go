package auth

import (
	"NeuroController/db/repository/user"
	"NeuroController/external/uiapi/response"

	"github.com/gin-gonic/gin"
)

//获取用户审计日志
// 获取所有用户审计日志
// 处理 GET /auth/userauditlogs/list 请求
// ✅ 查询用户审计日志：调用 GetUserAuditLogs 函数 → 返回日志列表
func HandleGetUserAuditLogs(c *gin.Context) {
	logs, err := user.GetUserAuditLogs()
	if err != nil {
		// 统一错误结构，HTTP 依然 200，由前端用 code 判断
		response.ErrorCode(c, 50000, "查询用户审计日志失败")
		return
	}

	// 成功：带消息与数据
	response.Success(c, "获取用户审计日志成功", logs)
}
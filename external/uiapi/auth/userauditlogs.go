package auth

import (
	"NeuroController/db/repository/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

//获取用户审计日志
// 获取所有用户审计日志
// 处理 GET /auth/userauditlogs/list 请求
// ✅ 查询用户审计日志：调用 GetUserAuditLogs 函数 → 返回日志列表
func HandleGetUserAuditLogs(c *gin.Context){
	logs ,err := user.GetUserAuditLogs()
	if err != nil {
		c.JSON((http.StatusInternalServerError), gin.H{"error": "查询用户审计日志失败"})
		return
	}
	c.JSON((http.StatusOK), gin.H{"logs": logs})
}
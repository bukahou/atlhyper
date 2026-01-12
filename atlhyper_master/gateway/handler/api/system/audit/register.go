// gateway/handler/api/system/audit/register.go
// 审计日志路由注册
package audit

import (
	"AtlHyper/atlhyper_master/gateway/middleware/auth"

	"github.com/gin-gonic/gin"
)

// Register 注册审计相关路由
// 权限说明：Viewer+（需要登录才能查看审计日志）
func Register(router *gin.RouterGroup) {
	g := router.Group("/system/audit")
	g.Use(auth.RequireAuth(), auth.RequireMinRole(auth.RoleViewer))
	{
		g.GET("/list", HandleGetAuditLogs)
	}
}

// gateway/handler/api/system/notify/register.go
// 通知配置路由注册
package notify

import (
	"AtlHyper/atlhyper_master/gateway/integration/alert"
	"AtlHyper/atlhyper_master/gateway/middleware/auth"

	"github.com/gin-gonic/gin"
)

// Register 注册通知配置相关路由
// 权限说明：
//   - 公开查看：slack/get, mail/get（低权限返回脱敏数据）
//   - Admin：slack/update, mail/update（配置修改）
func Register(router *gin.RouterGroup) {
	g := router.Group("/system/notify")
	{
		// 公开接口（低权限返回脱敏数据）
		g.POST("/slack/get", GetSlackConfig)
		g.POST("/mail/get", GetMailConfig)
		g.GET("/preview", alert.HandleAlertSlackPreview)

		// 管理员接口
		admin := g.Group("")
		admin.Use(auth.RequireAuth(), auth.RequireMinRole(auth.RoleAdmin))
		{
			admin.POST("/slack/update", UpdateSlackConfig)
			admin.POST("/mail/update", UpdateMailConfig)
		}
	}
}

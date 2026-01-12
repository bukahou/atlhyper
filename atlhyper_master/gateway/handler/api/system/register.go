// gateway/handler/api/system/register.go
// 系统设置页 API
package system

import (
	"AtlHyper/atlhyper_master/gateway/handler/api/system/audit"
	"AtlHyper/atlhyper_master/gateway/handler/api/system/notify"

	"github.com/gin-gonic/gin"
)

// Register 注册系统管理相关路由
func Register(router *gin.RouterGroup) {
	// 通知配置模块
	notify.Register(router)

	// 审计日志模块
	audit.Register(router)
}

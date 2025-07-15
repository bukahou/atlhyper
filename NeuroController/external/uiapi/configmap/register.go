// =======================================================================================
// 📄 register.go（external/uiapi/configmap）
//
// ✨ 文件说明：
//     注册 ConfigMap 相关的所有路由接口：
//     - 按命名空间查询列表
//     - 查询详情
//
// ✍️ 作者：bukahou (@ZGMF-X10A)
// =======================================================================================

package configmap

import (
	"github.com/gin-gonic/gin"
)

// RegisterConfigMapRoutes 将 ConfigMap 模块相关的路由注册到指定分组
func RegisterConfigMapRoutes(router *gin.RouterGroup) {
	router.GET("/list", ListAllConfigMapsHandler)                          // ✅ 所有命名空间
	router.GET("/list/by-namespace/:ns", ListConfigMapsByNamespaceHandler) // ✅ 指定命名空间
	router.GET("/get/:ns/:name", GetConfigMapDetailHandler)                // ✅ 详情查询
	// ⚙️ 告警系统配置相关接口（独立更新）
	router.GET("/alert/get", GetAlertSettingsHandler)            // ✅ 获取当前告警配置
	router.POST("/alert/slack", UpdateSlackConfigHandler)        // ✅ 更新 Slack 配置
	router.POST("/alert/webhook", UpdateWebhookSwitchHandler)    // ✅ 更新 Webhook 开关
	router.POST("/alert/mail", UpdateMailConfigHandler)          // ✅ 更新邮件配置
}

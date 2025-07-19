package commonapi

import "github.com/gin-gonic/gin"

func RegisterCommonAPIRoutes(r *gin.RouterGroup) {
	r.GET("/cleaned-events", HandleCleanedEvents)
	
	// 满足策略条件时的完整告警数据（如邮件发送）
	r.GET("/alert-group", HandleAlertGroup)

	// 轻量展示用的告警数据（如 Slack/页面概览）
	r.GET("/alert-group-light", HandleLightweightAlertGroup)
}

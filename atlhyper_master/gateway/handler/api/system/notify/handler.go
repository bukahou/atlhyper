// gateway/handler/api/system/notify/handler.go
// 通知配置处理器
package notify

import (
	"AtlHyper/atlhyper_master/gateway/middleware/auth"
	"AtlHyper/atlhyper_master/gateway/middleware/response"
	svcfg "AtlHyper/atlhyper_master/service/db/config"

	"github.com/gin-gonic/gin"
)

// getUserRole 从上下文获取用户角色，未登录返回 0
func getUserRole(c *gin.Context) int {
	if role, exists := c.Get("role"); exists {
		if r, ok := role.(float64); ok {
			return int(r)
		}
	}
	return 0
}

// ================== Slack 配置 ==================

// GetSlackConfig 获取 Slack 配置
// POST /uiapi/system/notify/slack/get
func GetSlackConfig(c *gin.Context) {
	role := getUserRole(c)

	// 低权限用户返回脱敏数据
	if role < auth.RoleOperator {
		response.Success(c, "OK", svcfg.GetSlackConfigMasked())
		return
	}

	cfg, err := svcfg.GetSlackConfigUI(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, 50000, "读取 Slack 配置失败")
		return
	}
	response.Success(c, "OK", cfg)
}

// UpdateSlackConfig 更新 Slack 配置
// POST /uiapi/system/notify/slack/update
func UpdateSlackConfig(c *gin.Context) {
	var req svcfg.SlackUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数格式错误")
		return
	}

	if req.Enable == nil && req.Webhook == nil && req.IntervalSec == nil {
		response.Error(c, "未提供任何可更新字段")
		return
	}

	if err := svcfg.UpdateSlackConfigUI(c.Request.Context(), req); err != nil {
		response.ErrorCode(c, 50000, "更新 Slack 配置失败")
		return
	}

	cfg, err := svcfg.GetSlackConfigUI(c.Request.Context())
	if err != nil {
		response.SuccessMsg(c, "Slack 配置已更新")
		return
	}
	response.Success(c, "Slack 配置已更新", cfg)
}

// ================== Mail 配置 ==================

// GetMailConfig 获取邮件配置
// POST /uiapi/system/notify/mail/get
func GetMailConfig(c *gin.Context) {
	role := getUserRole(c)

	// 低权限用户返回脱敏数据
	if role < auth.RoleOperator {
		response.Success(c, "OK", svcfg.GetMailConfigMasked())
		return
	}

	cfg, err := svcfg.GetMailConfigUI(c.Request.Context())
	if err != nil {
		response.ErrorCode(c, 50000, "读取邮件配置失败")
		return
	}
	response.Success(c, "OK", cfg)
}

// UpdateMailConfig 更新邮件配置
// POST /uiapi/system/notify/mail/update
func UpdateMailConfig(c *gin.Context) {
	var req svcfg.MailUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数格式错误")
		return
	}

	if req.Enable == nil && req.SMTPHost == nil && req.SMTPPort == nil &&
		req.Username == nil && req.Password == nil && req.FromAddr == nil &&
		req.ToAddrs == nil && req.IntervalSec == nil {
		response.Error(c, "未提供任何可更新字段")
		return
	}

	if err := svcfg.UpdateMailConfigUI(c.Request.Context(), req); err != nil {
		response.ErrorCode(c, 50000, "更新邮件配置失败")
		return
	}

	cfg, err := svcfg.GetMailConfigUI(c.Request.Context())
	if err != nil {
		response.SuccessMsg(c, "邮件配置已更新")
		return
	}
	response.Success(c, "邮件配置已更新", cfg)
}

package web_api

import (
	uicfg "AtlHyper/atlhyper_master/interfaces/ui_interfaces/config"
	"AtlHyper/atlhyper_master/server/api/response"

	"github.com/gin-gonic/gin"
)

// GET/POST /uiapi/config/slack/get
func GetSlackConfig(c *gin.Context) {
	cfg, err := uicfg.GetSlackConfigUI()
	if err != nil {
		response.ErrorCode(c, 50000, "读取 Slack 配置失败")
		return
	}
	response.Success(c, "OK", cfg)
}

type updateReq = uicfg.SlackUpdateReq 

// POST /uiapi/config/slack/update
func UpdateSlackConfig(c *gin.Context) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, "参数格式错误")
		return
	}
	// 如果全部字段都是 nil，则直接提示
	if req.Enable == nil && req.Webhook == nil && req.IntervalSec == nil {
		response.Error(c, "未提供任何可更新字段")
		return
	}
	if err := uicfg.UpdateSlackConfigUI(req); err != nil {
		response.ErrorCode(c, 50000, "更新 Slack 配置失败")
		return
	}
	// 返回最新配置
	cfg, err := uicfg.GetSlackConfigUI()
	if err != nil {
		response.SuccessMsg(c, "Slack 配置已更新")
		return
	}
	response.Success(c, "Slack 配置已更新", cfg)
}

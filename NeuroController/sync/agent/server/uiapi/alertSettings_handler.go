package uiapi

import (
	uiapi "NeuroController/interfaces/ui_api"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GET /uiapi/config/alert
func HandleGetAlertConfig(c *gin.Context) {
	cfg, err := uiapi.GetCurrentAlertConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "❌ 获取配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// POST /uiapi/config/slack
func HandleUpdateSlackConfig(c *gin.Context) {
	var req struct {
		Enabled bool   `json:"enabled"`
		Webhook string `json:"webhook"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "❌ 请求格式错误: " + err.Error()})
		return
	}

	if err := uiapi.UpdateSlackConfig(req.Enabled, req.Webhook); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "❌ 更新 Slack 配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "✅ Slack 配置更新成功"})
}

// POST /uiapi/config/webhook
func HandleUpdateWebhookConfig(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "❌ 请求格式错误: " + err.Error()})
		return
	}

	if err := uiapi.UpdateWebhookEnabled(req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "❌ 更新 Webhook 配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "✅ Webhook 配置更新成功"})
}

// POST /uiapi/config/mail
func HandleUpdateMailConfig(c *gin.Context) {
	var req struct {
		Enabled  bool     `json:"enabled"`
		Username string   `json:"username"`
		Password string   `json:"password"`
		From     string   `json:"from"`
		To       string   `json:"to"` // 支持逗号分隔字符串或数组格式
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "❌ 请求格式错误: " + err.Error()})
		return
	}

	toList := strings.Split(req.To, ",")
	for i := range toList {
		toList[i] = strings.TrimSpace(toList[i])
	}

	if err := uiapi.UpdateMailConfig(req.Enabled, req.Username, req.Password, req.From, toList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "❌ 更新邮件配置失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "✅ 邮件配置更新成功"})
}
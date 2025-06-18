package slack

import (
	"NeuroController/config"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SendSlackAlert 发送 BlockKit 消息到 Slack Webhook
func SendSlackAlert(payload map[string]interface{}) error {

	webhookURL := config.GlobalConfig.Slack.WebhookURL

	if webhookURL == "" {
		return fmt.Errorf("Slack Webhook 未配置（SLACK_WEBHOOK_URL）")
	}

	// ✅ JSON 编码
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON 编码失败: %v", err)
	}

	// ✅ 构造 POST 请求
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// ✅ 执行请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Slack 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// ✅ 返回状态检查
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Slack 返回异常状态码: %d", resp.StatusCode)
	}

	return nil
}

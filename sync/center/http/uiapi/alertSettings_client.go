package uiapi

import "NeuroController/sync/center/http"

//// ========================
// 🔧 外部调用接口
//// ========================

// Slack
func UpdateSlack(enabled bool, webhook string) error {
	payload := map[string]any{
		"enabled": enabled,
		"webhook": webhook,
	}
	return http.PostToAgent("/agent/uiapi/config/slack", payload)
}

// Webhook
func UpdateWebhook(enabled bool) error {
	payload := map[string]any{
		"enabled": enabled,
	}
	return http.PostToAgent("/agent/uiapi/config/webhook", payload)
}

// Mail
func UpdateMail(enabled bool, username, password, from string, to []string) error {
	payload := map[string]any{
		"enabled":  enabled,
		"username": username,
		"password": password,
		"from":     from,
		"to":       to, // 支持逗号分隔或数组均可
	}
	return http.PostToAgent("/agent/uiapi/config/mail", payload)
}

// 查询配置（只从第一个 Agent 获取）
func GetAlertConfig() (map[string]any, error) {
	var result map[string]any
	err := http.GetFromAgent("/agent/uiapi/config/alert", &result)
	return result, err
}
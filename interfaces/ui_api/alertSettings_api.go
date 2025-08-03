package uiapi

// import (
// 	"NeuroController/internal/operator/configmap"
// )

// // ==============================
// // 📥 更新 Slack 配置（Webhook + 开关）
// // ==============================
// func UpdateSlackConfig(enabled bool, webhook string) error {
// 	return configmap.SetSlackConfig(enabled, webhook)
// }

// // ==============================
// // 📥 更新 Webhook 配置（CI/CD Webhook 开关）
// // ==============================
// func UpdateWebhookEnabled(enabled bool) error {
// 	return configmap.SetWebhookEnabled(enabled)
// }

// // ==============================
// // 📥 更新邮件配置（全字段 + 多人）
// // ==============================
// func UpdateMailConfig(enabled bool, username, password, from string, to []string) error {
// 	return configmap.SetMailConfig(enabled, username, password, from, to)
// }

// // ==============================
// // 📤 查询当前 ConfigMap（全部字段）
// // ==============================
// func GetCurrentAlertConfig() (map[string]string, error) {
// 	return configmap.GetCurrentConfigMap()
// }

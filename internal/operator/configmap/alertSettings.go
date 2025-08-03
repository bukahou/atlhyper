package configmap

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"strings"

// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// 	"NeuroController/internal/utils"
// )

// // ✅ 读取当前 Pod 所在命名空间（K8s Downward API 挂载）
// func getCurrentNamespace() string {
// 	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
// 	if err != nil {
// 		return "default"
// 	}
// 	return strings.TrimSpace(string(data))
// }

// const cmName = "neurocontroller-config"

// // ==============================================
// // ✅ 获取当前 ConfigMap 配置（用于 GET 显示）
// // ==============================================
// func GetCurrentConfigMap() (map[string]string, error) {
// 	client := utils.GetCoreClient()
// 	ns := getCurrentNamespace()
// 	ctx := context.TODO()

// 	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, cmName, metav1.GetOptions{})
// 	if err != nil {
// 		return nil, fmt.Errorf("❌ 获取 ConfigMap 失败: %w", err)
// 	}

// 	// 返回副本防止外部修改原始引用
// 	result := make(map[string]string)
// 	for k, v := range cm.Data {
// 		result[k] = v
// 	}
// 	return result, nil
// }

// // ==============================================
// // ✅ 通用更新函数：支持批量字段更新
// // ==============================================
// func UpdateConfigMapFields(fields map[string]string) error {
// 	client := utils.GetCoreClient()
// 	ns := getCurrentNamespace()
// 	ctx := context.TODO()

// 	cm, err := client.CoreV1().ConfigMaps(ns).Get(ctx, cmName, metav1.GetOptions{})
// 	if err != nil {
// 		return fmt.Errorf("❌ 获取 ConfigMap 失败: %w", err)
// 	}

// 	if cm.Data == nil {
// 		cm.Data = map[string]string{}
// 	}
// 	for k, v := range fields {
// 		cm.Data[k] = v
// 	}

// 	_, err = client.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
// 	if err != nil {
// 		return fmt.Errorf("❌ 更新 ConfigMap 字段失败: %w", err)
// 	}
// 	return nil
// }

// // ==============================================
// // ✅ ① 更新 Slack 告警配置
// // ==============================================
// func SetSlackConfig(enabled bool, webhook string) error {
// 	return UpdateConfigMapFields(map[string]string{
// 		"ENABLE_SLACK_ALERT":  boolToStr(enabled),
// 		"SLACK_WEBHOOK_URL":   webhook,
// 	})
// }

// // ==============================================
// // ✅ ② 更新 CI/CD Webhook 开关
// // ==============================================
// func SetWebhookEnabled(enabled bool) error {
// 	return UpdateConfigMapFields(map[string]string{
// 		"ENABLE_WEBHOOK_SERVER": boolToStr(enabled),
// 	})
// }

// // ==============================================
// // ✅ ③ 更新邮件配置（含多人收件 + 开关）
// // ==============================================
// func SetMailConfig(enabled bool, username, password, from string, to []string) error {
// 	return UpdateConfigMapFields(map[string]string{
// 		"ENABLE_EMAIL_ALERT": boolToStr(enabled),
// 		"MAIL_USERNAME":      username,
// 		"MAIL_PASSWORD":      password,
// 		"MAIL_FROM":          from,
// 		"MAIL_TO":            strings.Join(to, ","),
// 	})
// }

// // ==============================================
// // ✅ 工具函数：布尔值转字符串
// // ==============================================
// func boolToStr(b bool) string {
// 	if b {
// 		return "true"
// 	}
// 	return "false"
// }

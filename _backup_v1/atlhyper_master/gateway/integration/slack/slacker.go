package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ✅ SendSlackAlert 发送 BlockKit 格式的消息到 Slack Webhook
//
// 参数：
//   - payload: 已构建好的 BlockKit JSON 对象（通过 BuildSlackBlockFromAlert 构造）
//
// 返回：
//   - error: 若发送失败则返回错误信息，否则返回 nil
func SendSlackAlert(webhookURL string, payload map[string]interface{}) error {

	if strings.TrimSpace(webhookURL) == "" {
		return fmt.Errorf("Slack Webhook 未配置")
	}

	// ✅ 将消息体编码为 JSON 格式
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON 编码失败: %v", err)
	}

	// ✅ 构造 HTTP POST 请求（目标为 Slack Webhook URL）
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("构造请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json") // 设置请求头为 JSON

	// ✅ 使用默认 HTTP 客户端执行请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Slack 请求失败: %v", err)
	}
	defer resp.Body.Close()

	// ✅ 检查 Slack 响应状态码（2xx 为成功）
	if resp.StatusCode >= 300 {
		return fmt.Errorf("Slack 返回异常状态码: %d", resp.StatusCode)
	}

	// ✅ 成功返回
	return nil
}

// atlhyper_master_v2/notifier/channel/redact.go
// 凭据脱敏：发送失败的错误信息会写进日志与送达记录，里面不能有 Webhook 地址或密码
package channel

import (
	"net/url"
	"sort"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
)

// redactedMark 凭据被替换成的占位文字
const redactedMark = "[已隐藏]"

// minSecretLength 短于这个长度的串不当作凭据替换：把单个字符之类的短串全部替换只会打碎正常文字
const minSecretLength = 4

// ScrubSecrets 把文本里出现的凭据换成占位文字。
// 长的先替换，免得短凭据先命中、把长凭据打碎后留下残片
func ScrubSecrets(text string, secrets ...string) string {
	ordered := make([]string, 0, len(secrets))
	for _, secret := range secrets {
		if len(secret) >= minSecretLength {
			ordered = append(ordered, secret)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	for _, secret := range ordered {
		text = strings.ReplaceAll(text, secret, redactedMark)
	}
	return text
}

// ReceiverSecrets 一个接收人的配置里不得外泄的串。
//
// Slack 的 Webhook 地址整体就是凭据；除完整地址外，主机与路径也单独列出 ——
// 底层网络库的报错常常只带其中一段（如 "lookup <主机>"、"dial tcp <主机:端口>"）。
// 邮件接收人自身没有凭据：SMTP 密码在全局发件设置里，由 ReceiverSender 发送时一并去掉
func ReceiverSecrets(receiver *database.NotifyReceiver) []string {
	if receiver.Slack != nil && receiver.Slack.WebhookURL != "" {
		return webhookSecrets(receiver.Slack.WebhookURL)
	}
	return nil
}

// webhookSecrets Webhook 地址及其各段
func webhookSecrets(webhookURL string) []string {
	secrets := []string{webhookURL}
	parsed, err := url.Parse(webhookURL)
	if err != nil {
		return secrets
	}
	secrets = append(secrets, parsed.Host, parsed.Hostname())
	if len(parsed.Path) > 1 {
		secrets = append(secrets, parsed.Path)
		// 路径的最后一段是令牌本身：单独列出，防止报错里只截了一部分路径
		if index := strings.LastIndex(parsed.Path, "/"); index >= 0 {
			secrets = append(secrets, parsed.Path[index+1:])
		}
	}
	if parsed.RawQuery != "" {
		secrets = append(secrets, parsed.RawQuery)
	}
	return secrets
}

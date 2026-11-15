// atlhyper_master_v2/model/notify_validation.go
// 通知设置的输入校验（纯函数）。
//
// ⛔ 错误文案会原样返回给调用方：只说哪一项不对，不回显用户填的 Webhook 地址或密码
package model

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// ErrInvalidNotifyInput 通知设置的输入不合规；接口据此返回 400
var ErrInvalidNotifyInput = errors.New("通知设置不合规")

const (
	maxNotifyNameRunes      = 64
	maxNotifyCommentRunes   = 256
	maxNotifyMatcherLength  = 253 // 集群 ID、命名空间、主体键的长度上限
	maxNotifyWebhookLength  = 2048
	maxNotifySecretLength   = 512
	maxNotifyRecipients     = 50
	maxNotifyRuleClusters   = 100
	maxNotifySilencePeriod  = 30 * 24 * time.Hour
	minNotifyFlapThreshold  = 0
	maxNotifyFlapThreshold  = 20
	minNotifyGroupWait      = 0
	maxNotifyGroupWait      = 600
	minNotifyGroupInterval  = 30
	maxNotifyGroupInterval  = 86400
	minNotifyRepeatInterval = 300
	maxNotifyRepeatInterval = 604800
)

func invalidNotify(format string, args ...any) error {
	return fmt.Errorf("%w：%s", ErrInvalidNotifyInput, fmt.Sprintf(format, args...))
}

// ValidateNotifyPolicy 校验策略：九个字段都必须给出，取值在范围内。
// 默认接收人是否都存在由 service 对照库里的数据检查
func ValidateNotifyPolicy(req *UpdateNotifyPolicyRequest) error {
	switch {
	case req.IncidentAlertsEnabled == nil:
		return invalidNotify("缺少 incidentAlertsEnabled")
	case req.MinSeverity == nil:
		return invalidNotify("缺少 minSeverity")
	case req.NotifyResolved == nil:
		return invalidNotify("缺少 notifyResolved")
	case req.WarningFlapThreshold == nil:
		return invalidNotify("缺少 warningFlapThreshold")
	case req.GroupWaitSeconds == nil:
		return invalidNotify("缺少 groupWaitSeconds")
	case req.GroupIntervalSeconds == nil:
		return invalidNotify("缺少 groupIntervalSeconds")
	case req.RepeatIntervalSeconds == nil:
		return invalidNotify("缺少 repeatIntervalSeconds")
	case req.K8sEventAlertsEnabled == nil:
		return invalidNotify("缺少 k8sEventAlertsEnabled")
	case req.DefaultReceiverIDs == nil:
		return invalidNotify("缺少 defaultReceiverIds")
	}
	if err := validateNotifySeverity("minSeverity", *req.MinSeverity); err != nil {
		return err
	}
	if err := validateNotifyRange("warningFlapThreshold", *req.WarningFlapThreshold, minNotifyFlapThreshold, maxNotifyFlapThreshold); err != nil {
		return err
	}
	if err := validateNotifyTiming(req.GroupWaitSeconds, req.GroupIntervalSeconds, req.RepeatIntervalSeconds); err != nil {
		return err
	}
	return validateNotifyReceiverIDs("defaultReceiverIds", *req.DefaultReceiverIDs)
}

// validateNotifyTiming 三个时间参数的取值范围（策略与规则相同）；nil 表示没给，不校验
func validateNotifyTiming(groupWait, groupInterval, repeatInterval *int) error {
	if groupWait != nil {
		if err := validateNotifyRange("groupWaitSeconds", *groupWait, minNotifyGroupWait, maxNotifyGroupWait); err != nil {
			return err
		}
	}
	if groupInterval != nil {
		if err := validateNotifyRange("groupIntervalSeconds", *groupInterval, minNotifyGroupInterval, maxNotifyGroupInterval); err != nil {
			return err
		}
	}
	if repeatInterval != nil {
		if err := validateNotifyRange("repeatIntervalSeconds", *repeatInterval, minNotifyRepeatInterval, maxNotifyRepeatInterval); err != nil {
			return err
		}
	}
	return nil
}

func validateNotifyRange(field string, value, min, max int) error {
	if value < min || value > max {
		return invalidNotify("%s 须在 %d–%d 之间", field, min, max)
	}
	return nil
}

func validateNotifySeverity(field, value string) error {
	if _, known := alert.ParseSeverity(value); !known {
		return invalidNotify("%s 须为 low / medium / high / critical 之一", field)
	}
	return nil
}

func validateNotifySource(field, value string) error {
	switch alert.Source(value) {
	case alert.SourceIncident, alert.SourceHeartbeat, alert.SourceK8sEvent:
		return nil
	}
	return invalidNotify("%s 须为 incident / heartbeat / k8s_event 之一", field)
}

// validateNotifyReceiverIDs id 须为正数且不重复
func validateNotifyReceiverIDs(field string, ids []int64) error {
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return invalidNotify("%s 里的接收人 id 须为正数", field)
		}
		if seen[id] {
			return invalidNotify("%s 里的接收人不能重复", field)
		}
		seen[id] = true
	}
	return nil
}

// validateNotifyName 名称必填、有字数上限、不含控制字符
func validateNotifyName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return invalidNotify("名称不能为空")
	case utf8.RuneCountInString(name) > maxNotifyNameRunes:
		return invalidNotify("名称不能超过 %d 个字", maxNotifyNameRunes)
	case containsControlRune(name):
		return invalidNotify("名称不能包含控制字符")
	}
	return nil
}

// ValidateNotifyReceiver 校验接收人。
//
// requireSecret 为真（新建，或修改时换了类型）时凭据必填；为假（修改）时凭据留空 = 保持原值，
// 不带 slack / email 段 = 整段配置保持原值
func ValidateNotifyReceiver(req *NotifyReceiverRequest, requireSecret bool) error {
	if err := validateNotifyName(req.Name); err != nil {
		return err
	}
	switch req.Type {
	case database.NotifyReceiverSlack:
		return validateNotifySlack(req.Slack, requireSecret)
	case database.NotifyReceiverEmail:
		return validateNotifyEmail(req.Email, requireSecret)
	default:
		return invalidNotify("接收人类型须为 slack 或 email")
	}
}

func validateNotifySlack(input *NotifySlackInput, requireSecret bool) error {
	if input == nil || input.WebhookURL == "" {
		if requireSecret {
			return invalidNotify("Slack 接收人必须填写 Webhook 地址")
		}
		return nil
	}
	if len(input.WebhookURL) > maxNotifyWebhookLength {
		return invalidNotify("Webhook 地址过长")
	}
	// 解析失败的错误原文会带上地址本身：不透出
	parsed, err := url.Parse(input.WebhookURL)
	if err != nil || parsed.Host == "" || containsControlRune(input.WebhookURL) {
		return invalidNotify("Webhook 地址不是合法的 URL")
	}
	if parsed.Scheme != "https" {
		return invalidNotify("Webhook 地址必须以 https:// 开头")
	}
	if parsed.User != nil {
		return invalidNotify("Webhook 地址里不能带账号密码")
	}
	return nil
}

func validateNotifyEmail(input *NotifyEmailInput, requireSecret bool) error {
	if input == nil {
		if requireSecret {
			return invalidNotify("邮件接收人必须填写 SMTP 配置")
		}
		return nil
	}
	switch {
	case input.SMTPHost == "":
		return invalidNotify("SMTP 主机不能为空")
	case len(input.SMTPHost) > maxNotifyMatcherLength || strings.ContainsAny(input.SMTPHost, " \t/") || containsControlRune(input.SMTPHost):
		return invalidNotify("SMTP 主机不合法")
	case input.SMTPPort < 1 || input.SMTPPort > 65535:
		return invalidNotify("SMTP 端口须在 1–65535 之间")
	case input.SMTPUser == "":
		return invalidNotify("SMTP 账号不能为空")
	case len(input.SMTPUser) > maxNotifyMatcherLength || containsControlRune(input.SMTPUser):
		return invalidNotify("SMTP 账号不合法")
	case requireSecret && input.SMTPPassword == "":
		return invalidNotify("邮件接收人必须填写 SMTP 密码")
	case len(input.SMTPPassword) > maxNotifySecretLength:
		return invalidNotify("SMTP 密码过长")
	case len(input.ToAddresses) == 0:
		return invalidNotify("至少需要一个收件人")
	case len(input.ToAddresses) > maxNotifyRecipients:
		return invalidNotify("收件人不能超过 %d 个", maxNotifyRecipients)
	}
	if input.FromAddress != "" && !isPlainMailAddress(input.FromAddress) {
		return invalidNotify("发件人地址不是合法的邮箱")
	}
	for _, recipient := range input.ToAddresses {
		if !isPlainMailAddress(recipient) {
			return invalidNotify("收件人地址不是合法的邮箱")
		}
	}
	return nil
}

// isPlainMailAddress 只接受「账号@域名」的纯地址：不带显示名、不带换行（换行能伪造出新的邮件头）
func isPlainMailAddress(address string) bool {
	if len(address) > maxNotifyMatcherLength || containsControlRune(address) {
		return false
	}
	parsed, err := mail.ParseAddress(address)
	return err == nil && parsed.Name == "" && parsed.Address == address
}

// ValidateNotifyRule 校验路由规则。接收人是否都存在由 service 对照库里的数据检查。
// receiverIds 可以为空：命中这条规则的告警不发给任何人
func ValidateNotifyRule(req *NotifyRuleRequest) error {
	if err := validateNotifyName(req.Name); err != nil {
		return err
	}
	if req.MinSeverity != "" {
		if err := validateNotifySeverity("minSeverity", req.MinSeverity); err != nil {
			return err
		}
	}
	if len(req.ClusterIDs) > maxNotifyRuleClusters {
		return invalidNotify("clusterIds 不能超过 %d 个", maxNotifyRuleClusters)
	}
	for _, clusterID := range req.ClusterIDs {
		if clusterID == "" {
			return invalidNotify("clusterIds 里不能有空串")
		}
		if err := validateNotifyMatcher("clusterIds", clusterID); err != nil {
			return err
		}
	}
	for _, source := range req.Sources {
		if err := validateNotifySource("sources", source); err != nil {
			return err
		}
	}
	if err := validateNotifyReceiverIDs("receiverIds", req.ReceiverIDs); err != nil {
		return err
	}
	return validateNotifyTiming(req.GroupWaitSeconds, req.GroupIntervalSeconds, req.RepeatIntervalSeconds)
}

// validateNotifyMatcher 匹配条件（集群 ID、命名空间、主体键）：有长度上限、不含控制字符
func validateNotifyMatcher(field, value string) error {
	if len(value) > maxNotifyMatcherLength {
		return invalidNotify("%s 过长", field)
	}
	if containsControlRune(value) {
		return invalidNotify("%s 不能包含控制字符", field)
	}
	return nil
}

// ValidateNotifyRuleMatch 校验试匹配的输入
func ValidateNotifyRuleMatch(req *NotifyRuleMatchRequest) error {
	if err := validateNotifyMatcher("clusterId", req.ClusterID); err != nil {
		return err
	}
	if err := validateNotifySource("source", req.Source); err != nil {
		return err
	}
	return validateNotifySeverity("severity", req.Severity)
}

// ParseNotifySilence 校验新建静默的输入并解析出起止时间。
//
// 开始时间留空 = 现在；结束须晚于开始、晚于现在；时长不超过 30 天；说明必填、不超过 256 字
func ParseNotifySilence(req *CreateNotifySilenceRequest, now time.Time) (startsAt, endsAt time.Time, err error) {
	fail := func(format string, args ...any) (time.Time, time.Time, error) {
		return time.Time{}, time.Time{}, invalidNotify(format, args...)
	}
	for field, value := range map[string]string{"clusterId": req.ClusterID, "namespace": req.Namespace, "subject": req.Subject} {
		if err := validateNotifyMatcher(field, value); err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	comment := strings.TrimSpace(req.Comment)
	switch {
	case comment == "":
		return fail("说明不能为空")
	case utf8.RuneCountInString(comment) > maxNotifyCommentRunes:
		return fail("说明不能超过 %d 个字", maxNotifyCommentRunes)
	case containsControlRune(comment):
		return fail("说明不能包含控制字符")
	}

	startsAt = now
	if req.StartsAt != "" {
		if startsAt, err = time.Parse(time.RFC3339, req.StartsAt); err != nil {
			return fail("startsAt 须为 RFC 3339 时间")
		}
	}
	if endsAt, err = time.Parse(time.RFC3339, req.EndsAt); err != nil {
		return fail("endsAt 须为 RFC 3339 时间")
	}
	switch {
	case !endsAt.After(startsAt):
		return fail("结束时间须晚于开始时间")
	case !endsAt.After(now):
		return fail("结束时间须晚于现在")
	case endsAt.Sub(startsAt) > maxNotifySilencePeriod:
		return fail("静默时长不能超过 30 天")
	}
	return startsAt.UTC(), endsAt.UTC(), nil
}

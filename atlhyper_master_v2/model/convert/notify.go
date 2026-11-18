// atlhyper_master_v2/model/convert/notify.go
// database.Notify* → model.Notify* 转换函数
package convert

import (
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/silence"
)

// NotifyPolicy 通知策略
func NotifyPolicy(src *database.NotifyPolicy) model.NotifyPolicy {
	return model.NotifyPolicy{
		IncidentAlertsEnabled: src.IncidentAlertsEnabled,
		MinSeverity:           src.MinSeverity,
		NotifyResolved:        src.NotifyResolved,
		WarningFlapThreshold:  src.WarningFlapThreshold,
		GroupWaitSeconds:      src.GroupWaitSeconds,
		GroupIntervalSeconds:  src.GroupIntervalSeconds,
		RepeatIntervalSeconds: src.RepeatIntervalSeconds,
		K8sEventAlertsEnabled: src.K8sEventAlertsEnabled,
		DefaultReceiverIDs:    nonNilInt64s(src.DefaultReceiverIDs),
	}
}

// NotifyAlertItem 一条告警。主体与事件号分别取自标签与详情
func NotifyAlertItem(src *database.NotifyAlert) model.NotifyAlert {
	return model.NotifyAlert{
		Fingerprint:    src.Fingerprint,
		Source:         src.Source,
		ClusterID:      src.ClusterID,
		Severity:       src.Severity,
		Status:         src.Status,
		Kind:           src.Kind,
		Title:          src.Title,
		Summary:        src.Summary,
		Subject:        src.Labels[alert.LabelSubject],
		IncidentID:     src.Details[alert.DetailIncidentID],
		StartsAt:       formatUTC(src.StartsAt),
		EndsAt:         formatOptionalUTC(src.EndsAt),
		LastNotifiedAt: formatOptionalUTC(src.LastNotifiedAt),
		NotifyCount:    src.NotifyCount,
		SuppressedBy:   src.SuppressedBy,
		SilencedBy:     src.SilencedBy,
	}
}

// NotifyAlertItems 批量转换
func NotifyAlertItems(src []*database.NotifyAlert) []model.NotifyAlert {
	result := make([]model.NotifyAlert, len(src))
	for i := range src {
		result[i] = NotifyAlertItem(src[i])
	}
	return result
}

// NotifyDeliveryItem 一条送达记录
func NotifyDeliveryItem(src *database.NotifyDelivery) model.NotifyDelivery {
	return model.NotifyDelivery{
		ID:           src.ID,
		BatchID:      src.BatchID,
		Fingerprint:  src.Fingerprint,
		AlertTitle:   src.AlertTitle,
		ReceiverID:   src.ReceiverID,
		ReceiverName: src.ReceiverName,
		ReceiverType: src.ReceiverType,
		Kind:         src.Kind,
		Status:       src.Status,
		Attempts:     src.Attempts,
		LastError:    src.LastError,
		CreatedAt:    formatUTC(src.CreatedAt),
		UpdatedAt:    formatUTC(src.UpdatedAt),
	}
}

// NotifyDeliveryItems 批量转换
func NotifyDeliveryItems(src []*database.NotifyDelivery) []model.NotifyDelivery {
	result := make([]model.NotifyDelivery, len(src))
	for i := range src {
		result[i] = NotifyDeliveryItem(src[i])
	}
	return result
}

// NotifyReceiverItem 一个接收人。
//
// ⛔ 凭据不写入结果：Webhook 地址只用来判断「是否已配置」
func NotifyReceiverItem(src *database.NotifyReceiver) model.NotifyReceiver {
	item := model.NotifyReceiver{
		ID:        src.ID,
		Name:      src.Name,
		Type:      src.Type,
		Enabled:   src.Enabled,
		CreatedAt: formatUTC(src.CreatedAt),
		UpdatedAt: formatUTC(src.UpdatedAt),
	}
	switch src.Type {
	case database.NotifyReceiverSlack:
		item.Slack = &model.NotifySlackConfig{WebhookConfigured: src.Slack != nil && src.Slack.WebhookURL != ""}
	case database.NotifyReceiverEmail:
		email := &model.NotifyEmailConfig{ToAddresses: []string{}}
		if src.Email != nil {
			email.ToAddresses = nonNilStrings(src.Email.ToAddresses)
		}
		item.Email = email
	}
	return item
}

// NotifySMTPView 全局发件设置。src 为 nil（从没保存过）时返回空设置。
//
// ⛔ 密码不写入结果，只用来判断「是否已配置」
func NotifySMTPView(src *database.NotifySMTP) model.NotifySMTP {
	if src == nil {
		return model.NotifySMTP{}
	}
	updatedAt := formatUTC(src.UpdatedAt)
	return model.NotifySMTP{
		Configured:         src.Configured(),
		Host:               src.Host,
		Port:               src.Port,
		User:               src.User,
		TLS:                src.TLS,
		FromAddress:        src.FromAddress,
		PasswordConfigured: src.Password != "",
		UpdatedAt:          &updatedAt,
	}
}

// NotifyReceiverItems 批量转换
func NotifyReceiverItems(src []*database.NotifyReceiver) []model.NotifyReceiver {
	result := make([]model.NotifyReceiver, len(src))
	for i := range src {
		result[i] = NotifyReceiverItem(src[i])
	}
	return result
}

// NotifyReceiverRef 接收人的简要信息（试匹配结果用）
func NotifyReceiverRef(src *database.NotifyReceiver) model.NotifyReceiverRef {
	return model.NotifyReceiverRef{ID: src.ID, Name: src.Name, Type: src.Type, Enabled: src.Enabled}
}

// NotifyRuleItem 一条路由规则
func NotifyRuleItem(src *database.NotifyRule) model.NotifyRule {
	return model.NotifyRule{
		ID:                    src.ID,
		Name:                  src.Name,
		Position:              src.Position,
		Enabled:               src.Enabled,
		ClusterIDs:            nonNilStrings(src.ClusterIDs),
		Sources:               nonNilStrings(src.Sources),
		MinSeverity:           src.MinSeverity,
		ReceiverIDs:           nonNilInt64s(src.ReceiverIDs),
		GroupWaitSeconds:      src.GroupWaitSeconds,
		GroupIntervalSeconds:  src.GroupIntervalSeconds,
		RepeatIntervalSeconds: src.RepeatIntervalSeconds,
		CreatedAt:             formatUTC(src.CreatedAt),
		UpdatedAt:             formatUTC(src.UpdatedAt),
	}
}

// NotifyRuleItems 批量转换
func NotifyRuleItems(src []*database.NotifyRule) []model.NotifyRule {
	result := make([]model.NotifyRule, len(src))
	for i := range src {
		result[i] = NotifyRuleItem(src[i])
	}
	return result
}

// NotifySilenceItem 一条静默；状态由给定时刻算出
func NotifySilenceItem(src *database.NotifySilence, now time.Time) model.NotifySilence {
	return model.NotifySilence{
		ID:        src.ID,
		ClusterID: src.ClusterID,
		Namespace: src.Namespace,
		Subject:   src.Subject,
		StartsAt:  formatUTC(src.StartsAt),
		EndsAt:    formatUTC(src.EndsAt),
		Comment:   src.Comment,
		CreatedBy: src.CreatedBy,
		CreatedAt: formatUTC(src.CreatedAt),
		State:     silence.State(src, now),
	}
}

// NotifySilenceItems 批量转换
func NotifySilenceItems(src []*database.NotifySilence, now time.Time) []model.NotifySilence {
	result := make([]model.NotifySilence, len(src))
	for i := range src {
		result[i] = NotifySilenceItem(src[i], now)
	}
	return result
}

// formatOptionalUTC 可空时间 → RFC 3339（UTC）；没有时返回空串
func formatOptionalUTC(at *time.Time) string {
	if at == nil {
		return ""
	}
	return formatUTC(*at)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilInt64s(values []int64) []int64 {
	if values == nil {
		return []int64{}
	}
	return values
}

// atlhyper_master_v2/service/operations/notify.go
// OpsNotify 实现 — 通知设置的写入：策略、接收人、路由规则、静默
//
// ⛔ 接收人的凭据（Webhook 地址、SMTP 密码）不写日志；校验错误的文案不回显它们
package operations

import (
	"context"
	"fmt"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/common/logger"
)

var notifyLog = logger.Module("NotifySettings")

// NotifyRepos 通知设置写入用到的仓库
type NotifyRepos struct {
	Policy    database.NotifyPolicyRepository
	Receivers database.NotifyReceiverRepository
	Rules     database.NotifyRuleRepository
	Silences  database.NotifySilenceRepository
}

// NotifyService 通知设置写入操作
type NotifyService struct {
	repos NotifyRepos
	now   func() time.Time // 时钟可替换，便于测试
}

// NewNotifyService 创建 NotifyService
func NewNotifyService(repos NotifyRepos) *NotifyService {
	return &NotifyService{repos: repos, now: time.Now}
}

// ==================== 策略 ====================

// UpdateNotifyPolicy 整体替换通知策略。
// 输入不合规、或默认接收人里有不存在的 → model.ErrInvalidNotifyInput
func (s *NotifyService) UpdateNotifyPolicy(ctx context.Context, req *model.UpdateNotifyPolicyRequest) error {
	if err := model.ValidateNotifyPolicy(req); err != nil {
		return err
	}
	if err := s.requireReceiversExist(ctx, "defaultReceiverIds", *req.DefaultReceiverIDs); err != nil {
		return err
	}
	policy := &database.NotifyPolicy{
		IncidentAlertsEnabled: *req.IncidentAlertsEnabled,
		MinSeverity:           *req.MinSeverity,
		NotifyResolved:        *req.NotifyResolved,
		WarningFlapThreshold:  *req.WarningFlapThreshold,
		GroupWaitSeconds:      *req.GroupWaitSeconds,
		GroupIntervalSeconds:  *req.GroupIntervalSeconds,
		RepeatIntervalSeconds: *req.RepeatIntervalSeconds,
		K8sEventAlertsEnabled: *req.K8sEventAlertsEnabled,
		DefaultReceiverIDs:    *req.DefaultReceiverIDs,
	}
	if err := s.repos.Policy.Save(ctx, policy); err != nil {
		return err
	}
	notifyLog.Info("通知策略已更新")
	return nil
}

// requireReceiversExist 给出的接收人 id 必须都存在
func (s *NotifyService) requireReceiversExist(ctx context.Context, field string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	receivers, err := s.repos.Receivers.List(ctx)
	if err != nil {
		return err
	}
	existing := make(map[int64]bool, len(receivers))
	for _, receiver := range receivers {
		existing[receiver.ID] = true
	}
	for _, id := range ids {
		if !existing[id] {
			return fmt.Errorf("%w：%s 里的接收人 %d 不存在", model.ErrInvalidNotifyInput, field, id)
		}
	}
	return nil
}

// ==================== 接收人 ====================

// CreateNotifyReceiver 新建接收人，返回它的 id。没传 enabled 时为启用。
// 输入不合规 → model.ErrInvalidNotifyInput；名称重复 → database.ErrNotifyReceiverNameTaken
func (s *NotifyService) CreateNotifyReceiver(ctx context.Context, req *model.NotifyReceiverRequest) (int64, error) {
	if err := model.ValidateNotifyReceiver(req, true); err != nil {
		return 0, err
	}
	receiver := &database.NotifyReceiver{
		Name:    strings.TrimSpace(req.Name),
		Type:    req.Type,
		Enabled: req.Enabled == nil || *req.Enabled,
	}
	applyReceiverConfig(receiver, req)
	if err := s.repos.Receivers.Create(ctx, receiver); err != nil {
		return 0, err
	}
	notifyLog.Info("接收人已创建", "id", receiver.ID, "type", receiver.Type)
	return receiver.ID, nil
}

// UpdateNotifyReceiver 修改接收人。
//
// 凭据（webhookUrl / smtpPassword）留空 = 保持原值；不带 slack / email 段 = 整段配置保持原值；
// 没传 enabled = 保持原值。换了类型时旧配置作废，必须给出新类型的完整配置。
// 不存在 → database.ErrNotifyReceiverNotFound；名称重复 → database.ErrNotifyReceiverNameTaken
func (s *NotifyService) UpdateNotifyReceiver(ctx context.Context, id int64, req *model.NotifyReceiverRequest) error {
	existing, err := s.repos.Receivers.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return database.ErrNotifyReceiverNotFound
	}

	typeChanged := req.Type != existing.Type
	if err := model.ValidateNotifyReceiver(req, typeChanged); err != nil {
		return err
	}
	if typeChanged {
		existing.Slack, existing.Email = nil, nil
	}
	existing.Name, existing.Type = strings.TrimSpace(req.Name), req.Type
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	applyReceiverConfig(existing, req)

	if err := s.repos.Receivers.Update(ctx, existing); err != nil {
		return err
	}
	notifyLog.Info("接收人已更新", "id", id, "type", existing.Type)
	return nil
}

// applyReceiverConfig 把请求里的配置合并到接收人上：凭据留空时保留接收人上已有的值
func applyReceiverConfig(receiver *database.NotifyReceiver, req *model.NotifyReceiverRequest) {
	switch req.Type {
	case database.NotifyReceiverSlack:
		if req.Slack == nil || req.Slack.WebhookURL == "" {
			return
		}
		receiver.Slack = &database.SlackConfig{WebhookURL: req.Slack.WebhookURL}

	case database.NotifyReceiverEmail:
		if req.Email == nil {
			return
		}
		password := req.Email.SMTPPassword
		if password == "" && receiver.Email != nil {
			password = receiver.Email.SMTPPassword
		}
		recipients := make([]string, len(req.Email.ToAddresses))
		copy(recipients, req.Email.ToAddresses)
		receiver.Email = &database.EmailConfig{
			SMTPHost:     req.Email.SMTPHost,
			SMTPPort:     req.Email.SMTPPort,
			SMTPUser:     req.Email.SMTPUser,
			SMTPPassword: password,
			SMTPTLS:      req.Email.SMTPTLS,
			FromAddress:  req.Email.FromAddress,
			ToAddresses:  recipients,
		}
	}
}

// DeleteNotifyReceiver 删除接收人。
// 仍被路由规则或默认策略引用 → *model.NotifyReceiverInUseError（写明被谁引用）；
// 不存在 → database.ErrNotifyReceiverNotFound
func (s *NotifyService) DeleteNotifyReceiver(ctx context.Context, id int64) error {
	rules, err := s.repos.Rules.List(ctx)
	if err != nil {
		return err
	}
	policy, err := s.repos.Policy.Get(ctx)
	if err != nil {
		return err
	}
	var references []string
	for _, rule := range rules {
		if containsReceiver(rule.ReceiverIDs, id) {
			references = append(references, "路由规则「"+rule.Name+"」")
		}
	}
	if containsReceiver(policy.DefaultReceiverIDs, id) {
		references = append(references, "默认策略")
	}
	if len(references) > 0 {
		return &model.NotifyReceiverInUseError{References: references}
	}

	if err := s.repos.Receivers.Delete(ctx, id); err != nil {
		return err
	}
	notifyLog.Info("接收人已删除", "id", id)
	return nil
}

func containsReceiver(ids []int64, wanted int64) bool {
	for _, id := range ids {
		if id == wanted {
			return true
		}
	}
	return false
}

// ==================== 路由规则 ====================

// CreateNotifyRule 新建路由规则（排在最后），返回它的 id。没传 enabled 时为启用；严重度没填 = low。
// 输入不合规、或接收人里有不存在的 → model.ErrInvalidNotifyInput
func (s *NotifyService) CreateNotifyRule(ctx context.Context, req *model.NotifyRuleRequest) (int64, error) {
	if err := s.validateRule(ctx, req); err != nil {
		return 0, err
	}
	rule := &database.NotifyRule{Enabled: req.Enabled == nil || *req.Enabled}
	applyRuleFields(rule, req)
	if err := s.repos.Rules.Create(ctx, rule); err != nil {
		return 0, err
	}
	notifyLog.Info("路由规则已创建", "id", rule.ID, "position", rule.Position)
	return rule.ID, nil
}

// UpdateNotifyRule 修改路由规则（不改位置）。没传 enabled = 保持原值。
// 不存在 → database.ErrNotifyRuleNotFound
func (s *NotifyService) UpdateNotifyRule(ctx context.Context, id int64, req *model.NotifyRuleRequest) error {
	existing, err := s.repos.Rules.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return database.ErrNotifyRuleNotFound
	}
	if err := s.validateRule(ctx, req); err != nil {
		return err
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	applyRuleFields(existing, req)
	if err := s.repos.Rules.Update(ctx, existing); err != nil {
		return err
	}
	notifyLog.Info("路由规则已更新", "id", id)
	return nil
}

func (s *NotifyService) validateRule(ctx context.Context, req *model.NotifyRuleRequest) error {
	if err := model.ValidateNotifyRule(req); err != nil {
		return err
	}
	return s.requireReceiversExist(ctx, "receiverIds", req.ReceiverIDs)
}

// applyRuleFields 把请求里除 enabled 之外的字段写到规则上
func applyRuleFields(rule *database.NotifyRule, req *model.NotifyRuleRequest) {
	rule.Name = strings.TrimSpace(req.Name)
	rule.ClusterIDs = append([]string{}, req.ClusterIDs...)
	rule.Sources = append([]string{}, req.Sources...)
	rule.ReceiverIDs = append([]int64{}, req.ReceiverIDs...)
	rule.MinSeverity = req.MinSeverity
	if rule.MinSeverity == "" {
		rule.MinSeverity = "low"
	}
	rule.GroupWaitSeconds = req.GroupWaitSeconds
	rule.GroupIntervalSeconds = req.GroupIntervalSeconds
	rule.RepeatIntervalSeconds = req.RepeatIntervalSeconds
}

// DeleteNotifyRule 删除路由规则。不存在 → database.ErrNotifyRuleNotFound
func (s *NotifyService) DeleteNotifyRule(ctx context.Context, id int64) error {
	if err := s.repos.Rules.Delete(ctx, id); err != nil {
		return err
	}
	notifyLog.Info("路由规则已删除", "id", id)
	return nil
}

// ReorderNotifyRules 按给定顺序重排规则。
// ids 不是全部规则的一个排列 → database.ErrNotifyRuleOrderMismatch
func (s *NotifyService) ReorderNotifyRules(ctx context.Context, ids []int64) error {
	if err := s.repos.Rules.Reorder(ctx, ids); err != nil {
		return err
	}
	notifyLog.Info("路由规则已重排", "count", len(ids))
	return nil
}

// ==================== 静默 ====================

// CreateNotifySilence 新建静默，返回它的 id。输入不合规 → model.ErrInvalidNotifyInput
func (s *NotifyService) CreateNotifySilence(ctx context.Context, req *model.CreateNotifySilenceRequest, createdBy string) (int64, error) {
	startsAt, endsAt, err := model.ParseNotifySilence(req, s.now())
	if err != nil {
		return 0, err
	}
	silence := &database.NotifySilence{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		Subject:   req.Subject,
		StartsAt:  startsAt,
		EndsAt:    endsAt,
		Comment:   strings.TrimSpace(req.Comment),
		CreatedBy: createdBy,
	}
	if err := s.repos.Silences.Create(ctx, silence); err != nil {
		return 0, err
	}
	notifyLog.Info("静默已创建", "id", silence.ID, "cluster", silence.ClusterID, "createdBy", createdBy)
	return silence.ID, nil
}

// ExpireNotifySilence 提前结束静默：结束时间置为现在。
//
// 还没开始的静默也可以结束（建错了撤销）—— 结束时间早于开始时间，它不会再生效。
// 已经结束的不改动。不存在 → database.ErrNotifySilenceNotFound
func (s *NotifyService) ExpireNotifySilence(ctx context.Context, id int64) error {
	existing, err := s.repos.Silences.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return database.ErrNotifySilenceNotFound
	}
	now := s.now()
	if !existing.EndsAt.After(now) {
		return nil
	}
	if err := s.repos.Silences.SetEndsAt(ctx, id, now); err != nil {
		return err
	}
	notifyLog.Info("静默已提前结束", "id", id)
	return nil
}

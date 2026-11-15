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

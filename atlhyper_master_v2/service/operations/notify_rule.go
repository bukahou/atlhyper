// atlhyper_master_v2/service/operations/notify_rule.go
// OpsNotify 实现（续）— 路由规则与静默的写入。策略与接收人见 notify.go
package operations

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

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

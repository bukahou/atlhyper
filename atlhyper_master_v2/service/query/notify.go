// atlhyper_master_v2/service/query/notify.go
// QueryNotify 实现 — 通知设置页的读取：策略、当前告警、送达记录、接收人、路由规则、静默、试匹配
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/model/convert"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/routing"
)

// 以下是各通知仓库的只读部分。查询层只拿读方法：写入在 service/operations，告警与送达记录由通知管道写

// NotifyPolicyReader 通知策略（只读）
type NotifyPolicyReader interface {
	Get(ctx context.Context) (*database.NotifyPolicy, error)
}

// NotifyAlertReader 告警（只读）
type NotifyAlertReader interface {
	List(ctx context.Context, opts database.NotifyAlertQueryOpts) ([]*database.NotifyAlert, error)
	Count(ctx context.Context, opts database.NotifyAlertQueryOpts) (int, error)
}

// NotifyDeliveryReader 送达记录（只读）
type NotifyDeliveryReader interface {
	List(ctx context.Context, opts database.NotifyDeliveryQueryOpts) ([]*database.NotifyDelivery, error)
	Count(ctx context.Context, opts database.NotifyDeliveryQueryOpts) (int, error)
}

// NotifyReceiverReader 接收人（只读）
type NotifyReceiverReader interface {
	GetByID(ctx context.Context, id int64) (*database.NotifyReceiver, error)
	List(ctx context.Context) ([]*database.NotifyReceiver, error)
}

// NotifyRuleReader 路由规则（只读）
type NotifyRuleReader interface {
	GetByID(ctx context.Context, id int64) (*database.NotifyRule, error)
	List(ctx context.Context) ([]*database.NotifyRule, error)
}

// NotifySilenceReader 静默（只读）
type NotifySilenceReader interface {
	GetByID(ctx context.Context, id int64) (*database.NotifySilence, error)
	List(ctx context.Context) ([]*database.NotifySilence, error)
}

// NotifySMTPReader 全局发件设置（只读）。没有设置时返回 (nil, nil)
type NotifySMTPReader interface {
	Get(ctx context.Context) (*database.NotifySMTP, error)
}

// NotifyRepos 通知查询所需的仓库集合，对应 QueryNotify 接口
type NotifyRepos struct {
	Policy     NotifyPolicyReader
	Alerts     NotifyAlertReader
	Deliveries NotifyDeliveryReader
	Receivers  NotifyReceiverReader
	Rules      NotifyRuleReader
	Silences   NotifySilenceReader
	SMTP       NotifySMTPReader
}

// GetNotifyPolicy 当前的通知策略
func (q *QueryService) GetNotifyPolicy(ctx context.Context) (*model.NotifyPolicy, error) {
	policy, err := q.notify.Policy.Get(ctx)
	if err != nil {
		return nil, err
	}
	item := convert.NotifyPolicy(policy)
	return &item, nil
}

// GetNotifySMTP 全局发件设置（不含密码）。从没保存过时返回空设置
func (q *QueryService) GetNotifySMTP(ctx context.Context) (*model.NotifySMTP, error) {
	settings, err := q.notify.SMTP.Get(ctx)
	if err != nil {
		return nil, err
	}
	view := convert.NotifySMTPView(settings)
	return &view, nil
}

// ListNotifyAlerts 告警列表（开始时间新的在前）与符合条件的总数
func (q *QueryService) ListNotifyAlerts(ctx context.Context, opts model.NotifyAlertQuery) ([]model.NotifyAlert, int, error) {
	filter := database.NotifyAlertQueryOpts{Status: opts.Status, ClusterID: opts.ClusterID, Limit: opts.Limit, Offset: opts.Offset}
	rows, err := q.notify.Alerts.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := q.notify.Alerts.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return convert.NotifyAlertItems(rows), total, nil
}

// ListNotifyDeliveries 送达记录（新的在前）与符合条件的总数
func (q *QueryService) ListNotifyDeliveries(ctx context.Context, opts model.NotifyDeliveryQuery) ([]model.NotifyDelivery, int, error) {
	filter := database.NotifyDeliveryQueryOpts{Fingerprint: opts.Fingerprint, Status: opts.Status, Limit: opts.Limit, Offset: opts.Offset}
	rows, err := q.notify.Deliveries.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total, err := q.notify.Deliveries.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return convert.NotifyDeliveryItems(rows), total, nil
}

// ListNotifyReceivers 全部接收人。凭据不在结果里
func (q *QueryService) ListNotifyReceivers(ctx context.Context) ([]model.NotifyReceiver, error) {
	receivers, err := q.notify.Receivers.List(ctx)
	if err != nil {
		return nil, err
	}
	return convert.NotifyReceiverItems(receivers), nil
}

// GetNotifyReceiver 单个接收人；不存在时返回 (nil, nil)
func (q *QueryService) GetNotifyReceiver(ctx context.Context, id int64) (*model.NotifyReceiver, error) {
	receiver, err := q.notify.Receivers.GetByID(ctx, id)
	if err != nil || receiver == nil {
		return nil, err
	}
	item := convert.NotifyReceiverItem(receiver)
	return &item, nil
}

// ListNotifyRules 全部路由规则，按 position 从小到大
func (q *QueryService) ListNotifyRules(ctx context.Context) ([]model.NotifyRule, error) {
	rules, err := q.notify.Rules.List(ctx)
	if err != nil {
		return nil, err
	}
	return convert.NotifyRuleItems(rules), nil
}

// GetNotifyRule 单条路由规则；不存在时返回 (nil, nil)
func (q *QueryService) GetNotifyRule(ctx context.Context, id int64) (*model.NotifyRule, error) {
	rule, err := q.notify.Rules.GetByID(ctx, id)
	if err != nil || rule == nil {
		return nil, err
	}
	item := convert.NotifyRuleItem(rule)
	return &item, nil
}

// ListNotifySilences 静默列表（开始时间新的在前）。state 为 pending / active / expired 时只返回该状态的，空 = 全部
func (q *QueryService) ListNotifySilences(ctx context.Context, state string) ([]model.NotifySilence, error) {
	silences, err := q.notify.Silences.List(ctx)
	if err != nil {
		return nil, err
	}
	items := convert.NotifySilenceItems(silences, q.currentTime())
	if state == "" {
		return items, nil
	}
	matched := make([]model.NotifySilence, 0, len(items))
	for _, item := range items {
		if item.State == state {
			matched = append(matched, item)
		}
	}
	return matched, nil
}

// GetNotifySilence 单条静默；不存在时返回 (nil, nil)
func (q *QueryService) GetNotifySilence(ctx context.Context, id int64) (*model.NotifySilence, error) {
	silence, err := q.notify.Silences.GetByID(ctx, id)
	if err != nil || silence == nil {
		return nil, err
	}
	item := convert.NotifySilenceItem(silence, q.currentTime())
	return &item, nil
}

// MatchNotifyRule 试匹配：这样一条告警会命中哪条规则、发给谁。
//
// 用的是通知管道同一个路由函数，结果与真实发送一致。输入不合规 → model.ErrInvalidNotifyInput
func (q *QueryService) MatchNotifyRule(ctx context.Context, req *model.NotifyRuleMatchRequest) (*model.NotifyRuleMatchResult, error) {
	if err := model.ValidateNotifyRuleMatch(req); err != nil {
		return nil, err
	}
	policy, err := q.notify.Policy.Get(ctx)
	if err != nil {
		return nil, err
	}
	rules, err := q.notify.Rules.List(ctx)
	if err != nil {
		return nil, err
	}
	receivers, err := q.notify.Receivers.List(ctx)
	if err != nil {
		return nil, err
	}

	decision := routing.Route(policy, rules, receivers, routing.Target{
		ClusterID: req.ClusterID, Source: alert.Source(req.Source), Severity: alert.Severity(req.Severity),
	})
	result := &model.NotifyRuleMatchResult{
		Receivers:        make([]model.NotifyReceiverRef, 0, len(decision.Receivers)),
		BelowMinSeverity: decision.BelowMinSeverity,
	}
	if decision.Rule != nil {
		result.MatchedRule = &model.NotifyRuleRef{ID: decision.Rule.ID, Name: decision.Rule.Name}
	}
	for _, receiver := range decision.Receivers {
		result.Receivers = append(result.Receivers, convert.NotifyReceiverRef(receiver))
	}
	return result, nil
}

// currentTime 当前时间；时钟可替换，便于测试
func (q *QueryService) currentTime() time.Time {
	if q.now != nil {
		return q.now()
	}
	return time.Now()
}

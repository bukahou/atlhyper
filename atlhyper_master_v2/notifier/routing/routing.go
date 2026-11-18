// atlhyper_master_v2/notifier/routing/routing.go
// 路由：一条告警该发给谁、用什么时间参数（纯函数，不读库、不看时钟）
package routing

import (
	"sort"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// Target 被路由的对象：规则只看这三项
type Target struct {
	ClusterID string
	Source    alert.Source
	Severity  alert.Severity
}

// TargetOf 取出告警里参与路由的三项
func TargetOf(a *alert.Alert) Target {
	return Target{ClusterID: a.ClusterID, Source: a.Source, Severity: a.Severity}
}

// Timing 分组等待、分组间隔、重复间隔
type Timing struct {
	GroupWait      time.Duration
	GroupInterval  time.Duration
	RepeatInterval time.Duration
}

// Decision 路由结果
type Decision struct {
	// Rule 命中的规则；nil = 没有规则命中，走默认策略
	Rule *database.NotifyRule
	// Receivers 规则（或默认策略）指向的接收人，含已停用的 —— 供「试匹配」如实展示。
	// 真正会收到的见 Deliverable
	Receivers []*database.NotifyReceiver
	Timing    Timing
	// BelowMinSeverity 走默认策略、且严重度不到策略的门槛：不会发送
	BelowMinSeverity bool
}

// RuleID 命中规则的 id；走默认策略时为 0
func (d Decision) RuleID() int64 {
	if d.Rule == nil {
		return 0
	}
	return d.Rule.ID
}

// Deliverable 真正会收到通知的接收人：已启用，且没有被严重度门槛挡住
func (d Decision) Deliverable() []*database.NotifyReceiver {
	if d.BelowMinSeverity {
		return nil
	}
	var enabled []*database.NotifyReceiver
	for _, receiver := range d.Receivers {
		if receiver.Enabled {
			enabled = append(enabled, receiver)
		}
	}
	return enabled
}

// Route 按 Position 从小到大匹配规则，第一条启用且条件全部满足的生效；都不命中用默认策略。
func Route(policy *database.NotifyPolicy, rules []*database.NotifyRule, receivers []*database.NotifyReceiver, target Target) Decision {
	ordered := make([]*database.NotifyRule, len(rules))
	copy(ordered, rules)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Position < ordered[j].Position })

	for _, rule := range ordered {
		if rule.Enabled && MatchesRule(rule, target) {
			return Decision{
				Rule:      rule,
				Receivers: pickReceivers(receivers, rule.ReceiverIDs),
				Timing:    ruleTiming(policy, rule),
			}
		}
	}

	decision := Decision{
		Timing:           policyTiming(policy),
		BelowMinSeverity: !target.Severity.AtLeast(alert.Severity(policy.MinSeverity)),
	}
	if len(policy.DefaultReceiverIDs) == 0 {
		// 没有指定默认接收人 = 全部已启用的接收人
		for _, receiver := range receivers {
			if receiver.Enabled {
				decision.Receivers = append(decision.Receivers, receiver)
			}
		}
	} else {
		decision.Receivers = pickReceivers(receivers, policy.DefaultReceiverIDs)
	}
	return decision
}

// MatchesRule 规则的条件是否全部满足（不看规则是否启用）：
// 集群在 ClusterIDs 里（空 = 全部）、来源在 Sources 里（空 = 全部）、严重度不低于 MinSeverity
func MatchesRule(rule *database.NotifyRule, target Target) bool {
	if len(rule.ClusterIDs) > 0 && !contains(rule.ClusterIDs, target.ClusterID) {
		return false
	}
	if len(rule.Sources) > 0 && !contains(rule.Sources, string(target.Source)) {
		return false
	}
	return target.Severity.AtLeast(alert.Severity(rule.MinSeverity))
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// pickReceivers 按给定 id 的顺序取接收人；已不存在的 id 略过
func pickReceivers(receivers []*database.NotifyReceiver, ids []int64) []*database.NotifyReceiver {
	byID := make(map[int64]*database.NotifyReceiver, len(receivers))
	for _, receiver := range receivers {
		byID[receiver.ID] = receiver
	}
	var picked []*database.NotifyReceiver
	for _, id := range ids {
		if receiver, exists := byID[id]; exists {
			picked = append(picked, receiver)
		}
	}
	return picked
}

func policyTiming(policy *database.NotifyPolicy) Timing {
	return Timing{
		GroupWait:      seconds(policy.GroupWaitSeconds),
		GroupInterval:  seconds(policy.GroupIntervalSeconds),
		RepeatInterval: seconds(policy.RepeatIntervalSeconds),
	}
}

// ruleTiming 规则里填了的时间参数覆盖全局值，没填（nil）的用全局值
func ruleTiming(policy *database.NotifyPolicy, rule *database.NotifyRule) Timing {
	timing := policyTiming(policy)
	if rule.GroupWaitSeconds != nil {
		timing.GroupWait = seconds(*rule.GroupWaitSeconds)
	}
	if rule.GroupIntervalSeconds != nil {
		timing.GroupInterval = seconds(*rule.GroupIntervalSeconds)
	}
	if rule.RepeatIntervalSeconds != nil {
		timing.RepeatInterval = seconds(*rule.RepeatIntervalSeconds)
	}
	return timing
}

func seconds(value int) time.Duration { return time.Duration(value) * time.Second }

// ReceiversForRule 一条规则此刻会发给哪些接收人（只含已启用的）；ruleID 为 0、或规则已不存在时取默认策略的接收人。
//
// 供管道在分组发出时使用：从告警入组到发出之间配置可能改过，以发出时的配置为准
func ReceiversForRule(policy *database.NotifyPolicy, rules []*database.NotifyRule, receivers []*database.NotifyReceiver, ruleID int64) []*database.NotifyReceiver {
	decision := Decision{}
	matched := false
	if ruleID != 0 {
		for _, rule := range rules {
			if rule.ID == ruleID {
				decision.Receivers = pickReceivers(receivers, rule.ReceiverIDs)
				matched = true
				break
			}
		}
	}
	if !matched {
		if len(policy.DefaultReceiverIDs) == 0 {
			decision.Receivers = receivers
		} else {
			decision.Receivers = pickReceivers(receivers, policy.DefaultReceiverIDs)
		}
	}
	return decision.Deliverable()
}

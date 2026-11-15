// atlhyper_master_v2/notifier/source/incident.go
// 事件 → 告警：把 AIOps 引擎发布的事件变化变成告警（设计文档「事件 → 告警」一表）
package source

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/notifier/alert"
	"AtlHyper/atlhyper_master_v2/notifier/template"
)

const (
	// clusterScopeNamespace 集群级实体（节点等）的键里占命名空间位置的那一段
	clusterScopeNamespace = "_cluster"
	// maxListedMembers 告警详情里最多列出多少个成员实体，其余只计数
	maxListedMembers = 10
)

// IncidentSource 订阅事件变化，交给通知管道
type IncidentSource struct {
	submitter AlertSubmitter
}

// NewIncidentSource 创建 IncidentSource
func NewIncidentSource(submitter AlertSubmitter) *IncidentSource {
	return &IncidentSource{submitter: submitter}
}

// OnTransition 引擎的订阅回调。
//
// 调用方持有状态机的锁，且可能在快照处理的主路径上：这里只把变化放进管道的有界队列就返回。
// 「要不要产生告警」取决于通知策略（开关、反复阈值），读库与判断都推迟到管道的协程里做。
// 队列满时这次变化被丢弃，由管道计数并限频记日志
func (s *IncidentSource) OnTransition(transition aiops.IncidentTransition) {
	if transition.Kind == aiops.TransitionCreated {
		return // 建事件永远不产生告警，不必占队列
	}
	// 成员切片可能被引擎复用：留一份自己的
	transition.Members = append([]string(nil), transition.Members...)
	s.submitter.SubmitDeferred(func(policy *database.NotifyPolicy) (alert.Alert, bool) {
		return IncidentAlert(transition, policy)
	})
}

// IncidentAlert 按当前策略把一次事件变化变成告警；第二个返回值为假表示这次变化不产生告警。
//
//	escalated（进入事件中）                         → 触发
//	severity_raised，且事件处于事件中               → 升级
//	recovery_started，事件升级过                    → 结束，发恢复通知
//	recurred，事件升级过                            → 复发
//	recurred，没升级过，近 24 小时发生次数 ≥ 阈值   → 反复
//	closed                                          → 结束（告警若仍在持续），发恢复通知
//	closed_stale                                    → 结束，不发恢复通知
//	其余（created、不到阈值的 recurred 等）         → 不产生告警
//
// 「结束」对没有在持续的告警不起作用（管道会忽略），所以这里不必知道告警当前的状态。
// 没升级过的事件进入恢复观察时不结束告警：反复类告警要等事件关闭才结束，
// 否则每抖一次都是「恢复 + 反复」两条。
// 事件告警被策略关掉时不再产生新的告警，但「结束」照常放行 —— 否则已有的告警会永远停在持续中
func IncidentAlert(transition aiops.IncidentTransition, policy *database.NotifyPolicy) (alert.Alert, bool) {
	peakRisk := riskPercent(transition.PeakRisk)
	switch transition.Kind {
	case aiops.TransitionEscalated:
		return openingAlert(transition, policy, alert.KindFiring,
			fmt.Sprintf("风险持续偏高，已升级为事件（峰值风险 %d）", peakRisk))

	case aiops.TransitionSeverityRaised:
		if transition.ToState != aiops.StateIncident {
			return alert.Alert{}, false
		}
		severity := template.SeverityText(incidentSeverity(transition.Severity))
		return openingAlert(transition, policy, alert.KindEscalated,
			fmt.Sprintf("严重度升高为%s（峰值风险 %d）", severity, peakRisk))

	case aiops.TransitionRecurred:
		if transition.ReachedIncident {
			return openingAlert(transition, policy, alert.KindRecurred,
				fmt.Sprintf("恢复后再次出现，第 %d 次复发", transition.Recurrence))
		}
		threshold := policy.WarningFlapThreshold
		if threshold <= 0 || transition.Occurrences24h < threshold {
			return alert.Alert{}, false
		}
		return openingAlert(transition, policy, alert.KindFlapping,
			fmt.Sprintf("近 24 小时内反复 %d 次，未达事件级别", transition.Occurrences24h))

	case aiops.TransitionRecoveryStarted:
		if !transition.ReachedIncident {
			return alert.Alert{}, false
		}
		return closingAlert(transition, "风险已回落，进入恢复观察", false), true

	case aiops.TransitionClosed:
		return closingAlert(transition, "观察期内未再出现，事件已关闭", false), true

	case aiops.TransitionClosedStale:
		// 主体失联（对象已删除或 Agent 断连）并不是真的恢复：结束告警，但不发恢复通知
		return closingAlert(transition, "主体长时间没有数据，事件已关闭", true), true

	default:
		return alert.Alert{}, false
	}
}

// openingAlert 一条「在持续」的告警；事件告警被策略关掉时不产生
func openingAlert(transition aiops.IncidentTransition, policy *database.NotifyPolicy, kind alert.Kind, summary string) (alert.Alert, bool) {
	if !policy.IncidentAlertsEnabled {
		return alert.Alert{}, false
	}
	base := incidentAlertBase(transition)
	base.Status, base.Kind, base.Summary, base.StartsAt = alert.StatusFiring, kind, summary, transition.At
	return base, true
}

// closingAlert 一条「结束」的告警
func closingAlert(transition aiops.IncidentTransition, summary string, quiet bool) alert.Alert {
	base := incidentAlertBase(transition)
	endsAt := transition.At
	base.Status, base.Kind, base.Summary, base.EndsAt, base.Quiet = alert.StatusResolved, alert.KindResolved, summary, &endsAt, quiet
	return base
}

// incidentAlertBase 各种变化共用的部分。
// 指纹用事件号（一个事件一条告警）；标题与标签用主体键 —— 领头实体会随 Pod 重建变化，主体不会
func incidentAlertBase(transition aiops.IncidentTransition) alert.Alert {
	namespace, subjectType := subjectParts(transition.SubjectKey)
	details := map[string]string{
		alert.DetailIncidentID:     transition.IncidentID,
		alert.DetailPeakRisk:       strconv.Itoa(riskPercent(transition.PeakRisk)),
		alert.DetailRecurrence:     strconv.Itoa(transition.Recurrence),
		alert.DetailOccurrences24h: strconv.Itoa(transition.Occurrences24h),
	}
	// 关闭类的变化不带成员与根因：没有值就不写，免得结束时把告警里已有的值冲成空
	if transition.RootCause != "" {
		details[alert.DetailRootCause] = transition.RootCause
	}
	if len(transition.Members) > 0 {
		details[alert.DetailMemberCount] = strconv.Itoa(len(transition.Members))
		details[alert.DetailMembers] = listMembers(transition.Members)
	}
	return alert.Alert{
		Fingerprint: alert.IncidentFingerprint(transition.IncidentID),
		Source:      alert.SourceIncident,
		ClusterID:   transition.ClusterID,
		Severity:    incidentSeverity(transition.Severity),
		Title:       transition.SubjectKey + " 风险异常",
		Labels: map[string]string{
			alert.LabelNamespace:   namespace,
			alert.LabelSubject:     transition.SubjectKey,
			alert.LabelSubjectType: subjectType,
		},
		Details: details,
	}
}

// incidentSeverity 事件的峰值严重度；认不出的取值按 low —— 宁可当成最低档发出，也不让它因为不满足任何门槛而无声消失
func incidentSeverity(text string) alert.Severity {
	if severity, known := alert.ParseSeverity(text); known {
		return severity
	}
	return alert.SeverityLow
}

// subjectParts 从主体键 <命名空间>/<类型>/<名字> 取出命名空间与类型；集群级主体没有命名空间
func subjectParts(subjectKey string) (namespace, subjectType string) {
	parts := strings.SplitN(subjectKey, "/", 3)
	if len(parts) < 3 {
		return "", ""
	}
	namespace, subjectType = parts[0], parts[1]
	if namespace == clusterScopeNamespace {
		namespace = ""
	}
	return namespace, subjectType
}

// listMembers 成员实体键，逗号分隔；过多时只列前几个
func listMembers(members []string) string {
	if len(members) <= maxListedMembers {
		return strings.Join(members, ", ")
	}
	return strings.Join(members[:maxListedMembers], ", ") + fmt.Sprintf(" 等 %d 个", len(members))
}

// riskPercent 风险值（0–1）→ 百分数（0–100，四舍五入）
func riskPercent(risk float64) int {
	return int(math.Round(risk * 100))
}

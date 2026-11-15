// atlhyper_master_v2/aiops/incident/detail_view.go
// 事件详情页的展示形态：关键证据（按信号类型去重）、结论的主证据、时间线的结构化记录。
//
// 全部是纯函数：输入库里的记录，输出给前端的结构化数据。文字由前端按 i18n 渲染，这里只给码与数值
package incident

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// SignalOf 指标名 → 信号类型
func SignalOf(metric string) string {
	switch {
	case metric == "container_anomaly":
		return aiops.SignalContainerCrash
	case metric == "deployment_impact":
		return aiops.SignalWorkloadUnavailable
	case metric == "critical_event":
		return aiops.SignalK8sEvent
	case metric == "restart_count" || metric == "max_container_restarts":
		return aiops.SignalRestarts
	case metric == "not_ready_containers" || metric == "is_running":
		return aiops.SignalNotReady
	case strings.HasSuffix(metric, "_pressure"):
		return aiops.SignalNodePressure
	case strings.HasPrefix(metric, "log_error"):
		return aiops.SignalLogErrors
	case strings.Contains(metric, "latency"):
		return aiops.SignalAPMLatency
	case strings.Contains(metric, "error_rate"):
		return aiops.SignalAPMErrors
	case strings.HasSuffix(metric, "_usage") || strings.HasPrefix(metric, "psi_"):
		return aiops.SignalResource
	default:
		return aiops.SignalOther
	}
}

// podUIDPattern K8s 事件原文里「pod 名_命名空间(UID)」中的 (UID) 部分：对读的人没有意义
var podUIDPattern = regexp.MustCompile(`\([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\)`)

// cleanSignalDetail 去掉原文里的 Pod UID 等噪音
func cleanSignalDetail(detail string) string {
	return strings.TrimSpace(podUIDPattern.ReplaceAllString(detail, ""))
}

// 原文由基线模块写出，再经 aiops.DescribeAnomaly 加上类别前缀（如「容器异常: 」「K8s 事件 」），按片段匹配而不是按开头
var (
	containerReason    = regexp.MustCompile(`容器 [^:]+: (\S+)`)           // 「容器 <名>: <原因>」
	unavailablePercent = regexp.MustCompile(`不可用 (\d+)%`)                // 「Deployment <ns>/<名> 不可用 <N>%」
	eventReason        = regexp.MustCompile(`(?:^|K8s 事件 )([A-Za-z]+):`) // 「<原因>: <消息>」
)

// signalReason 从原文里取出原因码 / 取值；取不到时为空串
func signalReason(signal, detail string) string {
	var pattern *regexp.Regexp
	switch signal {
	case aiops.SignalContainerCrash:
		pattern = containerReason
	case aiops.SignalWorkloadUnavailable:
		pattern = unavailablePercent
	case aiops.SignalK8sEvent:
		pattern = eventReason
	default:
		return ""
	}
	if match := pattern.FindStringSubmatch(detail); match != nil {
		return match[1]
	}
	return ""
}

// BuildEvidence 把（成员, 指标）粒度的信号记录按信号类型合并：
// 中招成员数去重、首次取最早、最近取最晚、次数相加、最高分取大、原文取最近的一条。按最高分从高到低
func BuildEvidence(signals []*database.AIOpsIncidentSignal) []aiops.IncidentEvidence {
	type group struct {
		evidence aiops.IncidentEvidence
		members  map[string]bool
	}
	groups := map[string]*group{}
	var order []string
	for _, s := range signals {
		signal := SignalOf(s.Metric)
		g, ok := groups[signal]
		if !ok {
			g = &group{
				evidence: aiops.IncidentEvidence{Signal: signal, FirstSeen: s.FirstSeen, LastSeen: s.LastSeen, Detail: s.Detail},
				members:  map[string]bool{},
			}
			groups[signal] = g
			order = append(order, signal)
		}
		e := &g.evidence
		g.members[s.EntityKey] = true
		e.Hits += s.Hits
		if s.MaxScore > e.MaxScore {
			e.MaxScore = s.MaxScore
		}
		if s.FirstSeen.Before(e.FirstSeen) {
			e.FirstSeen = s.FirstSeen
		}
		if s.LastSeen.After(e.LastSeen) {
			e.LastSeen, e.Detail = s.LastSeen, s.Detail
		}
	}

	result := make([]aiops.IncidentEvidence, 0, len(order))
	for _, signal := range order {
		g := groups[signal]
		e := g.evidence
		e.AffectedMembers = len(g.members)
		e.Detail = cleanSignalDetail(e.Detail)
		e.Reason = signalReason(signal, e.Detail)
		result = append(result, e)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].MaxScore != result[j].MaxScore {
			return result[i].MaxScore > result[j].MaxScore
		}
		return result[i].AffectedMembers > result[j].AffectedMembers
	})
	return result
}

// BuildHeadline 结论的主证据：分数最高的一条；没有证据时为 nil
func BuildHeadline(evidence []aiops.IncidentEvidence, totalMembers int) *aiops.IncidentHeadline {
	if len(evidence) == 0 {
		return nil
	}
	top := evidence[0]
	return &aiops.IncidentHeadline{Signal: top.Signal, Reason: top.Reason, AffectedMembers: top.AffectedMembers, TotalMembers: totalMembers}
}

var timelineRisk = regexp.MustCompile(`风险 (\d+)\)`)
var raisedSeverity = regexp.MustCompile(`严重度升高 → (\w+)`)

// BuildTimelineView 时间线的展示形态：状态与严重度的变化逐条转成结构化记录，检测记录合并成最后一条。
// 原文由本包的 transition.go 按固定格式写出，这里按同一格式读回
func BuildTimelineView(timeline []*aiops.IncidentTimeline) []aiops.IncidentTimelineItem {
	items := []aiops.IncidentTimelineItem{}
	var anomaly *aiops.IncidentTimelineItem
	for _, t := range timeline {
		if t.EventType == aiops.TimelineAnomalyDetected {
			if anomaly == nil {
				first := t.Timestamp
				anomaly = &aiops.IncidentTimelineItem{At: t.Timestamp, Kind: aiops.TimelineItemAnomaly, FirstAt: &first}
			}
			last := t.Timestamp
			anomaly.Count++
			anomaly.LastAt = &last
			continue
		}
		items = append(items, timelineItem(t))
	}
	if anomaly != nil {
		items = append(items, *anomaly)
	}
	return items
}

func timelineItem(t *aiops.IncidentTimeline) aiops.IncidentTimelineItem {
	item := aiops.IncidentTimelineItem{At: t.Timestamp, Risk: parseTimelineRisk(t.Detail)}
	switch {
	case t.EventType == aiops.TimelineSeverityRaised:
		item.Kind = aiops.TimelineItemSeverityRaised
		if match := raisedSeverity.FindStringSubmatch(t.Detail); match != nil {
			item.Severity = match[1]
		}
	case t.EventType == aiops.TimelineRecoveryStarted || (t.EventType == aiops.TimelineStateChange && strings.HasPrefix(t.Detail, legacyRecoveryPrefix)):
		item.Kind = aiops.TimelineItemRecoveryStarted
	case t.EventType == aiops.TimelineRecurrence:
		item.Kind = aiops.TimelineItemRecurred
	case t.EventType == aiops.TimelineStateChange && strings.Contains(t.Detail, "healthy → warning"):
		item.Kind = aiops.TimelineItemCreated
	case t.EventType == aiops.TimelineStateChange && strings.Contains(t.Detail, "→ "+string(aiops.StateIncident)):
		item.Kind = aiops.TimelineItemEscalated
	case t.EventType == aiops.TimelineStateChange && strings.Contains(t.Detail, "没有数据"):
		item.Kind = aiops.TimelineItemClosedStale
	case t.EventType == aiops.TimelineStateChange && strings.HasPrefix(t.Detail, "事件已关闭"):
		item.Kind = aiops.TimelineItemClosed
	default:
		item.Kind, item.Detail, item.Risk = aiops.TimelineItemNote, t.Detail, nil
	}
	return item
}

func parseTimelineRisk(detail string) *int {
	match := timelineRisk.FindStringSubmatch(detail)
	if match == nil {
		return nil
	}
	risk, err := strconv.Atoi(match[1])
	if err != nil {
		return nil
	}
	return &risk
}

// signalRecord 一个成员的一条信号 → 累计记录（只填写入需要的字段，首次 / 次数由库里累计）
func signalRecord(incidentID string, member *aiops.EntityRisk, signal aiops.RiskSignal, at time.Time) *database.AIOpsIncidentSignal {
	return &database.AIOpsIncidentSignal{
		IncidentID: incidentID, EntityKey: member.EntityKey, Metric: signal.Metric,
		LastSeen: at, MaxScore: signal.Score, Detail: signal.Detail,
	}
}

// RecordActivity 事件进行中（两次状态变化之间）的一轮记录：
// 成员表跟上当前成员与领头（Pod 重建换了名字的新成员也能进来），并累计成员的信号（关键证据）。
// 没有本地信号的成员（风险来自依赖传播）不记证据
func (s *Store) RecordActivity(ctx context.Context, incidentID string, risk *aiops.SubjectRisk, at time.Time) {
	if risk == nil || len(risk.Members) == 0 {
		return
	}
	lead := aiops.IncidentTransition{IncidentID: incidentID}
	if risk.Lead != nil {
		lead.RootCause = risk.Lead.EntityKey
	}
	writes := append(s.leadWrites(lead, risk), s.memberSignalWrites(incidentID, risk.Members, at)...)
	s.submit(ctx, writes...)
}

// memberSignalWrites 每个成员的每条信号一步累计写入
func (s *Store) memberSignalWrites(incidentID string, members []*aiops.EntityRisk, at time.Time) []pendingWrite {
	var writes []pendingWrite
	for _, member := range members {
		if member == nil || member.RLocal == 0 {
			continue
		}
		for _, signal := range member.Signals {
			record := signalRecord(incidentID, member, signal, at)
			writes = append(writes, pendingWrite{"累计事件证据 " + incidentID, func(ctx context.Context) error {
				return s.repo.UpsertSignal(ctx, record)
			}})
		}
	}
	return writes
}

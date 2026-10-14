// atlhyper_master_v2/aiops/enricher/evidence.go
// 一次事件分析所需的全部证据：后台摘要（Complete）与深度分析（Analyze）共用一份
//
// 此前两条路径各拼各的：深度分析只有概要 / 实体 / 时间线，没有历史、OTel、Pod 状态；
// 后台分析超长截断时重建上下文又把 OTel 与 Pod 状态丢了。统一后证据只在这里组装一次。
package enricher

import (
	"context"
	"fmt"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/prompts"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/cluster"
)

// historyLookback 历史相似事件回看窗口
const historyLookback = 90 * 24 * time.Hour

// incidentEvidence 事件分析证据（原始数据；渲染与截断在 promptContext 里做）
type incidentEvidence struct {
	incident   *database.AIOpsIncident
	entities   []*database.AIOpsIncidentEntity
	timeline   []*database.AIOpsIncidentTimeline
	historical []*database.AIOpsIncident
	snapshot   *cluster.ClusterSnapshot // 可为 nil（无 Store 或集群离线）

	eventsQueried bool                     // 是否注入了 event 源（未注入则上下文不出现该节）
	events        []*database.ClusterEvent // 事件窗口内的持久化 event（已排序、已限量）
	eventsFailed  bool
	concurrent    []*database.AIOpsIncident // 同时段其他事件
}

// assembleIncidentContext 查齐一个事件的全部证据
func (e *Enricher) assembleIncidentContext(ctx context.Context, incidentID string) (*incidentEvidence, error) {
	incident, err := e.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("事件不存在: %s", incidentID)
	}

	entities, err := e.incidentRepo.GetEntities(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("查询受影响实体失败: %w", err)
	}
	timeline, err := e.incidentRepo.GetTimeline(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("查询时间线失败: %w", err)
	}

	ev := &incidentEvidence{incident: incident, entities: entities, timeline: timeline}

	if incident.RootCause != "" {
		historical, _ := e.incidentRepo.ListByEntity(ctx, incident.RootCause, time.Now().Add(-historyLookback))
		ev.historical = excludeIncident(historical, incident.ID)
	}

	if e.events != nil {
		from, to := eventWindow(incident, timeline, time.Now())
		ev.eventsQueried = true
		ev.events, ev.eventsFailed = collectWindowEvents(ctx, e.events, incident.ClusterID, entities, from, to)
	}
	ev.concurrent = e.collectConcurrent(ctx, incident)

	if e.store != nil && incident.ClusterID != "" {
		if snap, err := e.store.GetSnapshot(incident.ClusterID); err == nil {
			ev.snapshot = snap
		}
	}
	return ev, nil
}

// truncLevel 截断档位：0 = 完整；越大删得越多。
// 顺序原则：越是直接证据越晚删 —— 同时段事件 → 历史模式 → K8s event → 时间线 → 实体
type truncLevel struct {
	concurrent, historical, events, timeline, entities fraction
}

// fraction 保留比例
type fraction int

const (
	keepAll fraction = iota
	keepHalf
	keepNone
)

func applyFraction[T any](s []T, f fraction) []T {
	switch f {
	case keepHalf:
		return s[:len(s)/2]
	case keepNone:
		return nil
	default:
		return s
	}
}

var truncLevels = []truncLevel{
	{keepAll, keepAll, keepAll, keepAll, keepAll},
	{keepNone, keepAll, keepAll, keepAll, keepAll},
	{keepNone, keepHalf, keepAll, keepAll, keepAll},
	{keepNone, keepNone, keepAll, keepAll, keepAll},
	{keepNone, keepNone, keepHalf, keepAll, keepAll},
	{keepNone, keepNone, keepHalf, keepHalf, keepAll},
	{keepNone, keepNone, keepHalf, keepHalf, keepHalf},
	{keepNone, keepNone, keepNone, keepNone, keepHalf},
	{keepNone, keepNone, keepNone, keepNone, keepNone},
}

// promptContext 按截断档位渲染为 LLM 上下文
func (ev *incidentEvidence) promptContext(level int) *prompts.IncidentPromptContext {
	lv := truncLevels[min(level, len(truncLevels)-1)]

	pc := BuildIncidentContext(
		ev.incident,
		applyFraction(ev.entities, lv.entities),
		applyFraction(ev.timeline, lv.timeline),
		applyFraction(ev.historical, lv.historical),
	)
	// 异常时长依赖完整时间线：截断时间线不能让「异常持续」变成「未记录」
	pc.IncidentSummary = buildSummary(ev.incident, ev.timeline)

	if ev.eventsQueried {
		pc.K8sEvents = renderEvents(applyFraction(ev.events, lv.events), len(ev.events), ev.eventsFailed, ev.incident.StartedAt.Location())
	}
	pc.ConcurrentIncidents = buildConcurrent(ev.incident, applyFraction(ev.concurrent, lv.concurrent), len(ev.concurrent))

	// 快照证据各自有条数上限，不参与截断
	if ev.snapshot != nil {
		pc.RecentErrorTraces, pc.RecentErrorLogs, pc.SLOContext = buildOTelContext(ev.snapshot.OTel, ev.entities)
		pc.PodStates = buildPodStateContext(ev.snapshot, ev.entities) // A10：容器状态作为证据
	}
	return pc
}

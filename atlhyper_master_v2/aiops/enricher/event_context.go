// atlhyper_master_v2/aiops/enricher/event_context.go
// 事件分析上下文：master 持久化的 K8s event + 同时段其他事件
//
// K8s 自身只保留约 1 小时 event，模型曾据此写「事件已丢失、数据不足」，
// 而 cluster_events 表里明明有事件期间的 readiness 探针超时原文。
package enricher

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops/incident"
	"AtlHyper/atlhyper_master_v2/database"
)

// EventHistoryReader 持久化 K8s event 的最小读接口（database.EventRepository 满足）
type EventHistoryReader interface {
	ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*database.ClusterEvent, error)
}

const (
	eventWindowBefore    = 15 * time.Minute // 事件开始前多看一段：触发信号往往早于状态变化
	eventWindowAfter     = 5 * time.Minute  // 恢复后多看一段：收尾的事件
	concurrentWindow     = 15 * time.Minute // 同时段 = 开始时间前后 15 分钟
	MaxEventEntries      = 20
	MaxConcurrentEntries = 20
	maxEventMessageRunes = 200
)

// SetEventSource 注入持久化 event 读取（可选；未注入时上下文不出现该节）
func (e *Enricher) SetEventSource(reader EventHistoryReader) {
	e.events = reader
}

// eventWindow 开始前 15min → 异常结束后 5min（仍在异常中则到 now）
func eventWindow(inc *database.AIOpsIncident, timeline []*database.AIOpsIncidentTimeline, now time.Time) (from, to time.Time) {
	from = inc.StartedAt.Add(-eventWindowBefore)
	if end, recovered := incident.ActiveWindow(timeline); recovered {
		return from, end.Add(eventWindowAfter)
	}
	return from, now
}

// entityEventTarget 实体 → event 的 involvedObject；service / ingress 等不挂 event，返回 ok=false
func entityEventTarget(entityKey string) (kind, namespace, name string, ok bool) {
	parts := strings.SplitN(entityKey, "/", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	switch parts[1] {
	case "pod":
		return "Pod", parts[0], parts[2], true
	case "node":
		return "Node", "", parts[2], true // 集群级资源的 event 没有 namespace
	default:
		return "", "", "", false
	}
}

// collectWindowEvents 查各实体在窗口内的 event：行的 [首次, 最近] 与窗口有交集即入选，
// 按最近一次倒序，最多 MaxEventEntries 行。任一查询失败 → failed=true（已取到的仍返回）
func collectWindowEvents(ctx context.Context, reader EventHistoryReader, clusterID string,
	entities []*database.AIOpsIncidentEntity, from, to time.Time) (rows []*database.ClusterEvent, failed bool) {

	seen := map[string]bool{}
	for _, ent := range entities {
		kind, ns, name, ok := entityEventTarget(ent.EntityKey)
		if !ok || seen[ent.EntityKey] {
			continue
		}
		seen[ent.EntityKey] = true

		list, err := reader.ListByInvolvedResource(ctx, clusterID, kind, ns, name)
		if err != nil {
			log.Warn("读取持久化 event 失败", "entity", ent.EntityKey, "err", err)
			failed = true
			continue
		}
		for _, ev := range list {
			if ev.LastTimestamp.Before(from) || ev.FirstTimestamp.After(to) {
				continue
			}
			rows = append(rows, ev)
		}
	}

	sort.SliceStable(rows, func(i, j int) bool { return rows[i].LastTimestamp.After(rows[j].LastTimestamp) })
	if len(rows) > MaxEventEntries {
		rows = rows[:MaxEventEntries]
	}
	return rows, failed
}

// renderEvents total = 截断前的行数（截断后 rows 可能更少）。
// 「没有事件」「读失败」「被长度限制省略」三种情况必须说得不一样
func renderEvents(rows []*database.ClusterEvent, total int, failed bool, loc *time.Location) string {
	if failed && len(rows) == 0 {
		return "（读取失败）"
	}
	if total == 0 {
		return "事件窗口内无 Warning 事件"
	}
	if len(rows) == 0 {
		return fmt.Sprintf("（共 %d 条，因长度限制省略）", total)
	}

	const ts = "01-02 15:04"
	var b strings.Builder
	for _, ev := range rows {
		target := ev.InvolvedName
		if ev.InvolvedNamespace != "" {
			target = ev.InvolvedNamespace + "/" + ev.InvolvedName
		}
		fmt.Fprintf(&b, "- [%s %s] %s（累计 %d 次，首次 %s，最近 %s）\n  %s\n",
			ev.InvolvedKind, target, ev.Reason, ev.Count,
			ev.FirstTimestamp.In(loc).Format(ts), ev.LastTimestamp.In(loc).Format(ts),
			truncateRunes(strings.TrimSpace(ev.Message), maxEventMessageRunes))
	}
	if omitted := total - len(rows); omitted > 0 {
		fmt.Fprintf(&b, "… 另有 %d 条因长度限制省略\n", omitted)
	}
	if failed {
		b.WriteString("（部分实体读取失败）\n")
	}
	return b.String()
}

// collectConcurrent 同集群、开始时间前后 15 分钟内的其他事件。
// started_at 以 RFC3339 字符串比较：上下界由 inc.StartedAt 推出，时区偏移与库中一致
func (e *Enricher) collectConcurrent(ctx context.Context, inc *database.AIOpsIncident) []*database.AIOpsIncident {
	list, err := e.incidentRepo.List(ctx, database.AIOpsIncidentQueryOpts{
		ClusterID: inc.ClusterID,
		From:      inc.StartedAt.Add(-concurrentWindow),
		To:        inc.StartedAt.Add(concurrentWindow),
		Limit:     MaxConcurrentEntries + 1, // 多取一条给自身
	})
	if err != nil {
		log.Warn("查询同时段事件失败", "incident", inc.ID, "err", err)
		return nil
	}
	others := excludeIncident(list, inc.ID)
	if len(others) > MaxConcurrentEntries {
		others = others[:MaxConcurrentEntries]
	}
	return others
}

// buildConcurrent total = 截断前的条数。多个不同命名空间同时出事只陈述事实，推断交给规则约束下的模型
func buildConcurrent(self *database.AIOpsIncident, others []*database.AIOpsIncident, total int) string {
	if total == 0 {
		return "同时段（开始前后 15 分钟）无其他事件"
	}
	if len(others) == 0 {
		return fmt.Sprintf("同时段（开始前后 15 分钟）有 %d 个其他事件（因长度限制省略）", total)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "同时段（开始前后 15 分钟）其他事件 (%d 个):\n", total)
	namespaces := map[string]bool{namespaceOf(self.RootCause): true}
	for _, inc := range others {
		namespaces[namespaceOf(inc.RootCause)] = true
		fmt.Fprintf(&b, "  - %s %s | 峰值风险 %s | %s\n",
			inc.StartedAt.In(self.StartedAt.Location()).Format("15:04:05"), inc.RootCause, riskScore(inc.PeakRisk), inc.State)
	}
	if n := len(namespaces); n >= 2 {
		fmt.Fprintf(&b, "  同时段 %d 个不同命名空间出现事件\n", n)
	}
	return b.String()
}

func namespaceOf(entityKey string) string {
	if i := strings.Index(entityKey, "/"); i > 0 {
		return entityKey[:i]
	}
	return entityKey
}

// truncateRunes 按字符截断（不切坏多字节字符）
func truncateRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

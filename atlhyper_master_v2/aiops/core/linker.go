// atlhyper_master_v2/aiops/core/linker.go
// 拓扑归并在引擎里的一环：每轮评估之后，对本集群的活跃事件归组，与上一轮比较，
// 把变化（新归入 / 换源头 / 解除）落库并发给订阅者（通知模块据此抑制下游告警）
package core

import (
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/grouping"
)

// linkState 各集群当前有效的关联：clusterID → 受影响事件号 → 关联
type linkState struct {
	mu      sync.Mutex
	current map[string]map[string]aiops.IncidentLink
}

// diff 用本轮归组结果替换某个集群的关联，返回变化（解除的 RootIncidentID 为空）。
// 只比较源头：路径每轮可能走出不同的等长分支，不算变化
func (s *linkState) diff(clusterID string, next map[string]grouping.Link, now time.Time) []aiops.IncidentLink {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		s.current = map[string]map[string]aiops.IncidentLink{}
	}
	previous := s.current[clusterID]
	updated := make(map[string]aiops.IncidentLink, len(next))
	var changes []aiops.IncidentLink

	for incidentID, link := range next {
		if old, ok := previous[incidentID]; ok && old.RootIncidentID == link.RootIncidentID {
			updated[incidentID] = old
			continue
		}
		change := aiops.IncidentLink{ClusterID: clusterID, IncidentID: incidentID, RootIncidentID: link.RootIncidentID, Path: link.Path, At: now}
		updated[incidentID] = change
		changes = append(changes, change)
	}
	for incidentID := range previous {
		if _, still := next[incidentID]; !still {
			changes = append(changes, aiops.IncidentLink{ClusterID: clusterID, IncidentID: incidentID, At: now})
		}
	}
	s.current[clusterID] = updated
	return changes
}

// restore 启动时放入库里恢复的有效关联
func (s *linkState) restore(links []aiops.IncidentLink) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		s.current = map[string]map[string]aiops.IncidentLink{}
	}
	for _, link := range links {
		if s.current[link.ClusterID] == nil {
			s.current[link.ClusterID] = map[string]aiops.IncidentLink{}
		}
		s.current[link.ClusterID][link.IncidentID] = link
	}
}

// updateLinks 本轮评估之后对本集群的活跃事件归组，变化落库并发布
func (e *engine) updateLinks(clusterID string, subjects map[string]string, now time.Time) {
	if e.sm == nil || e.incidentStore == nil || e.corr == nil {
		return
	}
	graph := e.corr.GetGraph(clusterID)
	if graph == nil {
		return
	}
	candidates := linkCandidates(e.sm.ActiveIncidents(clusterID), subjectMembers(graph, subjects), now)
	changes := e.links.diff(clusterID, grouping.Group(candidates, graph), now)
	if len(changes) == 0 {
		return
	}
	e.incidentStore.ApplyLinks(e.bgCtx, changes)
	for _, change := range changes {
		e.publishLink(change)
	}
}

// subjectMembers 主体 → 成员实体：依赖图里的每个实体按主体归类；没有上级主体的实体，主体就是自己
func subjectMembers(graph *aiops.DependencyGraph, subjects map[string]string) map[string][]string {
	members := map[string][]string{}
	for key := range graph.Nodes {
		subject, ok := subjects[key]
		if !ok {
			subject = key
		}
		members[subject] = append(members[subject], key)
	}
	return members
}

// linkCandidates 状态机里的活跃条目 → 归组的输入。开始时刻取这一轮异常的起点
func linkCandidates(entries []aiops.StateMachineEntry, members map[string][]string, now time.Time) []grouping.Candidate {
	candidates := make([]grouping.Candidate, 0, len(entries))
	for _, entry := range entries {
		started := now
		switch {
		case entry.EpisodeStartedAt != 0:
			started = time.Unix(entry.EpisodeStartedAt, 0)
		case len(entry.OccurrenceTimes) > 0:
			started = time.Unix(entry.OccurrenceTimes[len(entry.OccurrenceTimes)-1], 0)
		}
		candidates = append(candidates, grouping.Candidate{
			IncidentID: entry.IncidentID, SubjectKey: entry.SubjectKey, StartedAt: started,
			Risk: entry.LastRFinal, Members: members[entry.SubjectKey],
		})
	}
	return candidates
}

// SubscribeLinks 订阅关联变化（须在 Start 之前调用）
func (e *engine) SubscribeLinks(fn func(aiops.IncidentLink)) {
	if fn != nil {
		e.linkSubscribers = append(e.linkSubscribers, fn)
	}
}

func (e *engine) publishLink(link aiops.IncidentLink) {
	for _, fn := range e.linkSubscribers {
		fn(link)
	}
}

// reloadLinks 启动时从库里恢复有效关联，并逐条发给订阅者（通知模块据此重建抑制）
func (e *engine) reloadLinks() {
	if e.incidentStore == nil {
		return
	}
	links := e.incidentStore.ActiveLinks(e.bgCtx)
	e.links.restore(links)
	for _, link := range links {
		e.publishLink(link)
	}
	if len(links) > 0 {
		log.Info("拓扑归并的关联已恢复", "count", len(links))
	}
}

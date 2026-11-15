// atlhyper_master_v2/aiops/incident/links.go
// 拓扑归并的关联：落库、恢复、详情与列表里的关联信息
package incident

import (
	"context"
	"encoding/json"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
)

// ApplyLinks 把关联变化落库（经写入队列，失败会重试）：RootIncidentID 为空 = 解除
func (s *Store) ApplyLinks(ctx context.Context, changes []aiops.IncidentLink) {
	writes := make([]pendingWrite, 0, len(changes))
	for _, change := range changes {
		change := change
		if change.RootIncidentID == "" {
			writes = append(writes, pendingWrite{"解除事件关联 " + change.IncidentID, func(ctx context.Context) error {
				return s.repo.UnlinkIncident(ctx, change.IncidentID, change.At)
			}})
			continue
		}
		path, _ := json.Marshal(change.Path)
		record := &database.AIOpsIncidentLink{
			IncidentID: change.IncidentID, ClusterID: change.ClusterID, RootIncidentID: change.RootIncidentID,
			Path: string(path), LinkedAt: change.At,
		}
		writes = append(writes, pendingWrite{"写入事件关联 " + change.IncidentID, func(ctx context.Context) error {
			return s.repo.SaveLink(ctx, record)
		}})
	}
	if len(writes) > 0 {
		s.submit(ctx, writes...)
	}
}

// ActiveLinks 全部集群当前有效的关联（启动恢复用）；读不到时为空
func (s *Store) ActiveLinks(ctx context.Context) []aiops.IncidentLink {
	return s.activeLinks(ctx, "")
}

func (s *Store) activeLinks(ctx context.Context, clusterID string) []aiops.IncidentLink {
	rows, err := s.repo.ListActiveLinks(ctx, clusterID)
	if err != nil {
		log.Warn("读取事件关联失败", "cluster", clusterID, "err", err)
		return nil
	}
	links := make([]aiops.IncidentLink, 0, len(rows))
	for _, row := range rows {
		links = append(links, toAIOpsLink(row))
	}
	return links
}

func toAIOpsLink(row *database.AIOpsIncidentLink) aiops.IncidentLink {
	var path []aiops.LinkPathStep
	if err := json.Unmarshal([]byte(row.Path), &path); err != nil || path == nil {
		path = []aiops.LinkPathStep{}
	}
	return aiops.IncidentLink{
		ClusterID: row.ClusterID, IncidentID: row.IncidentID, RootIncidentID: row.RootIncidentID, Path: path, At: row.LinkedAt,
	}
}

// annotateLinks 列表里的每条事件带上关联：受影响的写源头、源头写受影响数；grouped 时源头再带上受影响事件的摘要
func (s *Store) annotateLinks(ctx context.Context, clusterID string, incidents []*aiops.Incident, grouped bool) {
	links := s.activeLinks(ctx, clusterID)
	if len(links) == 0 {
		return
	}
	rootOf := make(map[string]string, len(links))
	children := map[string][]aiops.IncidentLink{}
	for _, link := range links {
		rootOf[link.IncidentID] = link.RootIncidentID
		children[link.RootIncidentID] = append(children[link.RootIncidentID], link)
	}
	for _, inc := range incidents {
		inc.RootIncidentID = rootOf[inc.ID]
		inc.LinkedCount = len(children[inc.ID])
		if grouped && inc.LinkedCount > 0 {
			inc.Impacted = s.linkRefs(ctx, children[inc.ID], false)
		}
	}
}

// annotateDetailLinks 详情：归到的源头（带路径），以及归到它下面的受影响事件（带路径）
func (s *Store) annotateDetailLinks(ctx context.Context, detail *aiops.IncidentDetail) {
	links := s.activeLinks(ctx, detail.ClusterID)
	var children []aiops.IncidentLink
	for _, link := range links {
		switch {
		case link.IncidentID == detail.ID:
			if refs := s.linkRefs(ctx, []aiops.IncidentLink{{IncidentID: link.RootIncidentID, Path: link.Path, At: link.At}}, true); len(refs) == 1 {
				detail.Root = &refs[0]
				detail.RootIncidentID = link.RootIncidentID
			}
		case link.RootIncidentID == detail.ID:
			children = append(children, link)
		}
	}
	detail.LinkedCount = len(children)
	detail.Impacted = s.linkRefs(ctx, children, true)
}

// linkRefs 关联另一端事件的摘要（按 IncidentID 取事件）；withPath 时带上路径与关联时刻。事件读不到的跳过
func (s *Store) linkRefs(ctx context.Context, links []aiops.IncidentLink, withPath bool) []aiops.IncidentLinkRef {
	refs := make([]aiops.IncidentLinkRef, 0, len(links))
	for _, link := range links {
		inc, err := s.repo.GetByID(ctx, link.IncidentID)
		if err != nil || inc == nil {
			continue
		}
		view := toAIOpsIncident(inc)
		ref := aiops.IncidentLinkRef{
			IncidentID: view.ID, Subject: view.Subject, SubjectType: view.SubjectType,
			State: view.State, Severity: view.Severity,
		}
		if withPath {
			at := link.At
			ref.Path, ref.LinkedAt = link.Path, &at
		}
		refs = append(refs, ref)
	}
	return refs
}

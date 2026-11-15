package incident

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// openIncidentFor 用一次「建事件」变化开一个事件（主体自身是成员）
func openIncidentFor(t *testing.T, s *Store, id, subject string, at time.Time) {
	t.Helper()
	risk := &aiops.SubjectRisk{SubjectKey: subject, RFinal: 0.6, Lead: &aiops.EntityRisk{EntityKey: subject, RFinal: 0.6}}
	risk.Members = []*aiops.EntityRisk{risk.Lead}
	s.ApplyTransition(context.Background(), aiops.IncidentTransition{
		Kind: aiops.TransitionCreated, IncidentID: id, ClusterID: "c1", SubjectKey: subject, RootCause: subject,
		Severity: "medium", RFinal: 0.6, PeakRisk: 0.6, At: at,
	}, risk)
}

func TestLinks_ListAndDetail(t *testing.T) {
	s, _ := openSQLiteStore(t)
	ctx := context.Background()
	openIncidentFor(t, s, "inc-root", "shop/deployment/api", transitionEpoch)
	openIncidentFor(t, s, "inc-svc", "shop/service/api", transitionEpoch.Add(time.Minute))
	openIncidentFor(t, s, "inc-solo", "other/deployment/x", transitionEpoch.Add(2*time.Minute))

	path := []aiops.LinkPathStep{{EntityKey: "shop/service/api"}, {EntityKey: "shop/pod/api-1", Edge: "selects"}}
	s.ApplyLinks(ctx, []aiops.IncidentLink{{ClusterID: "c1", IncidentID: "inc-svc", RootIncidentID: "inc-root", Path: path, At: transitionEpoch.Add(time.Minute)}})

	// 平铺：三条都在，受影响的写源头、源头写数量，impacted 为空数组
	flat, total, err := s.GetIncidents(ctx, aiops.IncidentQueryOpts{ClusterID: "c1"})
	if err != nil || total != 3 || len(flat) != 3 {
		t.Fatalf("平铺应有 3 条，实际 %d / %d (%v)", len(flat), total, err)
	}
	byID := map[string]*aiops.Incident{}
	for _, inc := range flat {
		byID[inc.ID] = inc
	}
	if byID["inc-svc"].RootIncidentID != "inc-root" || byID["inc-root"].LinkedCount != 1 || byID["inc-solo"].LinkedCount != 0 {
		t.Errorf("平铺时的关联标注不对: svc=%+v root=%d", byID["inc-svc"].RootIncidentID, byID["inc-root"].LinkedCount)
	}
	if byID["inc-root"].Impacted == nil || len(byID["inc-root"].Impacted) != 0 {
		t.Errorf("平铺时 impacted 应是空数组: %+v", byID["inc-root"].Impacted)
	}

	// 折叠：受影响的不单独出现，total 按折叠后算，源头带摘要
	grouped, total, err := s.GetIncidents(ctx, aiops.IncidentQueryOpts{ClusterID: "c1", Grouped: true})
	if err != nil || total != 2 || len(grouped) != 2 {
		t.Fatalf("折叠后应有 2 条，实际 %d / %d (%v)", len(grouped), total, err)
	}
	for _, inc := range grouped {
		if inc.ID == "inc-svc" {
			t.Error("受影响事件不应单独出现")
		}
		if inc.ID == "inc-root" && (len(inc.Impacted) != 1 || inc.Impacted[0].IncidentID != "inc-svc" ||
			inc.Impacted[0].SubjectType != "service" || inc.Impacted[0].Path != nil) {
			t.Errorf("源头应带受影响事件摘要（列表里不带路径）: %+v", inc.Impacted)
		}
	}

	// 详情：受影响的带源头与路径；源头带受影响事件与路径
	child := s.GetIncident(ctx, "inc-svc")
	if child.Root == nil || child.Root.IncidentID != "inc-root" || child.Root.Subject != "shop/deployment/api" ||
		len(child.Root.Path) != 2 || child.Root.LinkedAt == nil {
		t.Errorf("受影响事件的详情应带源头与路径: %+v", child.Root)
	}
	root := s.GetIncident(ctx, "inc-root")
	if root.Root != nil || root.LinkedCount != 1 || len(root.Impacted) != 1 || len(root.Impacted[0].Path) != 2 {
		t.Errorf("源头的详情应带受影响事件与路径: root=%+v impacted=%+v", root.Root, root.Impacted)
	}

	// 解除后都不再出现
	s.ApplyLinks(ctx, []aiops.IncidentLink{{ClusterID: "c1", IncidentID: "inc-svc", At: transitionEpoch.Add(5 * time.Minute)}})
	if detail := s.GetIncident(ctx, "inc-svc"); detail.Root != nil || detail.RootIncidentID != "" {
		t.Errorf("解除后不应再有源头: %+v", detail.Root)
	}
	if active := s.ActiveLinks(ctx); len(active) != 0 {
		t.Errorf("解除后不应有有效关联: %+v", active)
	}
}

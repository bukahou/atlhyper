package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// 关联以受影响事件为键：换源头覆盖；解除后不再列出；重新关联会清掉解除时刻；按集群过滤
func TestIncidentLink_SaveUnlinkList(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	for _, id := range []string{"inc-root", "inc-root2", "inc-a", "inc-b"} {
		storeSubjectIncident(t, db, id, "c1", "shop/deployment/"+id, "high")
	}
	storeSubjectIncident(t, db, "inc-other", "c2", "x/service/y", "low")
	at := time.Date(2026, 11, 15, 7, 0, 0, 0, time.UTC)
	save := func(id, cluster, root string) {
		t.Helper()
		if err := db.AIOpsIncident.SaveLink(ctx, &database.AIOpsIncidentLink{
			IncidentID: id, ClusterID: cluster, RootIncidentID: root, Path: `[{"entityKey":"x","edge":""}]`, LinkedAt: at,
		}); err != nil {
			t.Fatal(err)
		}
	}
	save("inc-a", "c1", "inc-root")
	save("inc-b", "c1", "inc-root")
	save("inc-other", "c2", "inc-root")
	save("inc-a", "c1", "inc-root2") // 换源头：覆盖

	links, err := db.AIOpsIncident.ListActiveLinks(ctx, "c1")
	if err != nil || len(links) != 2 {
		t.Fatalf("c1 应有 2 条有效关联，实际 %d (%v)", len(links), err)
	}
	byID := map[string]*database.AIOpsIncidentLink{}
	for _, l := range links {
		byID[l.IncidentID] = l
	}
	if byID["inc-a"].RootIncidentID != "inc-root2" || byID["inc-a"].Path == "" || !byID["inc-a"].LinkedAt.Equal(at) {
		t.Errorf("换源头应覆盖: %+v", byID["inc-a"])
	}

	if err := db.AIOpsIncident.UnlinkIncident(ctx, "inc-b", at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.UnlinkIncident(ctx, "inc-none", at); err != nil {
		t.Errorf("没有关联时解除不应报错: %v", err)
	}
	if links, _ := db.AIOpsIncident.ListActiveLinks(ctx, "c1"); len(links) != 1 || links[0].IncidentID != "inc-a" {
		t.Errorf("解除后不应再列出: %+v", links)
	}
	save("inc-b", "c1", "inc-root")
	if links, _ := db.AIOpsIncident.ListActiveLinks(ctx, ""); len(links) != 3 {
		t.Errorf("重新关联后应恢复有效；不传集群时列全部，期望 3 条，实际 %d", len(links))
	}
}

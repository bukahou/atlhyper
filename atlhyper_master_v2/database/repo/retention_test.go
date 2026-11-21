package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

func TestAuditDeleteBefore(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, at := range []time.Time{now.Add(-200 * 24 * time.Hour), now.Add(-time.Hour)} {
		if err := db.Audit.Create(ctx, &database.AuditLog{Timestamp: at, Action: "login", Resource: "user", Method: "POST", Path: "/x", StatusCode: 200}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := db.Audit.DeleteBefore(ctx, now.Add(-180*24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("DeleteBefore = (%d, %v), want (1, nil)", n, err)
	}
	if left, _ := db.Audit.Count(ctx, database.AuditQueryOpts{}); left != 1 {
		t.Errorf("剩 %d 条，want 1", left)
	}
}

// 只删「已恢复且恢复得够早」的事件，子表（成员实体）随外键级联删除
func TestIncidentDeleteResolvedBefore(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	old := now.Add(-200 * 24 * time.Hour)

	storeIncident(t, db, "old-resolved", "c1", "default/pod/a", old)
	storeIncident(t, db, "recent-resolved", "c1", "default/pod/b", now.Add(-time.Hour))
	storeIncident(t, db, "old-open", "c1", "default/pod/c", old)
	if err := db.AIOpsIncident.Resolve(ctx, "old-resolved", old.Add(time.Hour), "recovered"); err != nil {
		t.Fatal(err)
	}
	if err := db.AIOpsIncident.Resolve(ctx, "recent-resolved", now.Add(-30*time.Minute), "recovered"); err != nil {
		t.Fatal(err)
	}

	n, err := db.AIOpsIncident.DeleteResolvedBefore(ctx, now.Add(-180*24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("DeleteResolvedBefore = (%d, %v), want (1, nil)", n, err)
	}
	for id, wantExists := range map[string]bool{"old-resolved": false, "recent-resolved": true, "old-open": true} {
		inc, err := db.AIOpsIncident.GetByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if (inc != nil) != wantExists {
			t.Errorf("%s 存在 = %v, want %v", id, inc != nil, wantExists)
		}
	}
	var children int
	if err := db.Conn.QueryRow("SELECT COUNT(*) FROM aiops_incident_entities WHERE incident_id = 'old-resolved'").Scan(&children); err != nil {
		t.Fatal(err)
	}
	if children != 0 {
		t.Errorf("被删事件的成员实体应级联删除，剩 %d 行", children)
	}
}

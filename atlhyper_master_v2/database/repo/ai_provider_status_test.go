package repo_test

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// 调用成功后清除错误状态：上一次的错误说明与时间一并清掉，卡片上不再显示旧的错误
func TestAIProviderUpdateStatus_SuccessClearsLastError(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	now := time.Now()
	p := &database.AIProvider{Name: "claude", Provider: "anthropic", Model: "m", CreatedAt: now, UpdatedAt: now}
	if err := db.AIProvider.Create(ctx, p); err != nil {
		t.Fatal(err)
	}

	if err := db.AIProvider.UpdateStatus(ctx, p.ID, "error", "连不上模型服务"); err != nil {
		t.Fatal(err)
	}
	got, err := db.AIProvider.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "error" || got.LastError != "连不上模型服务" || got.LastErrorAt == nil {
		t.Fatalf("记错误后 status=%q lastError=%q lastErrorAt=%v", got.Status, got.LastError, got.LastErrorAt)
	}

	if err := db.AIProvider.UpdateStatus(ctx, p.ID, "active", ""); err != nil {
		t.Fatal(err)
	}
	got, err = db.AIProvider.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "active" || got.LastError != "" || got.LastErrorAt != nil {
		t.Fatalf("成功后 status=%q lastError=%q lastErrorAt=%v，want active / 空 / nil", got.Status, got.LastError, got.LastErrorAt)
	}
	if got.StatusCheckedAt == nil {
		t.Error("成功后应记下检查时间")
	}
}

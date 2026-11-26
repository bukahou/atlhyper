package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"AtlHyper/atlhyper_master_v2/database/sqlite/schema"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func schemaVersion(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	p, err := goose.NewProvider(goose.DialectSQLite3, db, schema.FS)
	if err != nil {
		t.Fatal(err)
	}
	v, err := p.GetDBVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func latestVersion(t *testing.T) int64 {
	t.Helper()
	p, err := goose.NewProvider(goose.DialectSQLite3, openTestDB(t), schema.FS)
	if err != nil {
		t.Fatal(err)
	}
	sources := p.ListSources()
	return sources[len(sources)-1].Version
}

func scalar(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var v sql.NullString
	if err := db.QueryRow(query, args...).Scan(&v); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return v.String
}

// 空库：建全部表、版本到最新、种子补齐
func TestMigrate_FreshDatabase(t *testing.T) {
	db := openTestDB(t)
	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if got, want := schemaVersion(t, db), latestVersion(t); got != want {
		t.Errorf("版本 = %d, want %d", got, want)
	}
	if n := scalar(t, db, "SELECT COUNT(*) FROM ai_provider_models WHERE provider = 'anthropic' AND model = 'claude-haiku-5-5'"); n != "1" {
		t.Errorf("内置模型目录应包含 claude-haiku-5-5，实际 %s 条", n)
	}
	if n := scalar(t, db, "SELECT COUNT(*) FROM ai_role_budget"); n != "3" {
		t.Errorf("默认角色预算应有 3 条，实际 %s", n)
	}
}

// 引入迁移工具之前就存在的库（有表、没有版本表）：不清库，接管为第 1 版后继续迁移
func TestMigrate_AdoptsExistingDatabase(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	// 造一个「旧库」：只有第 1 版的表结构，没有 goose 的版本表
	p, err := goose.NewProvider(goose.DialectSQLite3, db, schema.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE goose_db_version"); err != nil {
		t.Fatal(err)
	}

	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	// 旧数据：东京时区写下的时间、HH:MM 的定时窗口、解析不了的时间、旧模型目录、entrypoint 级 SLO 目标
	mustExec(`INSERT INTO users (username, password_hash, display_name, role, status, created_at, updated_at)
		VALUES ('ops', 'x', 'Ops', 2, 1, '2026-11-19T19:00:00+09:00', '2026-11-19T19:00:00+09:00')`)
	mustExec(`INSERT INTO ai_role_budget (role, schedule_start_time, schedule_end_time, daily_reset_at, updated_at)
		VALUES ('chat', '09:00', '18:00', '2026-11-19T08:30:00+09:00', 'not-a-time')`)
	mustExec(`INSERT INTO ai_provider_models (provider, model, display_name, is_default, sort_order, context_window, created_at)
		VALUES ('anthropic', 'claude-sonnet-4-20250514', 'Claude Sonnet 4', 1, 1, 200000, '2026-11-01T00:00:00Z')`)
	mustExec(`INSERT INTO slo_targets (cluster_id, host, ingress_name, namespace, created_at, updated_at)
		VALUES ('c1', 'web@entrypoint', 'web', 'default', '2026-11-01T00:00:00Z', '2026-11-01T00:00:00Z')`)

	if err := migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if got, want := schemaVersion(t, db), latestVersion(t); got != want {
		t.Errorf("版本 = %d, want %d", got, want)
	}
	if got := scalar(t, db, "SELECT created_at FROM users WHERE username = 'ops'"); got != "2026-11-19T10:00:00Z" {
		t.Errorf("带偏移的时间应换成 UTC，实际 %q", got)
	}
	if got := scalar(t, db, "SELECT daily_reset_at FROM ai_role_budget WHERE role = 'chat'"); got != "2026-11-18T23:30:00Z" {
		t.Errorf("跨日的偏移换算不对，实际 %q", got)
	}
	if got := scalar(t, db, "SELECT schedule_start_time FROM ai_role_budget WHERE role = 'chat'"); got != "09:00" {
		t.Errorf("HH:MM 的定时窗口不应改动，实际 %q", got)
	}
	if got := scalar(t, db, "SELECT updated_at FROM ai_role_budget WHERE role = 'chat'"); got != "not-a-time" {
		t.Errorf("解析不了的值应原样保留，实际 %q", got)
	}
	if n := scalar(t, db, "SELECT COUNT(*) FROM ai_provider_models WHERE model = 'claude-sonnet-4-20250514'"); n != "0" {
		t.Error("带日期的旧 Claude 模型目录条目应被移除")
	}
	if n := scalar(t, db, "SELECT COUNT(*) FROM slo_targets WHERE host LIKE '%@entrypoint%'"); n != "0" {
		t.Error("entrypoint 级的 SLO 目标应被清理")
	}
}

// 重复执行：不报错、版本不变、数据不变
func TestMigrate_Idempotent(t *testing.T) {
	db := openTestDB(t)
	for i := 0; i < 2; i++ {
		if err := migrate(db); err != nil {
			t.Fatalf("第 %d 次 migrate: %v", i+1, err)
		}
	}
	if got, want := schemaVersion(t, db), latestVersion(t); got != want {
		t.Errorf("版本 = %d, want %d", got, want)
	}
}

// 部署器删除后它的两张表不再有读写方：已有库升级时连同旧数据删掉，新库里也没有
func TestMigrate_DropsDeployerTables(t *testing.T) {
	ctx := context.Background()
	tableExists := func(db *sql.DB, name string) bool {
		return scalar(t, db, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name) == "1"
	}

	// 上一版本（第 4 版）的已有库：两张表都在，且有数据
	existing := openTestDB(t)
	p, err := goose.NewProvider(goose.DialectSQLite3, existing, schema.FS)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 4); err != nil {
		t.Fatalf("迁到第 4 版: %v", err)
	}
	if _, err := existing.Exec(`INSERT INTO deploy_config (cluster_id, repo_url, created_at, updated_at) VALUES ('c1', 'https://example.com/repo.git', '2026-11-01T00:00:00Z', '2026-11-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := existing.Exec(`INSERT INTO deploy_history (cluster_id, path, namespace, commit_sha, deployed_at) VALUES ('c1', 'apps/x', 'default', 'abc1234', '2026-11-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(existing); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, table := range []string{"deploy_config", "deploy_history"} {
		if tableExists(existing, table) {
			t.Errorf("已有库升级后 %s 应已删除", table)
		}
	}

	// 新库
	fresh := openTestDB(t)
	if err := migrate(fresh); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, table := range []string{"deploy_config", "deploy_history"} {
		if tableExists(fresh, table) {
			t.Errorf("新库里不应有 %s", table)
		}
	}
}

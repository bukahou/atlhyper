package repo_test

import (
	"path/filepath"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
)

func openAt(t *testing.T, path string) *database.DB {
	t.Helper()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: path}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	return db
}

// 模型目录种子逐行幂等（用户裁定 D2）：已有库重启后补齐缺失的内置模型，已有行不动
func TestModelCatalogSeed_IdempotentAndBackfills(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.db")

	db := openAt(t, path)
	// 模拟「旧版本建的库」：没有 bedrock 行，且某个内置行被改过
	if _, err := db.Conn.Exec(`DELETE FROM ai_provider_models WHERE provider = 'bedrock'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Conn.Exec(`UPDATE ai_provider_models SET context_window = 12345 WHERE provider = 'openai' AND model = 'gpt-4o'`); err != nil {
		t.Fatal(err)
	}
	db.Conn.Close()

	db = openAt(t, path) // 再次启动：执行迁移与种子
	defer db.Conn.Close()

	rows, err := db.Conn.Query(`SELECT model, context_window FROM ai_provider_models WHERE provider = 'bedrock' ORDER BY sort_order`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int{}
	for rows.Next() {
		var m string
		var cw int
		if err := rows.Scan(&m, &cw); err != nil {
			t.Fatal(err)
		}
		got[m] = cw
	}
	rows.Close()
	want := map[string]int{
		"qwen.qwen3-235b-a22b-2507-v1:0": 256000,
		"openai.gpt-oss-120b-1:0":        128000,
		"minimax.minimax-m2.5":           196000,
	}
	for m, cw := range want {
		if got[m] != cw {
			t.Errorf("bedrock %s context_window = %d, want %d（ai/context.go 据此裁剪消息）", m, got[m], cw)
		}
	}

	var cw int
	if err := db.Conn.QueryRow(`SELECT context_window FROM ai_provider_models WHERE provider = 'openai' AND model = 'gpt-4o'`).Scan(&cw); err != nil || cw != 12345 {
		t.Errorf("已有行不得被覆盖：%d %v", cw, err)
	}

	var n int
	db.Conn.QueryRow(`SELECT COUNT(*) FROM ai_provider_models WHERE provider = 'bedrock'`).Scan(&n)
	if n != 3 {
		t.Errorf("重复启动不得产生重复行：%d", n)
	}
}

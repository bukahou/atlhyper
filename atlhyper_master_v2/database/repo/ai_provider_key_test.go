package repo_test

import (
	"context"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
)

// 占位凭据：不是任何真实系统的 API Key
const testAPIKey = "sk-placeholder-not-a-real-key"

// API Key 加密落库、读出解密；加密密钥换过之后解不开：按没填处理，⛔ 不把密文当 API Key 用
func TestAIProvider_APIKeyEncryptedAndNeverLeaksCiphertext(t *testing.T) {
	db := openNotifyTestDB(t) // 设置加密密钥
	ctx := context.Background()
	p := &database.AIProvider{Name: "p", Provider: "openai", Model: "m", APIKey: testAPIKey}
	if err := db.AIProvider.Create(ctx, p); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.Conn.QueryRowContext(ctx, "SELECT api_key FROM ai_providers WHERE id = ?", p.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == testAPIKey || stored == "" {
		t.Fatalf("库里应是密文，实际 %q", stored)
	}
	got, err := db.AIProvider.GetByID(ctx, p.ID)
	if err != nil || got.APIKey != testAPIKey {
		t.Fatalf("读出应解密成原文: %v %q", err, got.APIKey)
	}

	// 换了加密密钥：解不开
	if err := repo.SetEncryptionSecret("another-test-secret-not-real"); err != nil {
		t.Fatal(err)
	}
	got, err = db.AIProvider.GetByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "" {
		t.Errorf("解不开时 API Key 应为空（按未填写处理），实际 %q", got.APIKey)
	}
	list, err := db.AIProvider.List(ctx)
	if err != nil || len(list) != 1 || list[0].APIKey != "" {
		t.Errorf("列表里同样不能带出密文: %v %+v", err, list)
	}
}

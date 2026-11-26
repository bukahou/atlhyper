package repo_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// 时间按秒存，测试里的时刻都取整秒
var tokenTestTime = time.Date(2026, 11, 26, 9, 30, 0, 0, time.UTC)

// issueToken 给账号写一枚令牌（上限放宽，不影响其他断言）
func issueToken(t *testing.T, db *database.DB, userID int64, tokenHash string) *database.APIToken {
	t.Helper()
	token := &database.APIToken{
		UserID: userID, Name: "token-" + tokenHash, TokenHash: tokenHash, TokenPrefix: "atlp_abc",
		CreatedBy: "admin", CreatedAt: tokenTestTime,
	}
	if err := db.APIToken.CreateWithinLimit(context.Background(), token, 100); err != nil {
		t.Fatal(err)
	}
	if token.ID == 0 {
		t.Fatal("签发后应回填 ID")
	}
	return token
}

func TestAPIToken_CreateAndGetByHash(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 0)
	expiresAt := tokenTestTime.Add(90 * 24 * time.Hour)

	token := &database.APIToken{
		UserID: account.ID, Name: "prod", TokenHash: "hash-1", TokenPrefix: "atlp_Ab3",
		ExpiresAt: &expiresAt, CreatedBy: "admin", CreatedAt: tokenTestTime,
	}
	if err := db.APIToken.CreateWithinLimit(ctx, token, 10); err != nil {
		t.Fatal(err)
	}

	got, err := db.APIToken.GetByHash(ctx, "hash-1")
	if err != nil || got == nil {
		t.Fatalf("GetByHash = %v, %v", got, err)
	}
	if got.ID != token.ID || got.UserID != account.ID || got.Name != "prod" || got.TokenPrefix != "atlp_Ab3" ||
		got.CreatedBy != "admin" || !got.CreatedAt.Equal(tokenTestTime) {
		t.Errorf("取回的令牌不对：%+v", got)
	}
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("到期时间不对：%v", got.ExpiresAt)
	}
	if got.LastUsedAt != nil || got.RevokedAt != nil {
		t.Errorf("新令牌不应有最近使用与吊销时间：%+v", got)
	}

	for _, hash := range []string{"", "no-such-hash"} {
		if got, err := db.APIToken.GetByHash(ctx, hash); err != nil || got != nil {
			t.Errorf("GetByHash(%q) 应返回 (nil, nil)：%+v, %v", hash, got, err)
		}
	}
}

// 每个账号未吊销的令牌有上限：满了拒绝签发，吊销一枚后又能签
func TestAPIToken_LimitCountsUnrevokedTokensPerAccount(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 0)
	other := createServiceAccount(t, db, "other", 0)
	const limit = 3

	create := func(userID int64, hash string) error {
		return db.APIToken.CreateWithinLimit(ctx, &database.APIToken{
			UserID: userID, Name: hash, TokenHash: hash, TokenPrefix: "atlp_abc", CreatedAt: tokenTestTime,
		}, limit)
	}
	var first int64
	for i := 0; i < limit; i++ {
		if err := create(account.ID, fmt.Sprintf("hash-%d", i)); err != nil {
			t.Fatalf("第 %d 枚：%v", i+1, err)
		}
	}
	if err := create(account.ID, "hash-over"); !errors.Is(err, database.ErrAPITokenLimitReached) {
		t.Fatalf("超出上限：err = %v, want ErrAPITokenLimitReached", err)
	}
	if token, _ := db.APIToken.GetByHash(ctx, "hash-over"); token != nil {
		t.Error("超出上限的令牌不应写入")
	}
	// 别的账号不受影响
	if err := create(other.ID, "hash-other"); err != nil {
		t.Errorf("别的账号应能签发：%v", err)
	}

	tokens, _ := db.APIToken.ListByUser(ctx, account.ID)
	first = tokens[len(tokens)-1].ID
	if err := db.APIToken.Revoke(ctx, account.ID, first, tokenTestTime); err != nil {
		t.Fatal(err)
	}
	if err := create(account.ID, "hash-after-revoke"); err != nil {
		t.Errorf("吊销一枚后应能再签发：%v", err)
	}
}

func TestAPIToken_CreateForUnknownAccount(t *testing.T) {
	db := openTestDB(t)
	err := db.APIToken.CreateWithinLimit(context.Background(), &database.APIToken{
		UserID: 999, Name: "x", TokenHash: "hash-x", TokenPrefix: "atlp_abc", CreatedAt: tokenTestTime,
	}, 10)
	if !errors.Is(err, database.ErrServiceAccountNotFound) {
		t.Errorf("err = %v, want ErrServiceAccountNotFound", err)
	}
}

// 吊销保留行、记下吊销时间；再吊销一次不改第一次的时间；只能吊销本账号的令牌
func TestAPIToken_Revoke(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 0)
	other := createServiceAccount(t, db, "other", 0)
	token := issueToken(t, db, account.ID, "hash-1")

	if err := db.APIToken.Revoke(ctx, account.ID, token.ID, tokenTestTime); err != nil {
		t.Fatal(err)
	}
	if err := db.APIToken.Revoke(ctx, account.ID, token.ID, tokenTestTime.Add(time.Hour)); err != nil {
		t.Fatalf("重复吊销不应报错：%v", err)
	}
	got, _ := db.APIToken.GetByHash(ctx, "hash-1")
	if got == nil || got.RevokedAt == nil || !got.RevokedAt.Equal(tokenTestTime) {
		t.Errorf("吊销时间应保留第一次的：%+v", got)
	}

	cases := []struct {
		name            string
		userID, tokenID int64
	}{
		{"别的账号的令牌", other.ID, token.ID},
		{"不存在的令牌", account.ID, 999},
	}
	for _, c := range cases {
		if err := db.APIToken.Revoke(ctx, c.userID, c.tokenID, tokenTestTime); !errors.Is(err, database.ErrAPITokenNotFound) {
			t.Errorf("%s：err = %v, want ErrAPITokenNotFound", c.name, err)
		}
	}
}

func TestAPIToken_Touch(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 0)
	token := issueToken(t, db, account.ID, "hash-1")

	usedAt := tokenTestTime.Add(5 * time.Minute)
	if err := db.APIToken.Touch(ctx, token.ID, usedAt); err != nil {
		t.Fatal(err)
	}
	got, _ := db.APIToken.GetByHash(ctx, "hash-1")
	if got.LastUsedAt == nil || !got.LastUsedAt.Equal(usedAt) {
		t.Errorf("最近使用时间 = %v, want %v", got.LastUsedAt, usedAt)
	}
}

// 列表：按账号取时新签发的在前；全部列出时先按账号再按新旧
func TestAPIToken_List(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	first := createServiceAccount(t, db, "haro", 0)
	second := createServiceAccount(t, db, "other", 0)
	older := issueToken(t, db, first.ID, "hash-1")
	newer := issueToken(t, db, first.ID, "hash-2")
	otherToken := issueToken(t, db, second.ID, "hash-3")

	byUser, err := db.APIToken.ListByUser(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(byUser) != 2 || byUser[0].ID != newer.ID || byUser[1].ID != older.ID {
		t.Errorf("ListByUser 顺序不对：%+v", byUser)
	}
	if empty, err := db.APIToken.ListByUser(ctx, 999); err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("没有令牌时应返回空切片：%v, %v", empty, err)
	}

	all, err := db.APIToken.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 || all[0].ID != newer.ID || all[1].ID != older.ID || all[2].ID != otherToken.ID {
		t.Errorf("List 顺序不对：%+v", all)
	}
}

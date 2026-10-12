package repo_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
)

// 真实 SQLite（临时文件 + 全量迁移），验证的是 SQL 与约束本身
func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "t.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())
	return db
}

const testIssuer = "https://issuer.example.com"

func federatedUser(name string) *database.User {
	return &database.User{Username: name, DisplayName: name, Role: 0, Status: 1}
}

func TestUserIdentity_CreateAndFind(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	u := federatedUser("alice")
	if err := db.UserIdentity.CreateUserWithIdentity(ctx, u, testIssuer, "sub-1"); err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 {
		t.Fatal("建号后应回填 user.ID")
	}

	id, found, err := db.UserIdentity.FindUserID(ctx, testIssuer, "sub-1")
	if err != nil || !found || id != u.ID {
		t.Errorf("FindUserID = %d, %v, %v；want %d", id, found, err, u.ID)
	}
	if _, found, _ := db.UserIdentity.FindUserID(ctx, "https://other.example.com", "sub-1"); found {
		t.Error("同一 subject 在不同 issuer 下是不同身份")
	}
}

// 同一 (issuer, subject) 不能建第二个账号；失败时整笔回滚，不留孤儿 users 行
func TestUserIdentity_DuplicateIdentityRollsBack(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if err := db.UserIdentity.CreateUserWithIdentity(ctx, federatedUser("alice"), testIssuer, "sub-1"); err != nil {
		t.Fatal(err)
	}

	err := db.UserIdentity.CreateUserWithIdentity(ctx, federatedUser("alice-2"), testIssuer, "sub-1")
	if !errors.Is(err, database.ErrIdentityExists) {
		t.Fatalf("err = %v，want ErrIdentityExists", err)
	}
	if u, _ := db.User.GetByUsername(ctx, "alice-2"); u != nil {
		t.Error("身份冲突时 users 行必须一起回滚")
	}
}

func TestUserIdentity_UsernameTaken(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if err := db.User.Create(ctx, &database.User{Username: "alice", PasswordHash: "x", Role: 1, Status: 1}); err != nil {
		t.Fatal(err)
	}
	err := db.UserIdentity.CreateUserWithIdentity(ctx, federatedUser("alice"), testIssuer, "sub-1")
	if !errors.Is(err, database.ErrUsernameTaken) {
		t.Fatalf("err = %v，want ErrUsernameTaken", err)
	}
	if _, found, _ := db.UserIdentity.FindUserID(ctx, testIssuer, "sub-1"); found {
		t.Error("用户名冲突时身份行不得写入")
	}
}

// 外部身份账号对密码登录而言与「用户不存在」相同
func TestUser_PasswordLoginLookupExcludesFederated(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if err := db.User.Create(ctx, &database.User{Username: "local", PasswordHash: "hash", Role: 1, Status: 1}); err != nil {
		t.Fatal(err)
	}
	if err := db.UserIdentity.CreateUserWithIdentity(ctx, federatedUser("ext"), testIssuer, "sub-1"); err != nil {
		t.Fatal(err)
	}

	if u, err := db.User.GetByUsernameForPasswordLogin(ctx, "local"); err != nil || u == nil {
		t.Errorf("本地账号应可查到：%v %v", u, err)
	}
	if u, err := db.User.GetByUsernameForPasswordLogin(ctx, "ext"); err != nil || u != nil {
		t.Errorf("外部身份账号应查不到：%v %v", u, err)
	}
}

func TestUserIdentity_ListFederatedUserIDs(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	local := &database.User{Username: "local", PasswordHash: "hash", Role: 1, Status: 1}
	if err := db.User.Create(ctx, local); err != nil {
		t.Fatal(err)
	}
	ext := federatedUser("ext")
	if err := db.UserIdentity.CreateUserWithIdentity(ctx, ext, testIssuer, "sub-1"); err != nil {
		t.Fatal(err)
	}

	ids, err := db.UserIdentity.ListFederatedUserIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !ids[ext.ID] || ids[local.ID] {
		t.Errorf("federated ids = %v", ids)
	}
}

package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/auth/apitoken"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
)

// 服务账号与访问令牌的完整流程，接真实的 SQLite（装配方式与 master.go 一致）：
// 新建账号 → 签发 → 校验 → 改等级 → 停用 / 启用 → 吊销 → 删除账号。
// 关注的是各层接起来之后的行为：管理接口签发的令牌，校验器经统一 Service 认得出；改动立即生效。
func TestServiceAccountFlow(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "master.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())

	q := query.NewQueryService(query.QueryServiceDeps{
		UserRepo: db.User, IdentityRepo: db.UserIdentity,
		ServiceAccountRepo: db.ServiceAccount, APITokenRepo: db.APIToken,
	})
	svc := service.NewService(q, nil, nil, nil, nil, operations.NewUserService(db.User, db.UserIdentity), nil, nil,
		operations.NewServiceAccountService(db.ServiceAccount, db.APIToken))
	authenticator := apitoken.NewAuthenticator(svc, svc, time.Now)

	level := func(v int) *int { return &v }

	// 1. 新建：出现在服务账号列表里，不出现在用户列表里，也不能密码登录
	accountID, err := svc.CreateServiceAccount(ctx, &model.CreateServiceAccountRequest{Name: "haro", DisplayName: "Haro", Role: level(0)})
	if err != nil {
		t.Fatal(err)
	}
	if views, _ := svc.ListServiceAccounts(ctx); len(views) != 1 || views[0].Name != "haro" || !views[0].CanIssueToken {
		t.Fatalf("服务账号列表不对：%+v", views)
	}
	if users, _ := svc.ListUsers(ctx); len(users) != 0 {
		t.Errorf("用户列表不应含服务账号：%+v", users)
	}
	if user, _ := svc.GetUserByUsernameForPasswordLogin(ctx, "haro"); user != nil {
		t.Error("服务账号不能密码登录")
	}

	// 2. 签发：校验器认得出，等级是账号当前的等级；最近使用时间被记下
	issued, err := svc.IssueAPIToken(ctx, accountID, &model.IssueAPITokenRequest{Name: "prod"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	caller, err := authenticator.Authenticate(ctx, issued.Token)
	if err != nil || caller.UserID != accountID || caller.Username != "haro" || caller.Level != 0 || caller.TokenName != "prod" {
		t.Fatalf("签发的令牌应能通过校验：%+v, %v", caller, err)
	}
	view, _ := svc.GetServiceAccount(ctx, accountID)
	if view == nil || len(view.Tokens) != 1 || view.Tokens[0].Status != model.APITokenStatusActive || view.Tokens[0].LastUsedAt == "" ||
		view.Tokens[0].TokenPrefix != issued.TokenPrefix || view.Tokens[0].CreatedBy != "admin" {
		t.Fatalf("签发后的那一行不对：%+v", view)
	}

	// 3. 改等级：同一枚令牌下一次校验就是新等级
	enabled, disabled := true, false
	displayName := "Haro"
	if err := svc.UpdateServiceAccount(ctx, accountID, &model.UpdateServiceAccountRequest{DisplayName: &displayName, Role: level(1), Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	if caller, _ := authenticator.Authenticate(ctx, issued.Token); caller.Level != 1 {
		t.Errorf("改等级后 Level = %d, want 1", caller.Level)
	}

	// 4. 停用：立即无效；启用后恢复
	if err := svc.UpdateServiceAccount(ctx, accountID, &model.UpdateServiceAccountRequest{DisplayName: &displayName, Role: level(1), Enabled: &disabled}); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticator.Authenticate(ctx, issued.Token); !errors.Is(err, apitoken.ErrInvalidToken) {
		t.Errorf("停用后：err = %v, want ErrInvalidToken", err)
	}
	if err := svc.UpdateServiceAccount(ctx, accountID, &model.UpdateServiceAccountRequest{DisplayName: &displayName, Role: level(1), Enabled: &enabled}); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticator.Authenticate(ctx, issued.Token); err != nil {
		t.Errorf("启用后应恢复：%v", err)
	}

	// 5. 吊销：立即无效，行保留、状态为 revoked
	if err := svc.RevokeAPIToken(ctx, accountID, view.Tokens[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticator.Authenticate(ctx, issued.Token); !errors.Is(err, apitoken.ErrInvalidToken) {
		t.Errorf("吊销后：err = %v, want ErrInvalidToken", err)
	}
	if view, _ := svc.GetServiceAccount(ctx, accountID); len(view.Tokens) != 1 || view.Tokens[0].Status != model.APITokenStatusRevoked {
		t.Errorf("吊销后的令牌应保留并标为 revoked：%+v", view)
	}

	// 6. 删除账号：另一枚令牌随之失效
	second, err := svc.IssueAPIToken(ctx, accountID, &model.IssueAPITokenRequest{Name: "second"}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteServiceAccount(ctx, accountID); err != nil {
		t.Fatal(err)
	}
	if _, err := authenticator.Authenticate(ctx, second.Token); !errors.Is(err, apitoken.ErrInvalidToken) {
		t.Errorf("删除账号后：err = %v, want ErrInvalidToken", err)
	}
	if views, _ := svc.ListServiceAccounts(ctx); len(views) != 0 {
		t.Errorf("删除后列表应为空：%+v", views)
	}
}

// 人的账号即使有令牌行也通不过校验：令牌只属于服务账号
func TestServiceAccountFlow_HumanCannotUseAPIToken(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "master.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())
	q := query.NewQueryService(query.QueryServiceDeps{UserRepo: db.User, ServiceAccountRepo: db.ServiceAccount, APITokenRepo: db.APIToken})
	svc := service.NewService(q, nil, nil, nil, nil, nil, nil, nil, operations.NewServiceAccountService(db.ServiceAccount, db.APIToken))

	human := &database.User{Username: "alice", PasswordHash: "placeholder-not-a-real-hash", Role: 2, Status: database.UserStatusActive}
	if err := db.User.Create(ctx, human); err != nil {
		t.Fatal(err)
	}
	issued, err := apitoken.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.APIToken.CreateWithinLimit(ctx, &database.APIToken{
		UserID: human.ID, Name: "x", TokenHash: issued.Hash, TokenPrefix: issued.Prefix, CreatedAt: time.Now(),
	}, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := apitoken.NewAuthenticator(svc, svc, time.Now).Authenticate(ctx, issued.Plaintext); !errors.Is(err, apitoken.ErrInvalidToken) {
		t.Errorf("人的令牌：err = %v, want ErrInvalidToken", err)
	}
}

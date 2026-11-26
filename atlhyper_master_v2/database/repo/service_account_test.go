package repo_test

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"AtlHyper/atlhyper_master_v2/database"
)

func createServiceAccount(t *testing.T, db *database.DB, name string, role int) *database.User {
	t.Helper()
	account := &database.User{Username: name, DisplayName: name + " 显示名", Role: role, Status: database.UserStatusActive}
	if err := db.ServiceAccount.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if account.ID == 0 {
		t.Fatal("建号后应回填 ID")
	}
	return account
}

func createHuman(t *testing.T, db *database.DB, name string) *database.User {
	t.Helper()
	user := &database.User{Username: name, PasswordHash: "placeholder-not-a-real-hash", Role: 3, Status: database.UserStatusActive}
	if err := db.User.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	return user
}

func TestServiceAccount_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	created := createServiceAccount(t, db, "haro", 1)

	got, err := db.ServiceAccount.GetByID(ctx, created.ID)
	if err != nil || got == nil {
		t.Fatalf("GetByID = %v, %v", got, err)
	}
	if got.Username != "haro" || got.DisplayName != "haro 显示名" || got.Role != 1 || got.Status != database.UserStatusActive ||
		got.Kind != database.UserKindService || got.CreatedAt.IsZero() {
		t.Errorf("取回的服务账号不对：%+v", got)
	}
	// 通用的按 id 读取也能读到它（访问令牌校验要用），种类如实带出
	if user, _ := db.User.GetByID(ctx, created.ID); user == nil || !user.IsServiceAccount() {
		t.Errorf("User.GetByID 应读到服务账号：%+v", user)
	}
}

// 服务账号不能密码登录：库里的口令哈希是占位值（任何口令都比对不上），密码登录的查询也把它当作不存在
func TestServiceAccount_CannotPasswordLogin(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	created := createServiceAccount(t, db, "haro", 0)

	stored, _ := db.User.GetByID(ctx, created.ID)
	if stored.PasswordHash != database.ServiceAccountPasswordHash {
		t.Errorf("password_hash = %q, want 占位值", stored.PasswordHash)
	}
	for _, password := range []string{"", database.ServiceAccountPasswordHash, "placeholder-not-a-real-password"} {
		if bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(password)) == nil {
			t.Errorf("占位值不应与任何口令相符：%q", password)
		}
	}
	if user, err := db.User.GetByUsernameForPasswordLogin(ctx, "haro"); err != nil || user != nil {
		t.Errorf("密码登录应查不到服务账号：%+v, %v", user, err)
	}
}

func TestServiceAccount_NameMustBeUniqueAcrossKinds(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	createHuman(t, db, "alice")
	createServiceAccount(t, db, "haro", 0)

	for _, name := range []string{"alice", "haro"} {
		err := db.ServiceAccount.Create(ctx, &database.User{Username: name, Status: database.UserStatusActive})
		if !errors.Is(err, database.ErrUsernameTaken) {
			t.Errorf("%s：err = %v, want ErrUsernameTaken", name, err)
		}
	}
}

// 人与服务账号各列各的
func TestServiceAccount_ListsAreSeparatedByKind(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	createHuman(t, db, "alice")
	createServiceAccount(t, db, "zeta-bot", 0)
	createServiceAccount(t, db, "haro", 2)

	accounts, err := db.ServiceAccount.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 || accounts[0].Username != "haro" || accounts[1].Username != "zeta-bot" {
		t.Errorf("服务账号列表应按账号名排序、只含服务账号：%+v", accounts)
	}

	humans, err := db.User.ListHumans(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(humans) != 1 || humans[0].Username != "alice" || humans[0].Kind != database.UserKindHuman {
		t.Errorf("用户列表应只含人：%+v", humans)
	}
}

func TestServiceAccount_GetByIDIgnoresHumans(t *testing.T) {
	db := openTestDB(t)
	human := createHuman(t, db, "alice")
	if got, err := db.ServiceAccount.GetByID(context.Background(), human.ID); err != nil || got != nil {
		t.Errorf("人的 id 不是服务账号，应返回 (nil, nil)：%+v, %v", got, err)
	}
}

func TestServiceAccount_Update(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 0)

	account.DisplayName, account.Role, account.Status = "Haro 语音助手", 2, database.UserStatusDisabled
	if err := db.ServiceAccount.Update(ctx, account); err != nil {
		t.Fatal(err)
	}
	got, _ := db.ServiceAccount.GetByID(ctx, account.ID)
	if got.DisplayName != "Haro 语音助手" || got.Role != 2 || got.Status != database.UserStatusDisabled || got.Username != "haro" {
		t.Errorf("修改后的服务账号不对：%+v", got)
	}

	human := createHuman(t, db, "alice")
	for _, id := range []int64{human.ID, 999} {
		err := db.ServiceAccount.Update(ctx, &database.User{ID: id, DisplayName: "x", Role: 0, Status: 0})
		if !errors.Is(err, database.ErrServiceAccountNotFound) {
			t.Errorf("id %d：err = %v, want ErrServiceAccountNotFound", id, err)
		}
	}
	if stored, _ := db.User.GetByID(ctx, human.ID); stored.Role != 3 || stored.Status != database.UserStatusActive {
		t.Errorf("人的账号不应被服务账号的修改波及：%+v", stored)
	}
}

// 删除服务账号时它的令牌随外键一起删；人的 id 删不动
func TestServiceAccount_DeleteCascadesTokens(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 0)
	issueToken(t, db, account.ID, "hash-1")

	if err := db.ServiceAccount.Delete(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := db.ServiceAccount.GetByID(ctx, account.ID); got != nil {
		t.Error("删除后不应再查到")
	}
	if token, _ := db.APIToken.GetByHash(ctx, "hash-1"); token != nil {
		t.Error("令牌应随账号一起删除")
	}
	if err := db.ServiceAccount.Delete(ctx, account.ID); !errors.Is(err, database.ErrServiceAccountNotFound) {
		t.Errorf("再删一次：err = %v, want ErrServiceAccountNotFound", err)
	}

	human := createHuman(t, db, "alice")
	if err := db.ServiceAccount.Delete(ctx, human.ID); !errors.Is(err, database.ErrServiceAccountNotFound) {
		t.Errorf("删人的 id：err = %v, want ErrServiceAccountNotFound", err)
	}
	if stored, _ := db.User.GetByID(ctx, human.ID); stored == nil {
		t.Error("人的账号不应被删")
	}
}

// 用户管理的写入只作用于人：拿服务账号的 id 调也改不动、删不掉
func TestUserWrites_OnlyAffectHumans(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	account := createServiceAccount(t, db, "haro", 1)

	if err := db.User.Update(ctx, &database.User{ID: account.ID, Role: 3, Status: database.UserStatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := db.User.Delete(ctx, account.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := db.ServiceAccount.GetByID(ctx, account.ID)
	if got == nil || got.Role != 1 {
		t.Errorf("服务账号不应被用户管理的写入改动：%+v", got)
	}
}

// 建号走哪条路就是哪种：用户管理与外部身份登录建的都是人，调用方传的种类不作数
func TestUserCreate_AlwaysHuman(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	local := &database.User{Username: "local", PasswordHash: "x", Status: 1, Kind: database.UserKindService}
	if err := db.User.Create(ctx, local); err != nil {
		t.Fatal(err)
	}
	federated := &database.User{Username: "ext", Status: 1, Kind: database.UserKindService}
	if err := db.UserIdentity.CreateUserWithIdentity(ctx, federated, testIssuer, "sub-1"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{local.ID, federated.ID} {
		if got, _ := db.User.GetByID(ctx, id); got == nil || got.Kind != database.UserKindHuman {
			t.Errorf("id %d 应为人：%+v", id, got)
		}
	}
}

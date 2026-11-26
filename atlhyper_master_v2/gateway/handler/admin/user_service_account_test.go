package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
)

// userHandlerOnRealDB 用户管理 handler 接真实的 SQLite 与服务层（与 master.go 的装配方式一致），
// 库里有一个人的管理员 ops 与一个服务账号 haro
func userHandlerOnRealDB(t *testing.T) (*UserHandler, *database.DB, int64) {
	t.Helper()
	db, err := database.NewDatabase(database.Config{Type: "sqlite", Path: filepath.Join(t.TempDir(), "master.db")}, sqlite.NewDialect())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Conn.Close() })
	repo.Init(db, sqlite.NewDialect())

	q := query.NewQueryService(query.QueryServiceDeps{
		UserRepo: db.User, IdentityRepo: db.UserIdentity, ServiceAccountRepo: db.ServiceAccount, APITokenRepo: db.APIToken,
	})
	svc := service.NewService(q, nil, nil, nil, nil, operations.NewUserService(db.User, db.UserIdentity), nil, nil,
		operations.NewServiceAccountService(db.ServiceAccount, db.APIToken))

	ctx := context.Background()
	if err := db.User.Create(ctx, &database.User{Username: "ops", PasswordHash: "placeholder-not-a-real-hash", Role: 3, Status: 1}); err != nil {
		t.Fatal(err)
	}
	role := 1
	accountID, err := svc.CreateServiceAccount(ctx, &model.CreateServiceAccountRequest{Name: "haro", Role: &role})
	if err != nil {
		t.Fatal(err)
	}
	return NewUserHandler(svc, svc), db, accountID
}

func httptestGet(h http.HandlerFunc) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec
}

// 用户管理只管人：列表里没有服务账号；对服务账号的 id 改角色、启停、删除一律按「用户不存在」处理，什么都不改
func TestUserHandler_ServiceAccountsAreNotUsers(t *testing.T) {
	h, db, accountID := userHandlerOnRealDB(t)

	rec := httptestGet(h.List)
	var listed struct {
		Data []UserDTO `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listed); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("列表：%d %s", rec.Code, rec.Body)
	}
	if len(listed.Data) != 1 || listed.Data[0].Username != "ops" {
		t.Errorf("用户列表不应含服务账号：%+v", listed.Data)
	}

	cases := map[string]func() int{
		"改角色": func() int { return post(h.UpdateRole, fmt.Sprintf(`{"userId":%d,"role":3}`, accountID)).Code },
		"停用":  func() int { return post(h.UpdateStatus, fmt.Sprintf(`{"userId":%d,"status":0}`, accountID)).Code },
		"删除":  func() int { return post(h.Delete, fmt.Sprintf(`{"userId":%d}`, accountID)).Code },
	}
	for name, call := range cases {
		if code := call(); code != http.StatusNotFound {
			t.Errorf("%s服务账号：status = %d, want 404", name, code)
		}
	}
	account, _ := db.ServiceAccount.GetByID(context.Background(), accountID)
	if account == nil || account.Role != 1 || account.Status != 1 {
		t.Errorf("服务账号不应被用户管理接口改动：%+v", account)
	}
}

// 服务账号不能密码登录：与未知用户名走同一条路（401），连占位的口令哈希本身也登不进去
func TestUserHandler_PasswordLoginRejectsServiceAccount(t *testing.T) {
	h, _, _ := userHandlerOnRealDB(t)
	for _, password := range []string{database.ServiceAccountPasswordHash, "placeholder-not-a-real-password"} {
		body := fmt.Sprintf(`{"username":"haro","password":%q}`, password)
		if rec := post(h.Login, body); rec.Code != http.StatusUnauthorized {
			t.Errorf("服务账号登录：status = %d, want 401（%s）", rec.Code, rec.Body)
		}
	}
}

// 新建用户时账号名与服务账号重复也算重名
func TestUserHandler_RegisterRejectsServiceAccountName(t *testing.T) {
	h, _, _ := userHandlerOnRealDB(t)
	if rec := post(h.Register, `{"username":"haro","password":"placeholder-not-real"}`); rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409（%s）", rec.Code, rec.Body)
	}
}

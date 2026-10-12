package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/atlhyper_master_v2/database"
)

type fakeUserLoader struct {
	users map[int64]*database.User
	err   error
}

func (f *fakeUserLoader) GetUserByID(_ context.Context, id int64) (*database.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.users[id], nil
}

func withTestJWTSecret(t *testing.T) {
	t.Helper()
	prev := config.GlobalConfig.JWT
	config.GlobalConfig.JWT.SecretKey = "test-secret-not-real"
	t.Cleanup(func() { config.GlobalConfig.JWT = prev })
}

// resolve 跑一次 Authenticate，返回 handler 里看到的 Principal。
func resolve(t *testing.T, loader UserLoader, authHeader string) Principal {
	t.Helper()
	var got Principal
	h := Authenticate(loader)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = GetPrincipal(r.Context())
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	h.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

func bearer(t *testing.T, userID int64, tokenRole int) string {
	t.Helper()
	tok, err := GenerateToken(userID, "u", tokenRole)
	if err != nil {
		t.Fatal(err)
	}
	return "Bearer " + tok
}

func TestAuthenticate_NoTokenIsGuest(t *testing.T) {
	p := resolve(t, &fakeUserLoader{}, "")
	if !p.IsGuest() || p.Level != RoleMember {
		t.Errorf("无 token 应为访客 0 级，得到 %+v", p)
	}
}

// 角色以库中当前值为准：token 里写着 Admin，库中已降为 Viewer → Viewer。
func TestAuthenticate_LevelComesFromDatabase(t *testing.T) {
	withTestJWTSecret(t)
	loader := &fakeUserLoader{users: map[int64]*database.User{
		7: {ID: 7, Username: "alice", Role: RoleViewer, Status: 1},
	}}
	p := resolve(t, loader, bearer(t, 7, RoleAdmin))
	if p.UserID != 7 || p.Username != "alice" || p.Level != RoleViewer {
		t.Errorf("应取库中角色 Viewer，得到 %+v", p)
	}
}

// 身份无效一律退化为访客，并记下原因（受限接口据此返回 401 提示重新登录）。
func TestAuthenticate_InvalidIdentityDegradesToGuest(t *testing.T) {
	withTestJWTSecret(t)
	cases := []struct {
		name   string
		loader *fakeUserLoader
		header string
	}{
		{"格式错误", &fakeUserLoader{}, "Token abc"},
		{"签名无效", &fakeUserLoader{}, "Bearer not.a.jwt"},
		{"用户已删除", &fakeUserLoader{users: map[int64]*database.User{}}, ""},
		{"用户已禁用", &fakeUserLoader{users: map[int64]*database.User{7: {ID: 7, Role: RoleAdmin, Status: 0}}}, ""},
		{"查库失败：失败即关", &fakeUserLoader{err: errors.New("db down")}, ""},
	}
	for _, c := range cases {
		header := c.header
		if header == "" {
			header = bearer(t, 7, RoleAdmin)
		}
		p := resolve(t, c.loader, header)
		if !p.IsGuest() || p.Level != RoleMember {
			t.Errorf("%s: 应退化为访客，得到 %+v", c.name, p)
		}
		if p.Rejected == "" {
			t.Errorf("%s: 应记录拒绝原因", c.name)
		}
	}
}

func TestRequirePermission(t *testing.T) {
	ok := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }
	cases := []struct {
		name string
		p    Principal
		perm Permission
		want int
	}{
		{"访客读集群", Principal{Level: RoleMember}, PermClusterRead, http.StatusOK},
		{"访客读日志正文：需登录", Principal{Level: RoleMember}, PermLogsRead, http.StatusUnauthorized},
		{"成员读日志正文：已登录但不够", Principal{UserID: 3, Level: RoleMember}, PermLogsRead, http.StatusForbidden},
		{"Viewer 读日志正文", Principal{UserID: 3, Level: RoleViewer}, PermLogsRead, http.StatusOK},
		{"Viewer 执行操作", Principal{UserID: 3, Level: RoleViewer}, PermClusterOperate, http.StatusForbidden},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil).WithContext(WithPrincipal(context.Background(), c.p))
		RequirePermission(c.perm, ok)(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: code = %d, want %d", c.name, rec.Code, c.want)
		}
	}
}

// 既有调用方（审计、AI 会话、用户管理）通过这三个函数取身份，语义保持不变。
func TestContextAccessors_ReadPrincipal(t *testing.T) {
	ctx := WithPrincipal(context.Background(), Principal{UserID: 9, Username: "bob", Level: RoleOperator})
	if id, ok := GetUserID(ctx); !ok || id != 9 {
		t.Errorf("GetUserID = %d, %v", id, ok)
	}
	if name, ok := GetUsername(ctx); !ok || name != "bob" {
		t.Errorf("GetUsername = %q, %v", name, ok)
	}
	if role, ok := GetRole(ctx); !ok || role != RoleOperator {
		t.Errorf("GetRole = %d, %v", role, ok)
	}
	if _, ok := GetUserID(WithPrincipal(context.Background(), Principal{})); ok {
		t.Error("访客没有 user id")
	}
}

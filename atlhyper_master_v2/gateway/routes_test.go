package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/service"
)

// 只实现主体解析需要的方法；受限路由在权限检查处就被拦下，不会走到 handler。
type stubRouterSvc struct {
	service.Service
	users map[int64]*database.User
}

func (s *stubRouterSvc) GetUserByID(_ context.Context, id int64) (*database.User, error) {
	return s.users[id], nil
}

type stubAudit struct{ database.AuditRepository }

func (stubAudit) Create(context.Context, *database.AuditLog) error { return nil }

type stubAI struct{ ai.AIService }
type stubGitHub struct{ github.Client }

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	prev := config.GlobalConfig.JWT
	config.GlobalConfig.JWT.SecretKey = "test-secret-not-real"
	t.Cleanup(func() { config.GlobalConfig.JWT = prev })

	svc := &stubRouterSvc{users: map[int64]*database.User{
		1: {ID: 1, Username: "member", Role: middleware.RoleMember, Status: 1},
		2: {ID: 2, Username: "viewer", Role: middleware.RoleViewer, Status: 1},
		3: {ID: 3, Username: "operator", Role: middleware.RoleOperator, Status: 1},
	}}
	return NewRouter(svc, &database.DB{Audit: stubAudit{}}, stubAI{}, nil, stubGitHub{}, nil).Handler()
}

func call(t *testing.T, h http.Handler, method, path string, userID int64) int {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if userID > 0 {
		tok, err := middleware.GenerateToken(userID, "x", middleware.RoleAdmin) // token 里的角色不被信任
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

// 内容级 / 凭据 / 写操作 / 配置接口：访客 401，成员 403（成员与访客权限相同，只是已识别身份）。
func TestRoutes_LevelZeroCannotReachSensitive(t *testing.T) {
	h := testRouter(t)
	sensitive := []struct{ method, path string }{
		{"GET", "/api/v2/observe/logs/query"},
		{"GET", "/api/v2/observe/traces"},
		{"GET", "/api/v2/observe/traces/abc"},
		{"GET", "/api/v2/observe/traces/correlations"},
		{"GET", "/api/v2/commands/history"},
		{"GET", "/api/v2/commands/abc"},
		{"GET", "/api/v2/ai/conversations"},
		{"POST", "/api/v2/ai/chat"},
		{"GET", "/api/github/connection"},
		{"GET", "/api/deploy/history"},
		{"GET", "/api/v2/secrets"},
		{"GET", "/api/v2/configmaps/x"},
		{"POST", "/api/v2/ops/secrets/data"},
		{"POST", "/api/v2/ops/pods/logs"},
		{"POST", "/api/v2/commands"},
		{"GET", "/api/v2/audit/logs"},
		{"GET", "/api/v2/ai/providers"},
		{"GET", "/api/v2/user/list"},
		{"POST", "/api/v2/user/update-role"},
	}
	for _, c := range sensitive {
		if got := call(t, h, c.method, c.path, 0); got != http.StatusUnauthorized {
			t.Errorf("访客 %s %s = %d, want 401", c.method, c.path, got)
		}
		if got := call(t, h, c.method, c.path, 1); got != http.StatusForbidden {
			t.Errorf("成员 %s %s = %d, want 403", c.method, c.path, got)
		}
	}
}

// 分档边界：Viewer 读不到凭据、做不了操作；Operator 管不了用户。
func TestRoutes_LevelBoundaries(t *testing.T) {
	h := testRouter(t)
	cases := []struct {
		method, path string
		userID       int64
		want         int
	}{
		{"GET", "/api/v2/secrets", 2, http.StatusForbidden},
		{"POST", "/api/v2/ops/deployments/scale", 2, http.StatusForbidden},
		{"GET", "/api/v2/user/list", 3, http.StatusForbidden},
		{"POST", "/api/v2/ai/providers/1", 3, http.StatusForbidden},
	}
	for _, c := range cases {
		if got := call(t, h, c.method, c.path, c.userID); got != c.want {
			t.Errorf("user %d %s %s = %d, want %d", c.userID, c.method, c.path, got, c.want)
		}
	}
}

// 无需主体的路由不受 token 状态影响：过期 / 无效 token 也能登录。
func TestRoutes_OpenRoutesIgnoreBadToken(t *testing.T) {
	h := testRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Authorization", "Bearer expired.or.bad")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("/health = %d", rec.Code)
	}
}

// 未启用 OIDC：providers 返回空列表，start / callback 不存在 —— 对现有部署零影响
func TestRoutes_OIDCDisabled(t *testing.T) {
	h := testRouter(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v2/auth/providers", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("providers = %d %s", rec.Code, rec.Body.String())
	}
	for _, p := range []string{"/api/v2/auth/oidc/start", "/api/v2/auth/oidc/callback"} {
		if got := call(t, h, http.MethodGet, p, 0); got != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", p, got)
		}
	}
}

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
	_, h := testRouterWithSpecs(t)
	return h
}

// testRouterWithSpecs 同 testRouter，另返回 Router 本身，供遍历已注册的路由声明
func testRouterWithSpecs(t *testing.T) (*Router, http.Handler) {
	t.Helper()
	prev := config.GlobalConfig.JWT
	config.GlobalConfig.JWT.SecretKey = "test-secret-not-real"
	t.Cleanup(func() { config.GlobalConfig.JWT = prev })

	svc := &stubRouterSvc{users: map[int64]*database.User{
		1: {ID: 1, Username: "member", Role: middleware.RoleMember, Status: 1},
		2: {ID: 2, Username: "viewer", Role: middleware.RoleViewer, Status: 1},
		3: {ID: 3, Username: "operator", Role: middleware.RoleOperator, Status: 1},
		4: {ID: 4, Username: "admin", Role: middleware.RoleAdmin, Status: 1},
	}}
	r := NewRouter(svc, &database.DB{Audit: stubAudit{}}, stubAI{}, nil, stubGitHub{}, nil)
	return r, r.Handler()
}

// probePath 把注册路径换成能命中它的具体请求路径（子树模式 "/x/" 补一段）
func probePath(path string) string {
	if strings.HasSuffix(path, "/") {
		return path + "probe"
	}
	return path
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
		{"POST", "/api/v2/observe/logs/query"},
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
		{"GET", "/api/v2/aiops/ai/analyze/status?incidentId=x"},
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
		{"PUT", "/api/v2/ai/providers/1", 3, http.StatusForbidden},
	}
	for _, c := range cases {
		if got := call(t, h, c.method, c.path, c.userID); got != c.want {
			t.Errorf("user %d %s %s = %d, want %d", c.userID, c.method, c.path, got, c.want)
		}
	}
}

// 同一路径上，写方法按自己声明的权限把关，不继承读方法的权限。
func TestRoutes_WritePermissionIndependentOfRead(t *testing.T) {
	h := testRouter(t)
	cases := []struct {
		method, path string
		userID       int64
		want         int
	}{
		{"PUT", "/api/v2/slo/targets", 0, http.StatusUnauthorized}, // 访客能看 SLO，不能改目标
		{"PUT", "/api/v2/slo/targets", 2, http.StatusForbidden},    // Viewer 也不能
		{"PUT", "/api/v2/slo/targets", 3, http.StatusBadRequest},   // Operator 过了权限，空请求体被校验拦下
		{"POST", "/api/v2/ai/providers", 3, http.StatusForbidden},  // Operator 能看 AI 配置，不能新建
		{"PUT", "/api/v2/ai/settings", 3, http.StatusForbidden},
		{"PATCH", "/api/v2/ai/settings", 3, http.StatusForbidden},
	}
	for _, c := range cases {
		if got := call(t, h, c.method, c.path, c.userID); got != c.want {
			t.Errorf("user %d %s %s = %d, want %d", c.userID, c.method, c.path, got, c.want)
		}
	}
}

// 路由按（方法，路径）注册：没声明的方法由 ServeMux 直接 405（带 Allow 头），到不了 handler。
// 唯一例外是 ServeMux 的子树补斜杠重定向：/x 没声明该方法、而 /x/ 声明了，就 307 到 /x/，
// 落到的仍是一条已声明（自带权限检查）的路由。用 Admin 请求，排除权限的干扰。
func TestRoutes_UndeclaredMethodRejected(t *testing.T) {
	r, h := testRouterWithSpecs(t)
	if len(r.specs) == 0 {
		t.Fatal("没有登记任何路由声明")
	}
	declared := map[string]map[string]bool{}
	for _, s := range r.specs {
		if declared[s.path] == nil {
			declared[s.path] = map[string]bool{}
		}
		declared[s.path][s.method] = true
	}
	tok, err := middleware.GenerateToken(4, "admin", middleware.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
	for path, allowed := range declared {
		for _, m := range methods {
			if allowed[m] {
				continue
			}
			req := httptest.NewRequest(m, probePath(path), nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			redirect := rec.Code == http.StatusTemporaryRedirect || rec.Code == http.StatusPermanentRedirect
			if redirect && declared[path+"/"][m] && rec.Header().Get("Location") == path+"/" {
				continue
			}
			if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") == "" {
				t.Errorf("%s %s = %d (Allow=%q), want 405 由 ServeMux 拒绝", m, path, rec.Code, rec.Header().Get("Allow"))
			}
		}
	}
}

// 访客能用的写方法只有登录类入口；挂了权限的写方法不得是 0 级权限，访客一律 401。
func TestRoutes_GuestCannotWrite(t *testing.T) {
	r, h := testRouterWithSpecs(t)
	openWrites := map[string]bool{
		"POST /api/v2/user/login": true,
	}
	for _, s := range r.specs {
		if s.method == http.MethodGet {
			continue
		}
		key := s.method + " " + s.path
		if s.perm == "" {
			if !openWrites[key] {
				t.Errorf("无需主体的写方法不在白名单里: %s", key)
			}
			continue
		}
		if middleware.Allowed(middleware.RoleMember, s.perm) {
			t.Errorf("写方法挂了 0 级权限: %s (%s)", key, s.perm)
		}
		if got := call(t, h, s.method, probePath(s.path), 0); got != http.StatusUnauthorized {
			t.Errorf("访客 %s = %d, want 401", key, got)
		}
	}
}

// AI 工具按网关权限名授权（ai 包不引用网关包，只用字符串）：名字必须都在权限表里，改名时这里会失败。
func TestAIToolPermissionsAreGatewayPermissions(t *testing.T) {
	known := map[string]bool{}
	for _, g := range middleware.PermissionTable() {
		known[string(g.Permission)] = true
	}
	for _, p := range ai.ToolPermissionNames() {
		if !known[p] {
			t.Errorf("AI 工具用到的权限 %q 不在网关权限表里", p)
		}
	}
}

// 2 级及以上（凭据与写操作）的非 GET 路由一律审计。
func TestRoutes_PrivilegedNonGetAudited(t *testing.T) {
	r, _ := testRouterWithSpecs(t)
	minLevel := map[middleware.Permission]int{}
	for _, g := range middleware.PermissionTable() {
		minLevel[g.Permission] = g.MinLevel
	}
	for _, s := range r.specs {
		if s.method == http.MethodGet || s.perm == "" {
			continue
		}
		if minLevel[s.perm] >= middleware.RoleOperator && !s.audited {
			t.Errorf("未审计: %s %s (%s)", s.method, s.path, s.perm)
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

// 整份快照不经任何接口对外返回：资源各走自己的列表 / 详情接口，由各接口按权限裁剪字段。
func TestRoutes_NoWholeSnapshotEndpoint(t *testing.T) {
	r, h := testRouterWithSpecs(t)
	for _, userID := range []int64{0, 1, 2, 3, 4} {
		if code := call(t, h, http.MethodGet, "/api/v2/clusters/c1", userID); code != http.StatusNotFound {
			t.Errorf("GET /api/v2/clusters/c1 (user %d) = %d, want 404", userID, code)
		}
	}
	// 集群列表不受影响
	listDeclared := false
	for _, s := range r.specs {
		if s.method == http.MethodGet && s.path == "/api/v2/clusters" {
			listDeclared = true
		}
	}
	if !listDeclared {
		t.Error("GET /api/v2/clusters 未登记")
	}
}

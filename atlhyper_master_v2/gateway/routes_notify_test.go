package gateway

import (
	"net/http"
	"testing"

	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

// 通知设置的全部接口都已登记：统一要求 notify.manage，非 GET 一律审计。
func TestRoutes_NotifyDeclared(t *testing.T) {
	r, _ := testRouterWithSpecs(t)
	want := map[string]bool{ // 方法 路径 → 是否审计
		"GET /api/v2/notify/policy":                false,
		"PUT /api/v2/notify/policy":                true,
		"GET /api/v2/notify/alerts":                false,
		"GET /api/v2/notify/deliveries":            false,
		"GET /api/v2/notify/receivers":             false,
		"POST /api/v2/notify/receivers":            true,
		"PUT /api/v2/notify/receivers/{id}":        true,
		"DELETE /api/v2/notify/receivers/{id}":     true,
		"POST /api/v2/notify/receivers/{id}/test":  true,
		"GET /api/v2/notify/rules":                 false,
		"POST /api/v2/notify/rules":                true,
		"PUT /api/v2/notify/rules/order":           true,
		"POST /api/v2/notify/rules/match":          true,
		"PUT /api/v2/notify/rules/{id}":            true,
		"DELETE /api/v2/notify/rules/{id}":         true,
		"GET /api/v2/notify/silences":              false,
		"POST /api/v2/notify/silences":             true,
		"POST /api/v2/notify/silences/{id}/expire": true,
	}
	found := map[string]bool{}
	for _, s := range r.specs {
		key := s.method + " " + s.path
		audited, expected := want[key]
		if !expected {
			continue
		}
		found[key] = true
		if s.perm != middleware.PermNotifyManage {
			t.Errorf("%s 的权限 = %q, want notify.manage", key, s.perm)
		}
		if s.audited != audited {
			t.Errorf("%s 的审计 = %v, want %v", key, s.audited, audited)
		}
	}
	for key := range want {
		if !found[key] {
			t.Errorf("路由未登记: %s", key)
		}
	}
}

// notify.manage 是 2 级权限：访客 401，成员与只读用户 403。
func TestRoutes_NotifyRequiresOperator(t *testing.T) {
	h := testRouter(t)
	requests := []struct{ method, path string }{
		{"GET", "/api/v2/notify/policy"},
		{"PUT", "/api/v2/notify/policy"},
		{"GET", "/api/v2/notify/alerts"},
		{"GET", "/api/v2/notify/deliveries"},
		{"GET", "/api/v2/notify/receivers"},
		{"POST", "/api/v2/notify/receivers"},
		{"PUT", "/api/v2/notify/receivers/1"},
		{"DELETE", "/api/v2/notify/receivers/1"},
		{"POST", "/api/v2/notify/receivers/1/test"},
		{"GET", "/api/v2/notify/rules"},
		{"POST", "/api/v2/notify/rules"},
		{"PUT", "/api/v2/notify/rules/order"},
		{"POST", "/api/v2/notify/rules/match"},
		{"PUT", "/api/v2/notify/rules/1"},
		{"DELETE", "/api/v2/notify/rules/1"},
		{"GET", "/api/v2/notify/silences"},
		{"POST", "/api/v2/notify/silences"},
		{"POST", "/api/v2/notify/silences/1/expire"},
	}
	for _, c := range requests {
		if got := call(t, h, c.method, c.path, 0); got != http.StatusUnauthorized {
			t.Errorf("访客 %s %s = %d, want 401", c.method, c.path, got)
		}
		for _, userID := range []int64{1, 2} { // 成员、只读用户
			if got := call(t, h, c.method, c.path, userID); got != http.StatusForbidden {
				t.Errorf("用户 %d %s %s = %d, want 403", userID, c.method, c.path, got)
			}
		}
	}
}

// 旧的「每种类型一条」渠道接口已删除。
func TestRoutes_LegacyNotifyChannelsRemoved(t *testing.T) {
	h := testRouter(t)
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/v2/notify/channels"},
		{"GET", "/api/v2/notify/channels/slack"},
		{"PUT", "/api/v2/notify/channels/slack"},
		{"POST", "/api/v2/notify/channels/slack/test"},
	} {
		if got := call(t, h, c.method, c.path, 4); got != http.StatusNotFound {
			t.Errorf("%s %s = %d, want 404", c.method, c.path, got)
		}
	}
}

// /rules/order 与 /rules/match 是固定路径，不会被当成 /rules/{id}：
// 操作员请求它们时走到的是各自的 handler（请求体为空 → 400），而不是按 id 解析
func TestRoutes_NotifyFixedRulePathsWinOverID(t *testing.T) {
	h := testRouter(t)
	if got := call(t, h, http.MethodPut, "/api/v2/notify/rules/order", 3); got != http.StatusBadRequest {
		t.Errorf("PUT /rules/order = %d, want 400（空请求体）", got)
	}
	if got := call(t, h, http.MethodPost, "/api/v2/notify/rules/match", 3); got != http.StatusBadRequest {
		t.Errorf("POST /rules/match = %d, want 400（空请求体）", got)
	}
	// 对 /rules/match 用 DELETE：落到 /rules/{id}，id 不是数字 → 400，而不是删掉什么
	if got := call(t, h, http.MethodDelete, "/api/v2/notify/rules/match", 3); got != http.StatusBadRequest {
		t.Errorf("DELETE /rules/match = %d, want 400（id 无效）", got)
	}
}

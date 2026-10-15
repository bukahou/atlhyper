package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

type stubUserReader struct {
	users map[int64]*database.User
}

func (s *stubUserReader) GetUserByID(_ context.Context, id int64) (*database.User, error) {
	return s.users[id], nil
}

func callAuthMe(t *testing.T, p middleware.Principal, users map[int64]*database.User) AuthMeResponse {
	t.Helper()
	h := NewAuthMeHandler(&stubUserReader{users: users})
	req := httptest.NewRequest(http.MethodGet, "/api/v2/auth/me", nil)
	rec := httptest.NewRecorder()
	h.Get(rec, req.WithContext(middleware.WithPrincipal(req.Context(), p)))
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d", rec.Code)
	}
	var body struct {
		Data AuthMeResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Data
}

// 访客拿到 0 级权限列表，user 为 null —— 前端据此展示，与新成员完全一致。
func TestAuthMe_Guest(t *testing.T) {
	got := callAuthMe(t, middleware.Principal{Level: middleware.RoleMember}, nil)
	if got.User != nil || got.Level != middleware.RoleMember {
		t.Errorf("访客应为 user=null / level 0，得到 %+v", got)
	}
	if !slices.Contains(got.Permissions, string(middleware.PermClusterRead)) ||
		slices.Contains(got.Permissions, string(middleware.PermLogsRead)) {
		t.Errorf("访客权限不对：%v", got.Permissions)
	}
}

// 身份无效（如已禁用）时同样返回访客形态，前端据此清掉本地登录态。
func TestAuthMe_RejectedIdentityLooksLikeGuest(t *testing.T) {
	got := callAuthMe(t, middleware.Principal{Level: middleware.RoleMember, Rejected: "账号已被禁用"}, nil)
	if got.User != nil || got.Level != middleware.RoleMember {
		t.Errorf("得到 %+v", got)
	}
}

func TestAuthMe_LoggedIn(t *testing.T) {
	users := map[int64]*database.User{7: {ID: 7, Username: "alice", DisplayName: "Alice", Role: middleware.RoleViewer, Status: 1}}
	got := callAuthMe(t, middleware.Principal{UserID: 7, Username: "alice", Level: middleware.RoleViewer}, users)
	if got.User == nil || got.User.Username != "alice" || got.User.DisplayName != "Alice" || got.User.Role != middleware.RoleViewer {
		t.Errorf("user 不对：%+v", got.User)
	}
	if got.Level != middleware.RoleViewer || !slices.Contains(got.Permissions, string(middleware.PermLogsRead)) {
		t.Errorf("Viewer 应含 logs.read，得到 %+v", got)
	}
}

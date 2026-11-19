package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

type memUsers struct {
	UserQuery
	UserOps
	users   map[int64]*database.User
	deleted []int64
	updated []*database.User
}

func (m *memUsers) GetUserByID(_ context.Context, id int64) (*database.User, error) {
	u := *m.users[id]
	return &u, nil
}
func (m *memUsers) ListUsers(context.Context) ([]*database.User, error) {
	var out []*database.User
	for _, u := range m.users {
		out = append(out, u)
	}
	return out, nil
}
func (m *memUsers) DeleteUser(_ context.Context, id int64) error {
	m.deleted = append(m.deleted, id)
	return nil
}
func (m *memUsers) UpdateUser(_ context.Context, u *database.User) error {
	m.updated = append(m.updated, u)
	return nil
}

func lastAdminFixture() *memUsers {
	return &memUsers{users: map[int64]*database.User{
		1: {ID: 1, Username: "ops-lead", Role: middleware.RoleAdmin, Status: 1},
		2: {ID: 2, Username: "viewer", Role: middleware.RoleViewer, Status: 1},
		3: {ID: 3, Username: "admin", Role: middleware.RoleAdmin, Status: 0}, // 名为 admin 但已禁用
	}}
}

func post(h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req = req.WithContext(middleware.WithPrincipal(req.Context(), middleware.Principal{UserID: 99, Username: "op", Level: middleware.RoleAdmin}))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// 唯一启用中的管理员：不能删除、禁用、降级（此前只保护写死的用户名 "admin"）
func TestUserHandler_LastActiveAdminProtected(t *testing.T) {
	cases := []struct {
		name string
		call func(h *UserHandler) *httptest.ResponseRecorder
	}{
		{"删除", func(h *UserHandler) *httptest.ResponseRecorder { return post(h.Delete, `{"userId":1}`) }},
		{"禁用", func(h *UserHandler) *httptest.ResponseRecorder {
			return post(h.UpdateStatus, `{"userId":1,"status":0}`)
		}},
		{"降级", func(h *UserHandler) *httptest.ResponseRecorder { return post(h.UpdateRole, `{"userId":1,"role":2}`) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			users := lastAdminFixture()
			rec := c.call(NewUserHandler(users, users))
			if rec.Code != http.StatusConflict || len(users.deleted) != 0 || len(users.updated) != 0 {
				t.Errorf("status = %d deleted=%v updated=%d, want 409 且什么都没改", rec.Code, users.deleted, len(users.updated))
			}
		})
	}
}

// 不是最后一个管理员时照常执行；用户名叫 admin 不再有特殊待遇
func TestUserHandler_NonLastAdminAllowed(t *testing.T) {
	users := lastAdminFixture()
	users.users[3].Status = 1 // 第二个启用中的管理员
	h := NewUserHandler(users, users)
	if rec := post(h.UpdateRole, `{"userId":1,"role":2}`); rec.Code != http.StatusOK {
		t.Errorf("还有别的管理员时降级应成功，status = %d: %s", rec.Code, rec.Body)
	}

	users = lastAdminFixture()
	h = NewUserHandler(users, users)
	if rec := post(h.Delete, `{"userId":3}`); rec.Code != http.StatusOK {
		t.Errorf("已禁用的、名为 admin 的用户可以删除，status = %d: %s", rec.Code, rec.Body)
	}
}

func (m *memUsers) GetUserByUsername(context.Context, string) (*database.User, error) {
	return nil, nil
}
func (m *memUsers) CreateUser(_ context.Context, u *database.User) error {
	m.updated = append(m.updated, u)
	return nil
}

// 建号：不填角色为 Viewer；0（Member）是有效值，⛔ 不当作没填；越界拒绝
func TestUserHandler_RegisterRole(t *testing.T) {
	cases := []struct {
		body     string
		wantCode int
		wantRole int
	}{
		{`{"username":"a","password":"placeholder-not-real"}`, http.StatusOK, middleware.RoleViewer},
		{`{"username":"b","password":"placeholder-not-real","role":0}`, http.StatusOK, middleware.RoleMember},
		{`{"username":"c","password":"placeholder-not-real","role":9}`, http.StatusBadRequest, 0},
	}
	for _, c := range cases {
		users := lastAdminFixture()
		rec := post(NewUserHandler(users, users).Register, c.body)
		if rec.Code != c.wantCode && !(c.wantCode == http.StatusOK && rec.Code == http.StatusCreated) {
			t.Errorf("%s: status = %d, want %d (%s)", c.body, rec.Code, c.wantCode, rec.Body)
			continue
		}
		if c.wantCode == http.StatusOK && (len(users.updated) != 1 || users.updated[0].Role != c.wantRole) {
			t.Errorf("%s: 建出来的角色 = %+v, want %d", c.body, users.updated, c.wantRole)
		}
	}
}

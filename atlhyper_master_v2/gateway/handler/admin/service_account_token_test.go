package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/api/contract"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/model"
)

// 测试用的令牌：明显是占位值，不是真实令牌
const placeholderAPIToken = "atlp_placeholder-not-a-real-token"

const serviceAccountsPath = "/api/v2/admin/service-accounts"

// stubServiceAccountSvc 同时充当读与写两个最小接口的替身
type stubServiceAccountSvc struct {
	views  []model.ServiceAccountView
	opsErr error // 写操作统一返回的错误

	created   *model.CreateServiceAccountRequest
	updatedID int64
	updated   *model.UpdateServiceAccountRequest
	deleted   int64
	issuedFor int64
	issueReq  *model.IssueAPITokenRequest
	issuedBy  string
	revoked   [2]int64
}

func (s *stubServiceAccountSvc) ListServiceAccounts(context.Context) ([]model.ServiceAccountView, error) {
	return s.views, nil
}

func (s *stubServiceAccountSvc) GetServiceAccount(_ context.Context, id int64) (*model.ServiceAccountView, error) {
	for i := range s.views {
		if s.views[i].ID == id {
			return &s.views[i], nil
		}
	}
	return nil, nil
}

func (s *stubServiceAccountSvc) CreateServiceAccount(_ context.Context, req *model.CreateServiceAccountRequest) (int64, error) {
	if s.opsErr != nil {
		return 0, s.opsErr
	}
	s.created = req
	s.views = append(s.views, filledServiceAccountView(41, req.Name))
	return 41, nil
}

func (s *stubServiceAccountSvc) UpdateServiceAccount(_ context.Context, id int64, req *model.UpdateServiceAccountRequest) error {
	if s.opsErr != nil {
		return s.opsErr
	}
	s.updatedID, s.updated = id, req
	return nil
}

func (s *stubServiceAccountSvc) DeleteServiceAccount(_ context.Context, id int64) error {
	if s.opsErr != nil {
		return s.opsErr
	}
	s.deleted = id
	return nil
}

func (s *stubServiceAccountSvc) IssueAPIToken(_ context.Context, accountID int64, req *model.IssueAPITokenRequest, issuedBy string) (*model.APITokenIssued, error) {
	if s.opsErr != nil {
		return nil, s.opsErr
	}
	s.issuedFor, s.issueReq, s.issuedBy = accountID, req, issuedBy
	return &model.APITokenIssued{Token: placeholderAPIToken, TokenPrefix: "atlp_pla", ExpiresAt: "2027-02-24T10:00:00Z"}, nil
}

func (s *stubServiceAccountSvc) RevokeAPIToken(_ context.Context, accountID, tokenID int64) error {
	if s.opsErr != nil {
		return s.opsErr
	}
	s.revoked = [2]int64{accountID, tokenID}
	return nil
}

// filledServiceAccountView 每个字段都有值的一行（契约测试要求规格里的每个属性至少出现一次）
func filledServiceAccountView(id int64, name string) model.ServiceAccountView {
	return model.ServiceAccountView{
		ID: id, Name: name, DisplayName: "Haro", Role: 1, Enabled: true,
		CreatedAt: "2026-11-20T00:00:00Z", LastUsedAt: "2026-11-26T09:30:00Z", CanIssueToken: true,
		Tokens: []model.APITokenView{{
			ID: 2, Name: "prod", TokenPrefix: "atlp_Ab3", Status: model.APITokenStatusRevoked, CreatedBy: "admin",
			CreatedAt: "2026-11-20T00:00:00Z", ExpiresAt: "2027-02-18T00:00:00Z",
			LastUsedAt: "2026-11-26T09:30:00Z", RevokedAt: "2026-11-26T08:00:00Z",
		}},
	}
}

// serviceAccountRequest 构造请求：带上 Admin 主体；pathValues 按 id、tokenId 的顺序填路径参数
func serviceAccountRequest(method, body string, pathValues ...string) *http.Request {
	req := httptest.NewRequest(method, serviceAccountsPath, strings.NewReader(body))
	req = req.WithContext(middleware.WithPrincipal(req.Context(), middleware.Principal{UserID: 1, Username: "ops-admin", Level: middleware.RoleAdmin}))
	for i, name := range []string{"id", "tokenId"} {
		if i < len(pathValues) {
			req.SetPathValue(name, pathValues[i])
		}
	}
	return req
}

func serve(h http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func newServiceAccountHandlerForTest(svc *stubServiceAccountSvc) *ServiceAccountHandler {
	return NewServiceAccountHandler(svc, svc)
}

func TestServiceAccountList_MatchesSpec(t *testing.T) {
	svc := &stubServiceAccountSvc{views: []model.ServiceAccountView{filledServiceAccountView(7, "haro")}}
	rec := serve(newServiceAccountHandlerForTest(svc).List, serviceAccountRequest(http.MethodGet, ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	contract.AssertResponse(t, http.MethodGet, serviceAccountsPath, http.StatusOK, rec.Body.Bytes())

	body := decodeBody(t, rec)
	assertSameJSON(t, string(body["limits"]), `{"assignableRoles":[0,1,2],"maxActiveTokens":10,"maxExpiresInDays":3650}`)
	if string(body["total"]) != "1" {
		t.Errorf("total = %s", body["total"])
	}
}

// 没有服务账号时 data 是空数组而不是 null，业务参数照样下发
func TestServiceAccountList_Empty(t *testing.T) {
	rec := serve(newServiceAccountHandlerForTest(&stubServiceAccountSvc{}).List, serviceAccountRequest(http.MethodGet, ""))
	contract.AssertValid(t, http.MethodGet, serviceAccountsPath, http.StatusOK, rec.Body.Bytes())
	if body := decodeBody(t, rec); string(body["data"]) != "[]" {
		t.Errorf("data = %s, want []", body["data"])
	}
}

func TestServiceAccountCreate(t *testing.T) {
	svc := &stubServiceAccountSvc{}
	rec := serve(newServiceAccountHandlerForTest(svc).Create,
		serviceAccountRequest(http.MethodPost, `{"name":"haro","displayName":"Haro","role":0}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d：%s", rec.Code, rec.Body)
	}
	contract.AssertResponse(t, http.MethodPost, serviceAccountsPath, http.StatusCreated, rec.Body.Bytes())
	if svc.created == nil || svc.created.Name != "haro" || svc.created.Role == nil || *svc.created.Role != 0 {
		t.Errorf("交给 service 的请求不对：%+v", svc.created)
	}
}

func TestServiceAccountCreate_Failures(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		opsErr     error
		wantStatus int
	}{
		{"请求体不是 JSON", `not json`, nil, http.StatusBadRequest},
		{"多出规格里没有的字段", `{"name":"haro","role":0,"kind":"human"}`, nil, http.StatusBadRequest},
		{"输入不合规", `{"name":"haro","role":3}`, fmt.Errorf("%w：等级", model.ErrInvalidServiceAccountInput), http.StatusBadRequest},
		{"账号名已被占用", `{"name":"admin","role":0}`, database.ErrUsernameTaken, http.StatusConflict},
		{"存储故障", `{"name":"haro","role":0}`, errors.New("disk I/O error at /var/lib/master.db"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := serve(newServiceAccountHandlerForTest(&stubServiceAccountSvc{opsErr: c.opsErr}).Create,
				serviceAccountRequest(http.MethodPost, c.body))
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d：%s", rec.Code, c.wantStatus, rec.Body)
			}
			contract.AssertValid(t, http.MethodPost, serviceAccountsPath, c.wantStatus, rec.Body.Bytes())
			if strings.Contains(rec.Body.String(), "master.db") {
				t.Errorf("内部错误的细节不应返回给调用方：%s", rec.Body)
			}
		})
	}
}

func TestServiceAccountUpdate(t *testing.T) {
	const path = serviceAccountsPath + "/{id}"
	cases := []struct {
		name       string
		id         string
		body       string
		opsErr     error
		wantStatus int
	}{
		{"修改成功", "7", `{"displayName":"Haro","role":2,"enabled":false}`, nil, http.StatusOK},
		{"id 不是数字", "abc", `{"displayName":"","role":0,"enabled":true}`, nil, http.StatusBadRequest},
		{"id 不是正数", "0", `{"displayName":"","role":0,"enabled":true}`, nil, http.StatusBadRequest},
		{"请求体不是 JSON", "7", `{`, nil, http.StatusBadRequest},
		{"输入不合规", "7", `{"displayName":"","role":3,"enabled":true}`, model.ErrInvalidServiceAccountInput, http.StatusBadRequest},
		{"不存在", "9", `{"displayName":"","role":0,"enabled":true}`, database.ErrServiceAccountNotFound, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubServiceAccountSvc{opsErr: c.opsErr, views: []model.ServiceAccountView{filledServiceAccountView(7, "haro")}}
			rec := serve(newServiceAccountHandlerForTest(svc).Update, serviceAccountRequest(http.MethodPut, c.body, c.id))
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d：%s", rec.Code, c.wantStatus, rec.Body)
			}
			if c.wantStatus != http.StatusOK {
				contract.AssertValid(t, http.MethodPut, path, c.wantStatus, rec.Body.Bytes())
				return
			}
			contract.AssertResponse(t, http.MethodPut, path, http.StatusOK, rec.Body.Bytes())
			if svc.updatedID != 7 || svc.updated == nil || *svc.updated.Role != 2 || *svc.updated.Enabled {
				t.Errorf("交给 service 的参数不对：%d %+v", svc.updatedID, svc.updated)
			}
		})
	}
}

func TestServiceAccountDelete(t *testing.T) {
	const path = serviceAccountsPath + "/{id}"
	cases := []struct {
		name       string
		id         string
		opsErr     error
		wantStatus int
	}{
		{"删除成功", "7", nil, http.StatusOK},
		{"id 无效", "-1", nil, http.StatusBadRequest},
		{"不存在", "9", database.ErrServiceAccountNotFound, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubServiceAccountSvc{opsErr: c.opsErr}
			rec := serve(newServiceAccountHandlerForTest(svc).Delete, serviceAccountRequest(http.MethodDelete, "", c.id))
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			contract.AssertResponse(t, http.MethodDelete, path, c.wantStatus, rec.Body.Bytes())
			if c.wantStatus == http.StatusOK && svc.deleted != 7 {
				t.Errorf("deleted = %d", svc.deleted)
			}
		})
	}
}

func TestServiceAccountIssueToken(t *testing.T) {
	const path = serviceAccountsPath + "/{id}/tokens"
	svc := &stubServiceAccountSvc{}
	rec := serve(newServiceAccountHandlerForTest(svc).IssueToken,
		serviceAccountRequest(http.MethodPost, `{"name":"prod","expiresInDays":90}`, "7"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d：%s", rec.Code, rec.Body)
	}
	contract.AssertResponse(t, http.MethodPost, path, http.StatusCreated, rec.Body.Bytes())
	if svc.issuedFor != 7 || svc.issueReq == nil || *svc.issueReq.ExpiresInDays != 90 {
		t.Errorf("交给 service 的参数不对：%d %+v", svc.issuedFor, svc.issueReq)
	}
	// 签发人取当前登录的管理员
	if svc.issuedBy != "ops-admin" {
		t.Errorf("issuedBy = %q, want ops-admin", svc.issuedBy)
	}
	// 响应里有令牌明文：不允许任何一层缓存留存它
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestServiceAccountIssueToken_Failures(t *testing.T) {
	const path = serviceAccountsPath + "/{id}/tokens"
	cases := []struct {
		name       string
		id         string
		body       string
		opsErr     error
		wantStatus int
	}{
		{"id 无效", "x", `{"name":"prod"}`, nil, http.StatusBadRequest},
		{"有效期写成字符串", "7", `{"name":"prod","expiresInDays":"90"}`, nil, http.StatusBadRequest},
		{"输入不合规", "7", `{"name":""}`, model.ErrInvalidServiceAccountInput, http.StatusBadRequest},
		{"账号不存在", "9", `{"name":"prod"}`, database.ErrServiceAccountNotFound, http.StatusNotFound},
		{"未吊销的令牌已达上限", "7", `{"name":"prod"}`, database.ErrAPITokenLimitReached, http.StatusConflict},
		{"存储故障", "7", `{"name":"prod"}`, errors.New("db down"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := serve(newServiceAccountHandlerForTest(&stubServiceAccountSvc{opsErr: c.opsErr}).IssueToken,
				serviceAccountRequest(http.MethodPost, c.body, c.id))
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d：%s", rec.Code, c.wantStatus, rec.Body)
			}
			contract.AssertValid(t, http.MethodPost, path, c.wantStatus, rec.Body.Bytes())
			if strings.Contains(rec.Body.String(), "atlp_") {
				t.Errorf("失败时不应返回任何令牌：%s", rec.Body)
			}
		})
	}
}

func TestServiceAccountRevokeToken(t *testing.T) {
	const path = serviceAccountsPath + "/{id}/tokens/{tokenId}"
	cases := []struct {
		name       string
		id         string
		tokenID    string
		opsErr     error
		wantStatus int
	}{
		{"吊销成功", "7", "3", nil, http.StatusOK},
		{"令牌 id 无效", "7", "abc", nil, http.StatusBadRequest},
		{"账号 id 无效", "0", "3", nil, http.StatusBadRequest},
		{"该账号下没有这枚令牌", "7", "99", database.ErrAPITokenNotFound, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubServiceAccountSvc{opsErr: c.opsErr}
			rec := serve(newServiceAccountHandlerForTest(svc).RevokeToken, serviceAccountRequest(http.MethodDelete, "", c.id, c.tokenID))
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			contract.AssertResponse(t, http.MethodDelete, path, c.wantStatus, rec.Body.Bytes())
			if c.wantStatus == http.StatusOK && svc.revoked != [2]int64{7, 3} {
				t.Errorf("revoked = %v", svc.revoked)
			}
		})
	}
}

// 每个处理函数只接受自己的方法（路由层已按方法注册，这里是第二道）
func TestServiceAccount_WrongMethod(t *testing.T) {
	h := newServiceAccountHandlerForTest(&stubServiceAccountSvc{})
	cases := map[string]struct {
		handle  http.HandlerFunc
		request *http.Request
	}{
		"List":        {h.List, serviceAccountRequest(http.MethodPost, "")},
		"Create":      {h.Create, serviceAccountRequest(http.MethodGet, "")},
		"Update":      {h.Update, serviceAccountRequest(http.MethodPost, "", "7")},
		"Delete":      {h.Delete, serviceAccountRequest(http.MethodPost, "", "7")},
		"IssueToken":  {h.IssueToken, serviceAccountRequest(http.MethodGet, "", "7")},
		"RevokeToken": {h.RevokeToken, serviceAccountRequest(http.MethodPost, "", "7", "3")},
	}
	for name, c := range cases {
		if rec := serve(c.handle, c.request); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want 405", name, rec.Code)
		}
	}
}

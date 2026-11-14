package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// 测试用的令牌：明显是占位值，不是真实令牌
const placeholderClusterToken = "atlk_placeholder-not-a-real-token"

// stubClusterRegistrySvc 同时充当读与写两个最小接口的替身
type stubClusterRegistrySvc struct {
	items  []model.ClusterRegistryItem
	opsErr error // 写操作统一返回的错误

	registered     *model.RegisterClusterRequest
	updatedCluster string
	updated        *model.UpdateClusterProfileRequest
	issuedFor      string
	unregistered   string
}

func (s *stubClusterRegistrySvc) ListClusterRegistry(context.Context) ([]model.ClusterRegistryItem, error) {
	return s.items, nil
}

func (s *stubClusterRegistrySvc) GetClusterRegistryItem(_ context.Context, clusterID string) (*model.ClusterRegistryItem, error) {
	for i := range s.items {
		if s.items[i].ClusterID == clusterID {
			return &s.items[i], nil
		}
	}
	return nil, nil
}

func (s *stubClusterRegistrySvc) RegisterCluster(_ context.Context, req *model.RegisterClusterRequest) error {
	if s.opsErr != nil {
		return s.opsErr
	}
	s.registered = req
	s.items = append(s.items, model.ClusterRegistryItem{
		ClusterID: req.ClusterID, Name: req.Name, Description: req.Description,
		Registered: true, Status: model.ClusterStatusNever,
	})
	return nil
}

func (s *stubClusterRegistrySvc) UpdateClusterProfile(_ context.Context, clusterID string, req *model.UpdateClusterProfileRequest) error {
	if s.opsErr != nil {
		return s.opsErr
	}
	s.updatedCluster, s.updated = clusterID, req
	return nil
}

func (s *stubClusterRegistrySvc) IssueClusterToken(_ context.Context, clusterID string) (*model.ClusterTokenIssued, error) {
	if s.opsErr != nil {
		return nil, s.opsErr
	}
	s.issuedFor = clusterID
	return &model.ClusterTokenIssued{
		Token: placeholderClusterToken, TokenPrefix: "atlk_pla", TokenIssuedAt: "2026-11-14T10:00:00Z",
	}, nil
}

func (s *stubClusterRegistrySvc) UnregisterCluster(_ context.Context, clusterID string) error {
	if s.opsErr != nil {
		return s.opsErr
	}
	s.unregistered = clusterID
	return nil
}

func newClusterRegistryHandlerForTest(svc *stubClusterRegistrySvc) *ClusterRegistryHandler {
	return NewClusterRegistryHandler(svc, svc)
}

// clusterRegistryRequest 构造请求；clusterID 非空时填入路径参数（与路由 {clusterId} 对应）
func clusterRegistryRequest(method, body, clusterID string) *http.Request {
	req := httptest.NewRequest(method, "/api/v2/admin/clusters", strings.NewReader(body))
	if clusterID != "" {
		req.SetPathValue("clusterId", clusterID)
	}
	return req
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON：%s", rec.Body.String())
	}
	return body
}

func TestClusterRegistryList(t *testing.T) {
	svc := &stubClusterRegistrySvc{items: []model.ClusterRegistryItem{
		{ClusterID: "c1", Name: "生产", Registered: true, TokenIssued: true, TokenPrefix: "atlk_Ab3",
			TokenIssuedAt: "2026-11-14T10:00:00Z", Status: "online", LastSeen: "2026-11-14T10:05:00Z", Authenticated: true},
	}}
	rec := httptest.NewRecorder()
	newClusterRegistryHandlerForTest(svc).List(rec, clusterRegistryRequest(http.MethodGet, "", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	want := `{"message":"获取成功","data":[{"clusterId":"c1","name":"生产","description":"","registered":true,` +
		`"tokenIssued":true,"tokenPrefix":"atlk_Ab3","tokenIssuedAt":"2026-11-14T10:00:00Z",` +
		`"status":"online","lastSeen":"2026-11-14T10:05:00Z","authenticated":true}],"total":1}`
	assertSameJSON(t, rec.Body.String(), want)
}

// 没有任何集群时 data 是空数组而不是 null（前端直接渲染）
func TestClusterRegistryList_Empty(t *testing.T) {
	rec := httptest.NewRecorder()
	newClusterRegistryHandlerForTest(&stubClusterRegistrySvc{}).List(rec, clusterRegistryRequest(http.MethodGet, "", ""))
	assertSameJSON(t, rec.Body.String(), `{"message":"获取成功","data":[],"total":0}`)
}

func TestClusterRegistryRegister(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		opsErr     error
		wantStatus int
	}{
		{"登记成功", `{"clusterId":"c1","name":"生产","description":""}`, nil, http.StatusCreated},
		{"请求体不是 JSON", `not json`, nil, http.StatusBadRequest},
		{"输入不合规", `{"clusterId":"C1","name":"生产"}`, model.ErrInvalidClusterInput, http.StatusBadRequest},
		{"集群 ID 已登记", `{"clusterId":"c1","name":"生产"}`, database.ErrClusterExists, http.StatusConflict},
		{"存储故障", `{"clusterId":"c1","name":"生产"}`, errors.New("disk I/O error at /var/lib/master.db"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubClusterRegistrySvc{opsErr: c.opsErr}
			rec := httptest.NewRecorder()
			newClusterRegistryHandlerForTest(svc).Register(rec, clusterRegistryRequest(http.MethodPost, c.body, ""))

			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d：%s", rec.Code, c.wantStatus, rec.Body.String())
			}
			if c.wantStatus == http.StatusInternalServerError && strings.Contains(rec.Body.String(), "master.db") {
				t.Errorf("内部错误的细节不应返回给调用方：%s", rec.Body.String())
			}
		})
	}
}

func TestClusterRegistryRegister_ReturnsTheNewRow(t *testing.T) {
	svc := &stubClusterRegistrySvc{}
	rec := httptest.NewRecorder()
	body := `{"clusterId":"c1","name":"生产","description":"东京"}`
	newClusterRegistryHandlerForTest(svc).Register(rec, clusterRegistryRequest(http.MethodPost, body, ""))

	if svc.registered == nil || *svc.registered != (model.RegisterClusterRequest{ClusterID: "c1", Name: "生产", Description: "东京"}) {
		t.Fatalf("交给 service 的请求不对: %+v", svc.registered)
	}
	want := `{"message":"已登记","data":{"clusterId":"c1","name":"生产","description":"东京","registered":true,` +
		`"tokenIssued":false,"tokenPrefix":"","tokenIssuedAt":"","status":"never","lastSeen":"","authenticated":false}}`
	assertSameJSON(t, rec.Body.String(), want)
}

func TestClusterRegistryUpdateProfile(t *testing.T) {
	cases := []struct {
		name       string
		clusterID  string
		body       string
		opsErr     error
		wantStatus int
	}{
		{"修改成功", "c1", `{"name":"生产","description":"东京"}`, nil, http.StatusOK},
		{"请求体不是 JSON", "c1", `{`, nil, http.StatusBadRequest},
		{"输入不合规", "c1", `{"name":""}`, model.ErrInvalidClusterInput, http.StatusBadRequest},
		{"集群未登记", "missing", `{"name":"生产"}`, database.ErrClusterNotFound, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubClusterRegistrySvc{
				opsErr: c.opsErr,
				items:  []model.ClusterRegistryItem{{ClusterID: "c1", Name: "生产", Description: "东京", Registered: true, Status: "never"}},
			}
			rec := httptest.NewRecorder()
			newClusterRegistryHandlerForTest(svc).UpdateProfile(rec, clusterRegistryRequest(http.MethodPut, c.body, c.clusterID))

			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d：%s", rec.Code, c.wantStatus, rec.Body.String())
			}
			if c.wantStatus == http.StatusOK {
				if svc.updatedCluster != "c1" || svc.updated == nil || svc.updated.Description != "东京" {
					t.Errorf("交给 service 的参数不对: %q %+v", svc.updatedCluster, svc.updated)
				}
				if _, ok := decodeBody(t, rec)["data"]; !ok {
					t.Errorf("成功时应返回修改后的那一行：%s", rec.Body.String())
				}
			}
		})
	}
}

func TestClusterRegistryIssueToken(t *testing.T) {
	svc := &stubClusterRegistrySvc{}
	rec := httptest.NewRecorder()
	newClusterRegistryHandlerForTest(svc).IssueToken(rec, clusterRegistryRequest(http.MethodPost, "", "c1"))

	if rec.Code != http.StatusOK || svc.issuedFor != "c1" {
		t.Fatalf("status = %d, issuedFor = %q", rec.Code, svc.issuedFor)
	}
	want := `{"message":"已签发","data":{"token":"` + placeholderClusterToken + `","tokenPrefix":"atlk_pla","tokenIssuedAt":"2026-11-14T10:00:00Z"}}`
	assertSameJSON(t, rec.Body.String(), want)
	// 响应里有令牌明文：不允许任何一层缓存留存它
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestClusterRegistryIssueToken_Failures(t *testing.T) {
	cases := []struct {
		name       string
		opsErr     error
		wantStatus int
	}{
		{"集群未登记", database.ErrClusterNotFound, http.StatusNotFound},
		{"存储故障", errors.New("db down"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			svc := &stubClusterRegistrySvc{opsErr: c.opsErr}
			newClusterRegistryHandlerForTest(svc).IssueToken(rec, clusterRegistryRequest(http.MethodPost, "", "c1"))
			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			if strings.Contains(rec.Body.String(), "atlk_") {
				t.Errorf("失败时不应返回任何令牌：%s", rec.Body.String())
			}
		})
	}
}

func TestClusterRegistryUnregister(t *testing.T) {
	cases := []struct {
		name       string
		opsErr     error
		wantStatus int
	}{
		{"删除成功", nil, http.StatusOK},
		{"集群未登记", database.ErrClusterNotFound, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := &stubClusterRegistrySvc{opsErr: c.opsErr}
			rec := httptest.NewRecorder()
			newClusterRegistryHandlerForTest(svc).Unregister(rec, clusterRegistryRequest(http.MethodDelete, "", "c1"))
			if rec.Code != c.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			if c.opsErr == nil && svc.unregistered != "c1" {
				t.Errorf("unregistered = %q", svc.unregistered)
			}
		})
	}
}

// 每个处理函数只接受自己的方法（路由层已按方法注册，这里是第二道）
func TestClusterRegistry_WrongMethod(t *testing.T) {
	h := newClusterRegistryHandlerForTest(&stubClusterRegistrySvc{})
	cases := []struct {
		name    string
		handle  http.HandlerFunc
		request *http.Request
	}{
		{"List", h.List, clusterRegistryRequest(http.MethodPost, "", "")},
		{"Register", h.Register, clusterRegistryRequest(http.MethodGet, "", "")},
		{"UpdateProfile", h.UpdateProfile, clusterRegistryRequest(http.MethodPost, "", "c1")},
		{"IssueToken", h.IssueToken, clusterRegistryRequest(http.MethodGet, "", "c1")},
		{"Unregister", h.Unregister, clusterRegistryRequest(http.MethodPost, "", "c1")},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		c.handle(rec, c.request)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status = %d, want 405", c.name, rec.Code)
		}
	}
}

// assertSameJSON 按 JSON 语义比较（忽略键顺序与空白）
func assertSameJSON(t *testing.T, got, want string) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("响应不是 JSON：%s", got)
	}
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("期望值不是 JSON：%s", want)
	}
	gotCanonical, _ := json.Marshal(gotValue)
	wantCanonical, _ := json.Marshal(wantValue)
	if string(gotCanonical) != string(wantCanonical) {
		t.Errorf("\n got %s\nwant %s", gotCanonical, wantCanonical)
	}
}

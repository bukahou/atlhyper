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
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/notifier"
)

// 占位凭据：不是任何真实系统的地址或密码
const (
	handlerTestWebhook  = "https://hooks.example.com/services/T000/B000/placeholder-not-real"
	handlerTestPassword = "placeholder-not-a-real-password"
)

// stubNotifyService 同时充当读与写两侧的替身，记录收到的参数
type stubNotifyService struct {
	policy     *model.NotifyPolicy
	alerts     []model.NotifyAlert
	deliveries []model.NotifyDelivery
	receivers  map[int64]*model.NotifyReceiver
	rules      map[int64]*model.NotifyRule
	silences   map[int64]*model.NotifySilence

	alertQuery    model.NotifyAlertQuery
	deliveryQuery model.NotifyDeliveryQuery
	silenceState  string

	receiverRequest *model.NotifyReceiverRequest
	policyRequest   *model.UpdateNotifyPolicyRequest
	reorderedIDs    []int64
	silenceCreator  string
	expiredID       int64
	updatedID       int64
	deletedID       int64

	opsErr error // 写操作统一返回的错误
}

func (s *stubNotifyService) GetNotifyPolicy(context.Context) (*model.NotifyPolicy, error) {
	return s.policy, nil
}
func (s *stubNotifyService) ListNotifyAlerts(_ context.Context, opts model.NotifyAlertQuery) ([]model.NotifyAlert, int, error) {
	s.alertQuery = opts
	return s.alerts, len(s.alerts), nil
}
func (s *stubNotifyService) ListNotifyDeliveries(_ context.Context, opts model.NotifyDeliveryQuery) ([]model.NotifyDelivery, int, error) {
	s.deliveryQuery = opts
	return s.deliveries, len(s.deliveries), nil
}
func (s *stubNotifyService) ListNotifyReceivers(context.Context) ([]model.NotifyReceiver, error) {
	var list []model.NotifyReceiver
	for _, receiver := range s.receivers {
		list = append(list, *receiver)
	}
	return list, nil
}
func (s *stubNotifyService) GetNotifyReceiver(_ context.Context, id int64) (*model.NotifyReceiver, error) {
	return s.receivers[id], nil
}
func (s *stubNotifyService) ListNotifyRules(context.Context) ([]model.NotifyRule, error) {
	var list []model.NotifyRule
	for _, rule := range s.rules {
		list = append(list, *rule)
	}
	return list, nil
}
func (s *stubNotifyService) GetNotifyRule(_ context.Context, id int64) (*model.NotifyRule, error) {
	return s.rules[id], nil
}
func (s *stubNotifyService) ListNotifySilences(_ context.Context, state string) ([]model.NotifySilence, error) {
	s.silenceState = state
	var list []model.NotifySilence
	for _, silence := range s.silences {
		list = append(list, *silence)
	}
	return list, nil
}
func (s *stubNotifyService) GetNotifySilence(_ context.Context, id int64) (*model.NotifySilence, error) {
	return s.silences[id], nil
}
func (s *stubNotifyService) MatchNotifyRule(_ context.Context, req *model.NotifyRuleMatchRequest) (*model.NotifyRuleMatchResult, error) {
	if err := model.ValidateNotifyRuleMatch(req); err != nil {
		return nil, err
	}
	return &model.NotifyRuleMatchResult{Receivers: []model.NotifyReceiverRef{}}, nil
}

func (s *stubNotifyService) UpdateNotifyPolicy(_ context.Context, req *model.UpdateNotifyPolicyRequest) error {
	s.policyRequest = req
	return s.opsErr
}
func (s *stubNotifyService) CreateNotifyReceiver(_ context.Context, req *model.NotifyReceiverRequest) (int64, error) {
	s.receiverRequest = req
	return 1, s.opsErr
}
func (s *stubNotifyService) UpdateNotifyReceiver(_ context.Context, id int64, req *model.NotifyReceiverRequest) error {
	s.updatedID, s.receiverRequest = id, req
	return s.opsErr
}
func (s *stubNotifyService) DeleteNotifyReceiver(_ context.Context, id int64) error {
	s.deletedID = id
	return s.opsErr
}
func (s *stubNotifyService) CreateNotifyRule(context.Context, *model.NotifyRuleRequest) (int64, error) {
	return 1, s.opsErr
}
func (s *stubNotifyService) UpdateNotifyRule(_ context.Context, id int64, _ *model.NotifyRuleRequest) error {
	s.updatedID = id
	return s.opsErr
}
func (s *stubNotifyService) DeleteNotifyRule(_ context.Context, id int64) error {
	s.deletedID = id
	return s.opsErr
}
func (s *stubNotifyService) ReorderNotifyRules(_ context.Context, ids []int64) error {
	s.reorderedIDs = ids
	return s.opsErr
}
func (s *stubNotifyService) CreateNotifySilence(_ context.Context, _ *model.CreateNotifySilenceRequest, createdBy string) (int64, error) {
	s.silenceCreator = createdBy
	return 1, s.opsErr
}
func (s *stubNotifyService) ExpireNotifySilence(_ context.Context, id int64) error {
	s.expiredID = id
	return s.opsErr
}

func newStubNotifyService() *stubNotifyService {
	return &stubNotifyService{
		policy: &model.NotifyPolicy{IncidentAlertsEnabled: true, MinSeverity: "low", DefaultReceiverIDs: []int64{}},
		receivers: map[int64]*model.NotifyReceiver{
			1: {ID: 1, Name: "值班频道", Type: "slack", Enabled: true, Slack: &model.NotifySlackConfig{WebhookConfigured: true}},
		},
		rules:    map[int64]*model.NotifyRule{1: {ID: 1, Name: "生产高严重度", Position: 1, ClusterIDs: []string{}, Sources: []string{}, ReceiverIDs: []int64{}}},
		silences: map[int64]*model.NotifySilence{1: {ID: 1, Comment: "数据库升级", State: "active"}},
	}
}

// fakeReceiverTester 测试发送的替身
type fakeReceiverTester struct {
	err    error
	gotID  int64
	called bool
}

func (f *fakeReceiverTester) TestReceiver(_ context.Context, id int64) error {
	f.called, f.gotID = true, id
	return f.err
}

// notifyCall 直接调用一个 handler 方法；pathID 非空时作为路径参数 {id}
func notifyCall(handle http.HandlerFunc, method, target, pathID, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if pathID != "" {
		req.SetPathValue("id", pathID)
	}
	rec := httptest.NewRecorder()
	handle(rec, req)
	return rec
}

func notifyBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON：%s", rec.Body.String())
	}
	return body
}

// ==================== 列表：空时是 [] 与 0，查询参数原样传到 service ====================

func TestNotifyLists_EmptyIsArrayNotNull(t *testing.T) {
	svc := newStubNotifyService()
	svc.receivers, svc.rules, svc.silences = nil, nil, nil
	h := NewNotifyHandler(svc, svc)

	for name, handle := range map[string]http.HandlerFunc{
		"alerts": h.ListAlerts, "deliveries": h.ListDeliveries, "receivers": h.ListReceivers, "rules": h.ListRules, "silences": h.ListSilences,
	} {
		rec := notifyCall(handle, http.MethodGet, "/api/v2/notify/"+name, "", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s = %d: %s", name, rec.Code, rec.Body.String())
		}
		body := notifyBody(t, rec)
		data, isArray := body["data"].([]any)
		if !isArray || len(data) != 0 {
			t.Errorf("%s：空列表的 data 应是 []，实际 %s", name, rec.Body.String())
		}
		if total, isNumber := body["total"].(float64); !isNumber || total != 0 {
			t.Errorf("%s：total 应是数字 0，实际 %v", name, body["total"])
		}
	}
}

func TestNotifyLists_QueryParams(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)

	notifyCall(h.ListAlerts, http.MethodGet, "/api/v2/notify/alerts?status=firing&cluster_id=prod&limit=20&offset=40", "", "")
	if svc.alertQuery != (model.NotifyAlertQuery{Status: "firing", ClusterID: "prod", Limit: 20, Offset: 40}) {
		t.Errorf("告警查询条件不对: %+v", svc.alertQuery)
	}
	notifyCall(h.ListAlerts, http.MethodGet, "/api/v2/notify/alerts", "", "")
	if svc.alertQuery.Status != "" || svc.alertQuery.Limit != defaultNotifyPageSize || svc.alertQuery.Offset != 0 {
		t.Errorf("不带参数 = 全部、默认分页: %+v", svc.alertQuery)
	}
	notifyCall(h.ListDeliveries, http.MethodGet, "/api/v2/notify/deliveries?fingerprint=incident:inc-1&status=failed&limit=9999", "", "")
	if svc.deliveryQuery.Fingerprint != "incident:inc-1" || svc.deliveryQuery.Status != "failed" || svc.deliveryQuery.Limit != maxNotifyPageSize {
		t.Errorf("送达记录查询条件不对（limit 有上限）: %+v", svc.deliveryQuery)
	}
	notifyCall(h.ListSilences, http.MethodGet, "/api/v2/notify/silences?state=expired", "", "")
	if svc.silenceState != "expired" {
		t.Errorf("静默状态没有传到 service: %q", svc.silenceState)
	}

	for _, bad := range []struct {
		handle http.HandlerFunc
		target string
	}{
		{h.ListAlerts, "/api/v2/notify/alerts?status=unknown"},
		{h.ListAlerts, "/api/v2/notify/alerts?limit=abc"},
		{h.ListAlerts, "/api/v2/notify/alerts?offset=-1"},
		{h.ListDeliveries, "/api/v2/notify/deliveries?status=sent"},
		{h.ListSilences, "/api/v2/notify/silences?state=done"},
	} {
		if rec := notifyCall(bad.handle, http.MethodGet, bad.target, "", ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s = %d，期望 400", bad.target, rec.Code)
		}
	}
}

// ==================== 策略 ====================

func TestNotifyPolicy_GetAndUpdate(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)

	rec := notifyCall(h.GetPolicy, http.MethodGet, "/api/v2/notify/policy", "", "")
	if rec.Code != http.StatusOK || notifyBody(t, rec)["data"].(map[string]any)["minSeverity"] != "low" {
		t.Errorf("读取策略不对: %d %s", rec.Code, rec.Body.String())
	}

	rec = notifyCall(h.UpdatePolicy, http.MethodPut, "/api/v2/notify/policy", "", `{"incidentAlertsEnabled":false,"minSeverity":"high"}`)
	if rec.Code != http.StatusOK || svc.policyRequest == nil || *svc.policyRequest.IncidentAlertsEnabled || *svc.policyRequest.MinSeverity != "high" {
		t.Errorf("保存策略不对: %d %+v", rec.Code, svc.policyRequest)
	}
	if body := notifyBody(t, rec); body["message"] != "已保存" || body["data"] == nil {
		t.Errorf("保存后应带回当前策略: %s", rec.Body.String())
	}

	svc.opsErr = errors.Join(model.ErrInvalidNotifyInput, errors.New("groupWaitSeconds 须在 0–600 之间"))
	if rec := notifyCall(h.UpdatePolicy, http.MethodPut, "/api/v2/notify/policy", "", `{}`); rec.Code != http.StatusBadRequest ||
		!strings.Contains(rec.Body.String(), "groupWaitSeconds") {
		t.Errorf("校验错误应是 400 并带出原因: %d %s", rec.Code, rec.Body.String())
	}
	if rec := notifyCall(h.UpdatePolicy, http.MethodPut, "/api/v2/notify/policy", "", `not json`); rec.Code != http.StatusBadRequest {
		t.Errorf("请求体不是 JSON 应是 400，实际 %d", rec.Code)
	}
}

// ==================== 接收人 ====================

func TestCreateReceiver(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)
	body := `{"name":"值班频道","type":"slack","enabled":true,"slack":{"webhookUrl":"` + handlerTestWebhook + `"}}`

	rec := notifyCall(h.CreateReceiver, http.MethodPost, "/api/v2/notify/receivers", "", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("新建应返回 201，实际 %d %s", rec.Code, rec.Body.String())
	}
	if svc.receiverRequest == nil || svc.receiverRequest.Slack.WebhookURL != handlerTestWebhook {
		t.Errorf("凭据应传到 service: %+v", svc.receiverRequest)
	}
	// 响应里是脱敏后的接收人：只有「是否已配置」
	if strings.Contains(rec.Body.String(), "hooks.example.com") || !strings.Contains(rec.Body.String(), `"webhookConfigured":true`) {
		t.Errorf("响应不应带 Webhook 地址: %s", rec.Body.String())
	}

	cases := []struct {
		err      error
		wantCode int
		wantText string
	}{
		{database.ErrNotifyReceiverNameTaken, http.StatusConflict, "名称"},
		{errors.Join(model.ErrInvalidNotifyInput, errors.New("Webhook 地址必须以 https:// 开头")), http.StatusBadRequest, "https"},
		{database.ErrNotifyEncryptionUnavailable, http.StatusInternalServerError, ""},
		{errors.New("disk full: " + handlerTestWebhook), http.StatusInternalServerError, ""},
	}
	for _, c := range cases {
		svc.opsErr = c.err
		rec := notifyCall(h.CreateReceiver, http.MethodPost, "/api/v2/notify/receivers", "", body)
		if rec.Code != c.wantCode || !strings.Contains(rec.Body.String(), c.wantText) {
			t.Errorf("%v → %d %s，期望 %d 且含 %q", c.err, rec.Code, rec.Body.String(), c.wantCode, c.wantText)
		}
		// 任何错误响应都不回显请求里的凭据，内部错误的细节也不外露
		if strings.Contains(rec.Body.String(), "placeholder-not-real") || strings.Contains(rec.Body.String(), "disk full") {
			t.Errorf("错误响应泄漏了内容: %s", rec.Body.String())
		}
	}
}

func TestUpdateAndDeleteReceiver(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)

	// 启停：其余字段原样带上、凭据传空串
	rec := notifyCall(h.UpdateReceiver, http.MethodPut, "/api/v2/notify/receivers/1", "1",
		`{"name":"值班频道","type":"slack","enabled":false,"slack":{"webhookUrl":""}}`)
	if rec.Code != http.StatusOK || svc.updatedID != 1 || svc.receiverRequest.Enabled == nil || *svc.receiverRequest.Enabled {
		t.Errorf("修改接收人不对: %d %+v", rec.Code, svc.receiverRequest)
	}

	for _, badID := range []string{"abc", "0", "-3", ""} {
		if rec := notifyCall(h.UpdateReceiver, http.MethodPut, "/api/v2/notify/receivers/x", badID, `{}`); rec.Code != http.StatusBadRequest {
			t.Errorf("id=%q 应是 400，实际 %d", badID, rec.Code)
		}
	}

	svc.opsErr = database.ErrNotifyReceiverNotFound
	if rec := notifyCall(h.UpdateReceiver, http.MethodPut, "/api/v2/notify/receivers/9", "9", `{"name":"x","type":"slack"}`); rec.Code != http.StatusNotFound {
		t.Errorf("不存在的接收人应是 404，实际 %d", rec.Code)
	}

	// 删除：仍被引用 → 409，文字里写明被谁引用
	svc.opsErr = &model.NotifyReceiverInUseError{References: []string{"路由规则「生产高严重度」", "默认策略"}}
	rec = notifyCall(h.DeleteReceiver, http.MethodDelete, "/api/v2/notify/receivers/1", "1", "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "生产高严重度") || !strings.Contains(rec.Body.String(), "默认策略") {
		t.Errorf("被引用时应是 409 并写明被谁引用: %d %s", rec.Code, rec.Body.String())
	}
	svc.opsErr = nil
	if rec := notifyCall(h.DeleteReceiver, http.MethodDelete, "/api/v2/notify/receivers/1", "1", ""); rec.Code != http.StatusOK || svc.deletedID != 1 {
		t.Errorf("删除应成功: %d", rec.Code)
	}
}

func TestTestReceiver(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)

	if rec := notifyCall(h.TestReceiver, http.MethodPost, "/api/v2/notify/receivers/1/test", "1", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("通知模块没接上时应是 503，实际 %d", rec.Code)
	}

	cases := []struct {
		name     string
		err      error
		wantCode int
		wantText string
	}{
		{"成功", nil, http.StatusOK, "测试消息已发送"},
		{"接收人不存在", notifier.ErrReceiverNotFound, http.StatusNotFound, "不存在"},
		{"配置不完整", notifier.ErrReceiverConfigIncomplete, http.StatusBadRequest, "不完整"},
		{"发送失败：502 带原因", errors.New("Slack 返回状态码 404: no_service"), http.StatusBadGateway, "no_service"},
	}
	for _, c := range cases {
		tester := &fakeReceiverTester{err: c.err}
		h.SetTester(tester)
		rec := notifyCall(h.TestReceiver, http.MethodPost, "/api/v2/notify/receivers/7/test", "7", "")
		if rec.Code != c.wantCode || !strings.Contains(rec.Body.String(), c.wantText) || !tester.called || tester.gotID != 7 {
			t.Errorf("%s：%d %s（期望 %d 且含 %q）", c.name, rec.Code, rec.Body.String(), c.wantCode, c.wantText)
		}
	}
}

// ==================== 路由规则 ====================

func TestRuleEndpoints(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)

	if rec := notifyCall(h.CreateRule, http.MethodPost, "/api/v2/notify/rules", "", `{"name":"生产高严重度","minSeverity":"high","receiverIds":[]}`); rec.Code != http.StatusCreated {
		t.Errorf("新建规则应返回 201，实际 %d %s", rec.Code, rec.Body.String())
	}
	if rec := notifyCall(h.UpdateRule, http.MethodPut, "/api/v2/notify/rules/1", "1", `{"name":"改名"}`); rec.Code != http.StatusOK || svc.updatedID != 1 {
		t.Errorf("修改规则不对: %d", rec.Code)
	}
	if rec := notifyCall(h.ReorderRules, http.MethodPut, "/api/v2/notify/rules/order", "", `{"ids":[3,1,2]}`); rec.Code != http.StatusOK ||
		len(svc.reorderedIDs) != 3 || svc.reorderedIDs[0] != 3 {
		t.Errorf("重排不对: %d %v", rec.Code, svc.reorderedIDs)
	}
	if rec := notifyCall(h.MatchRule, http.MethodPost, "/api/v2/notify/rules/match", "", `{"clusterId":"prod","source":"incident","severity":"high"}`); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"matchedRule":null`) || !strings.Contains(rec.Body.String(), `"receivers":[]`) {
		t.Errorf("试匹配不对: %d %s", rec.Code, rec.Body.String())
	}
	if rec := notifyCall(h.MatchRule, http.MethodPost, "/api/v2/notify/rules/match", "", `{"clusterId":"prod","source":"incident","severity":"urgent"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("试匹配的严重度不合法应是 400，实际 %d", rec.Code)
	}

	svc.opsErr = database.ErrNotifyRuleOrderMismatch
	if rec := notifyCall(h.ReorderRules, http.MethodPut, "/api/v2/notify/rules/order", "", `{"ids":[1]}`); rec.Code != http.StatusBadRequest {
		t.Errorf("不是全部规则的排列应是 400，实际 %d", rec.Code)
	}
	svc.opsErr = database.ErrNotifyRuleNotFound
	if rec := notifyCall(h.DeleteRule, http.MethodDelete, "/api/v2/notify/rules/9", "9", ""); rec.Code != http.StatusNotFound {
		t.Errorf("不存在的规则应是 404，实际 %d", rec.Code)
	}
}

// ==================== 静默 ====================

func TestSilenceEndpoints(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)

	// 创建人取自当前登录的主体，不信任请求体
	req := httptest.NewRequest(http.MethodPost, "/api/v2/notify/silences",
		strings.NewReader(`{"clusterId":"prod","endsAt":"2026-11-15T12:00:00Z","comment":"数据库升级","createdBy":"someone-else"}`))
	req = req.WithContext(middleware.WithPrincipal(req.Context(), middleware.Principal{UserID: 3, Username: "operator", Level: middleware.RoleOperator}))
	rec := httptest.NewRecorder()
	h.CreateSilence(rec, req)
	if rec.Code != http.StatusCreated || svc.silenceCreator != "operator" {
		t.Errorf("新建静默不对: %d 创建人=%q", rec.Code, svc.silenceCreator)
	}

	if rec := notifyCall(h.ExpireSilence, http.MethodPost, "/api/v2/notify/silences/1/expire", "1", ""); rec.Code != http.StatusOK || svc.expiredID != 1 {
		t.Errorf("提前结束不对: %d", rec.Code)
	}
	svc.opsErr = database.ErrNotifySilenceNotFound
	if rec := notifyCall(h.ExpireSilence, http.MethodPost, "/api/v2/notify/silences/9/expire", "9", ""); rec.Code != http.StatusNotFound {
		t.Errorf("不存在的静默应是 404，实际 %d", rec.Code)
	}
}

// 请求体有大小上限：不让一个请求占住大量内存
func TestNotifyBodyTooLarge(t *testing.T) {
	svc := newStubNotifyService()
	h := NewNotifyHandler(svc, svc)
	huge := `{"name":"` + strings.Repeat("x", maxNotifyBodyBytes) + `"}`
	if rec := notifyCall(h.CreateRule, http.MethodPost, "/api/v2/notify/rules", "", huge); rec.Code != http.StatusBadRequest {
		t.Errorf("超大的请求体应被拒绝，实际 %d", rec.Code)
	}
}

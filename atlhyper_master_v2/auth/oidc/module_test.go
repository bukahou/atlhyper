package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

func testModule(acc *memAccounts) *Module {
	return NewModule(Config{
		Issuer:      testIssuer,
		DisplayName: "Akasha",
		NewUserRole: 0,
		Accounts:    acc,
		IssueToken: func(userID int64, username string, role int) (string, error) {
			return "tok-" + username, nil
		},
	})
}

// fragment 解析 302 目标里 # 之后的参数
func fragment(t *testing.T, location string) url.Values {
	t.Helper()
	if !strings.HasPrefix(location, webCallbackPath+"#") {
		t.Fatalf("应跳到前端回调页并用片段传参，得到 %q", location)
	}
	v, err := url.ParseQuery(strings.TrimPrefix(location, webCallbackPath+"#"))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestOnAuthenticated_IssuesTokenViaFragment(t *testing.T) {
	m := testModule(newMemAccounts())
	req := httptest.NewRequest(http.MethodGet, "/api/v2/auth/oidc/callback", nil)
	to, err := m.onAuthenticated(httptest.NewRecorder(), req, oidcrp.Result{
		Identity: oidcrp.Identity{Subject: "sub-1", PreferredUsername: "alice"},
		Next:     "/observe/logs",
	})
	if err != nil {
		t.Fatal(err)
	}
	f := fragment(t, to)
	if f.Get("token") != "tok-alice" || f.Get("next") != "/observe/logs" {
		t.Errorf("片段不对：%v", f)
	}
}

// next 在这里再校验一次（开放重定向多验一次没有坏处）
func TestOnAuthenticated_UnsafeNextDropped(t *testing.T) {
	m := testModule(newMemAccounts())
	req := httptest.NewRequest(http.MethodGet, "/cb", nil)
	to, _ := m.onAuthenticated(httptest.NewRecorder(), req, oidcrp.Result{
		Identity: oidcrp.Identity{Subject: "sub-1"}, Next: "//evil.example.com",
	})
	if got := fragment(t, to).Get("next"); got != "" {
		t.Errorf("不安全的 next 应丢弃，得到 %q", got)
	}
}

func TestOnAuthenticated_DisabledMapsToReason(t *testing.T) {
	acc := newMemAccounts()
	m := testModule(acc)
	req := httptest.NewRequest(http.MethodGet, "/cb", nil)
	first, _ := m.onAuthenticated(httptest.NewRecorder(), req, oidcrp.Result{Identity: oidcrp.Identity{Subject: "sub-1"}})
	_ = first
	for _, u := range acc.users {
		u.Status = 0
	}
	to, err := m.onAuthenticated(httptest.NewRecorder(), req, oidcrp.Result{Identity: oidcrp.Identity{Subject: "sub-1"}})
	if err != nil {
		t.Fatal(err)
	}
	f := fragment(t, to)
	if f.Get("error") != "disabled" || f.Get("token") != "" {
		t.Errorf("禁用账号应得 error=disabled 且无 token：%v", f)
	}
}

// 原因码按白名单透传，其余一律归为 application，不把任意文案带进页面
func TestOnError_WhitelistsReason(t *testing.T) {
	m := testModule(newMemAccounts())
	for reason, want := range map[string]string{
		oidcrp.ReasonCancelled: "cancelled",
		oidcrp.ReasonState:     "state",
		oidcrp.ReasonUpstream:  "upstream",
		"<script>":             "application",
	} {
		rec := httptest.NewRecorder()
		m.onError(rec, httptest.NewRequest(http.MethodGet, "/cb", nil), reason, "")
		if rec.Code != http.StatusFound {
			t.Fatalf("code = %d", rec.Code)
		}
		if got := fragment(t, rec.Header().Get("Location")).Get("error"); got != want {
			t.Errorf("reason %q → %q, want %q", reason, got, want)
		}
	}
}

func TestProviders(t *testing.T) {
	var nilModule *Module
	for _, c := range []struct {
		m    *Module
		want int
	}{{nilModule, 0}, {testModule(newMemAccounts()), 1}} {
		rec := httptest.NewRecorder()
		c.m.Providers(rec, httptest.NewRequest(http.MethodGet, "/api/v2/auth/providers", nil))
		var body struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Data) != c.want {
			t.Errorf("providers = %+v, want %d 项", body.Data, c.want)
		}
		if c.want == 1 && (body.Data[0].ID != "oidc" || body.Data[0].DisplayName != "Akasha") {
			t.Errorf("provider = %+v", body.Data[0])
		}
	}
}

type fakeFlow struct{ started int }

func (f *fakeFlow) Start(w http.ResponseWriter, _ *http.Request) {
	f.started++
	w.WriteHeader(http.StatusFound)
}
func (f *fakeFlow) Callback(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusFound) }

// issuer 不可达不阻塞启动：首次使用时初始化，失败送回 upstream，过了重试间隔再试
func TestStart_LazyInitWithRetry(t *testing.T) {
	m := testModule(newMemAccounts())
	now := time.Unix(1000, 0)
	m.now = func() time.Time { return now }
	calls := 0
	flow := &fakeFlow{}
	m.newFlow = func(context.Context) (flowHandler, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("issuer down")
		}
		return flow, nil
	}

	start := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		m.Start(rec, httptest.NewRequest(http.MethodGet, "/api/v2/auth/oidc/start", nil))
		return rec
	}

	rec := start()
	if got := fragment(t, rec.Header().Get("Location")).Get("error"); got != "upstream" {
		t.Errorf("初始化失败应送回 upstream，得到 %q", got)
	}
	start() // 重试间隔内：不应再次尝试
	if calls != 1 {
		t.Errorf("重试间隔内不应重复初始化，calls=%d", calls)
	}

	now = now.Add(retryInterval + time.Second)
	start()
	if calls != 2 || flow.started != 1 {
		t.Errorf("过了间隔应重试并成功：calls=%d started=%d", calls, flow.started)
	}
	start()
	if calls != 2 || flow.started != 2 {
		t.Errorf("成功后不再初始化：calls=%d started=%d", calls, flow.started)
	}
}

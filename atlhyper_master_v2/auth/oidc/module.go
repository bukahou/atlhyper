// atlhyper_master_v2/auth/oidc/module.go
// OIDC 登录模块：懒初始化 oidcrp 流程 + 三个 HTTP handler
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"AtlHyper/common/logger"

	"github.com/bukahou/akasha/pkg/oidcrp"
)

var log = logger.Module("OIDC")

const (
	// webCallbackPath 前端回调页：token / 错误码经 URL 片段交付（片段不发往服务器、不进访问日志）
	webCallbackPath = "/auth/callback"
	// retryInterval issuer 不可达时两次初始化尝试的最小间隔
	retryInterval = 15 * time.Second
	// initTimeout 单次初始化（拉 discovery）的超时
	initTimeout = 10 * time.Second
	// cookiePath 状态 cookie 只出现在 OIDC 两个端点上
	cookiePath = "/api/v2/auth/oidc/"
)

// Config 模块配置
type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	CookieSecret string
	CookieSecure bool
	DisplayName  string

	NewUserRole int             // 新账号的角色（成员 = 0）
	Accounts    Accounts        // 必填
	Admission   AdmissionPolicy // 留空 = AdmitAll
	IssueToken  TokenIssuer     // 必填
}

// Module OIDC 登录模块（对外 Facade）
type Module struct {
	cfg      Config
	resolver *accountResolver

	mu      sync.Mutex
	flow    flowHandler
	lastTry time.Time
	newFlow func(ctx context.Context) (flowHandler, error) // 测试可替换
	now     func() time.Time                               // 测试可替换
}

// NewModule 创建模块；不连 issuer（首次使用时才拉 discovery，issuer 不可达不阻塞启动）
func NewModule(cfg Config) *Module {
	if cfg.Admission == nil {
		cfg.Admission = AdmitAll{}
	}
	m := &Module{
		cfg: cfg,
		resolver: &accountResolver{
			accounts:    cfg.Accounts,
			admission:   cfg.Admission,
			issuer:      cfg.Issuer,
			newUserRole: cfg.NewUserRole,
		},
		now: time.Now,
	}
	m.newFlow = m.buildFlow
	return m
}

func (m *Module) buildFlow(ctx context.Context) (flowHandler, error) {
	return oidcrp.New(ctx, oidcrp.Config{
		IssuerURL:       m.cfg.Issuer,
		ClientID:        m.cfg.ClientID,
		ClientSecret:    m.cfg.ClientSecret,
		RedirectURL:     m.cfg.RedirectURL,
		CookieSecret:    m.cfg.CookieSecret,
		CookiePrefix:    "atlhyper_oidc_",
		CookiePath:      cookiePath,
		CookieSecure:    m.cfg.CookieSecure,
		OnAuthenticated: m.onAuthenticated,
		OnError:         m.onError,
	})
}

// ensureFlow 懒初始化；失败后 retryInterval 内不重试（避免每次点击都打 issuer）
func (m *Module) ensureFlow(ctx context.Context) (flowHandler, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flow != nil {
		return m.flow, nil
	}
	if !m.lastTry.IsZero() && m.now().Sub(m.lastTry) < retryInterval {
		return nil, errors.New("issuer 暂不可用（重试间隔内）")
	}
	m.lastTry = m.now()

	ctx, cancel := context.WithTimeout(ctx, initTimeout)
	defer cancel()
	flow, err := m.newFlow(ctx)
	if err != nil {
		log.Error("OIDC 初始化失败", "issuer", m.cfg.Issuer, "err", err)
		return nil, err
	}
	log.Info("OIDC 初始化完成", "issuer", m.cfg.Issuer)
	m.flow = flow
	return flow, nil
}

// Providers 登录框据此决定是否显示外部登录按钮（未启用时 Module 为 nil，返回空列表）
// GET /api/v2/auth/providers
func (m *Module) Providers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	type provider struct {
		ID          string `json:"id"`
		DisplayName string `json:"displayName"`
	}
	providers := []provider{}
	if m != nil {
		providers = append(providers, provider{ID: "oidc", DisplayName: m.cfg.DisplayName})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"message": "获取成功", "data": providers})
}

// Start 发起登录
// GET /api/v2/auth/oidc/start?next=/path
func (m *Module) Start(w http.ResponseWriter, r *http.Request) {
	flow, err := m.ensureFlow(r.Context())
	if err != nil {
		redirectWithFragment(w, r, url.Values{"error": {oidcrp.ReasonUpstream}})
		return
	}
	flow.Start(w, r)
}

// Callback issuer 回跳
// GET /api/v2/auth/oidc/callback
func (m *Module) Callback(w http.ResponseWriter, r *http.Request) {
	flow, err := m.ensureFlow(r.Context())
	if err != nil {
		redirectWithFragment(w, r, url.Values{"error": {oidcrp.ReasonUpstream}})
		return
	}
	flow.Callback(w, r)
}

// onAuthenticated 身份已验签：查或建本地账号，签发 token，经片段交给前端
// 账号层面的拒绝（禁用 / 准入）用跳转表达而非返回 error，以便前端显示准确原因
func (m *Module) onAuthenticated(_ http.ResponseWriter, r *http.Request, res oidcrp.Result) (string, error) {
	user, err := m.resolver.resolve(r.Context(), res.Identity)
	switch {
	case errors.Is(err, errDisabled):
		return fragmentURL(url.Values{"error": {"disabled"}}), nil
	case errors.Is(err, errRejected):
		return fragmentURL(url.Values{"error": {"rejected"}}), nil
	case err != nil:
		return "", err // oidcrp 记日志并走 onError(application)，不向用户泄漏细节
	}

	token, err := m.cfg.IssueToken(user.ID, user.Username, user.Role)
	if err != nil {
		return "", err
	}
	if err := m.cfg.Accounts.UpdateUserLastLogin(r.Context(), user.ID, clientIP(r)); err != nil {
		log.Warn("更新最后登录时间失败", "user_id", user.ID, "err", err)
	}
	log.Info("OIDC 登录成功", "user_id", user.ID, "username", user.Username, "role", user.Role)

	v := url.Values{"token": {token}}
	if oidcrp.DefaultSafeNext(res.Next) { // oidcrp 在 Start 时已验过，这里再验一次
		v.Set("next", res.Next)
	}
	return fragmentURL(v), nil
}

// knownReasons 可透传给前端的原因码；其余一律 application（不把任意文案带进页面）
var knownReasons = map[string]bool{
	oidcrp.ReasonCancelled: true,
	oidcrp.ReasonState:     true,
	oidcrp.ReasonUpstream:  true,
}

func (m *Module) onError(w http.ResponseWriter, r *http.Request, reason, _ string) {
	if !knownReasons[reason] {
		reason = oidcrp.ReasonApplication
	}
	redirectWithFragment(w, r, url.Values{"error": {reason}})
}

func fragmentURL(v url.Values) string {
	return webCallbackPath + "#" + v.Encode()
}

func redirectWithFragment(w http.ResponseWriter, r *http.Request, v url.Values) {
	http.Redirect(w, r, fragmentURL(v), http.StatusFound)
}

// clientIP 取 X-Forwarded-For 第一段（与审计中间件一致）
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	return r.RemoteAddr
}

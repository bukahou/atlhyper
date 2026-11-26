package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/agent"
)

// 测试用占位令牌（⛔ 不是真实令牌）
const (
	tokenLevel0 = "atlp_placeholder-level0-not-real"
	tokenLevel1 = "atlp_placeholder-level1-not-real"
)

type fakeAuthenticator struct {
	callers map[string]Caller
	err     error // 非 nil：模拟暂时无法校验
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, plaintext string) (Caller, error) {
	if f.err != nil {
		return Caller{}, f.err
	}
	caller, ok := f.callers[plaintext]
	if !ok {
		return Caller{}, ErrInvalidToken
	}
	return caller, nil
}

type fakeClusters struct {
	ids []string
	err error
}

func (f *fakeClusters) ListClusters(context.Context) ([]agent.ClusterInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	clusters := make([]agent.ClusterInfo, 0, len(f.ids))
	for _, id := range f.ids {
		clusters = append(clusters, agent.ClusterInfo{ClusterID: id, Status: "online"})
	}
	return clusters, nil
}

type auditRecorder struct {
	mu      sync.Mutex
	entries []*database.AuditLog
}

func (a *auditRecorder) Create(_ context.Context, entry *database.AuditLog) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return nil
}

func (a *auditRecorder) all() []*database.AuditLog {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.entries)
}

type echoParams struct {
	Query string `json:"query,omitempty" jsonschema:"查询词"`
}

// 测试用的权限：p0 给 0 级，p1 给 1 级
func testAllowed(level int, permission string) bool {
	need := map[string]int{"p0": 0, "p1": 1}
	min, ok := need[permission]
	return ok && level >= min
}

type harness struct {
	url      string
	clusters *fakeClusters
	audit    *auditRecorder
	auth     *fakeAuthenticator
	seen     *[]string // 工具处理函数收到的「集群|参数」
}

func newHarness(t *testing.T, mutate func(*Config)) *harness {
	t.Helper()
	var seen []string
	echo := func(name, permission string, noCluster bool) *toolkit.Tool {
		return toolkit.MustDefine(toolkit.Spec[echoParams]{
			Name: name, Description: name + " 的说明", Permission: permission, NoCluster: noCluster,
			Handler: func(_ context.Context, clusterID string, p echoParams) (string, error) {
				seen = append(seen, clusterID+"|"+p.Query)
				return `{"tool":"` + name + `","cluster":"` + clusterID + `"}`, nil
			},
		})
	}
	reg := toolkit.NewRegistry()
	reg.Register(
		echo("status", "p0", false),
		echo("detail", "p1", false),
		echo("clusters", "p0", true),
		toolkit.MustDefine(toolkit.Spec[echoParams]{
			Name: "big", Description: "很长的结果", Permission: "p0",
			Handler: func(context.Context, string, echoParams) (string, error) {
				return strings.Repeat("长", 500), nil
			},
		}),
		toolkit.MustDefine(toolkit.Spec[echoParams]{
			Name: "boom", Description: "总是失败", Permission: "p0",
			Handler: func(context.Context, string, echoParams) (string, error) {
				return "", errors.New("底层查询失败")
			},
		}),
		toolkit.MustDefine(toolkit.Spec[echoParams]{
			Name: "hidden", Description: "没列进开放名单", Permission: "p0",
			Handler: func(context.Context, string, echoParams) (string, error) { return "{}", nil },
		}),
	)

	h := &harness{
		clusters: &fakeClusters{ids: []string{"c1"}},
		audit:    &auditRecorder{},
		auth: &fakeAuthenticator{callers: map[string]Caller{
			tokenLevel0: {UserID: 7, Username: "svc-zero", Level: 0, TokenName: "t0"},
			tokenLevel1: {UserID: 8, Username: "svc-one", Level: 1, TokenName: "t1"},
		}},
		seen: &seen,
	}
	cfg := Config{
		Version:        "test",
		Tools:          reg,
		Authenticator:  h.auth,
		Clusters:       h.clusters,
		Audit:          h.audit,
		Allowed:        testAllowed,
		Levels:         []int{0, 1},
		MaxResultRunes: 100,
	}
	if mutate != nil {
		mutate(&cfg)
	}
	server, err := newServer(cfg, []string{"status", "detail", "clusters", "big", "boom"})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(server.Handler())
	t.Cleanup(ts.Close)
	h.url = ts.URL + "/mcp"
	return h
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if b.token != "" {
		r.Header.Set("Authorization", "Bearer "+b.token)
	}
	return http.DefaultTransport.RoundTrip(r)
}

func (h *harness) connect(t *testing.T, token string) (*mcp.ClientSession, error) {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             h.url,
		HTTPClient:           &http.Client{Transport: bearerTransport{token: token}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err == nil {
		t.Cleanup(func() { session.Close() })
	}
	return session, err
}

func (h *harness) mustConnect(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	session, err := h.connect(t, token)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	return session
}

func toolNames(t *testing.T, session *mcp.ClientSession) []string {
	t.Helper()
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("调用 %s 出现协议错误: %v", name, err)
	}
	return result
}

func resultText(result *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			sb.WriteString(text.Text)
		}
	}
	return sb.String()
}

// 没有令牌、令牌无效都连不上；有效令牌连得上
func TestAuth_RequiresValidToken(t *testing.T) {
	h := newHarness(t, nil)
	if _, err := h.connect(t, ""); err == nil {
		t.Error("不带令牌不应连上")
	}
	if _, err := h.connect(t, "atlp_placeholder-unknown-not-real"); err == nil {
		t.Error("无效令牌不应连上")
	}
	h.mustConnect(t, tokenLevel0)
}

// 暂时无法校验（查库失败）是 5xx，不当成令牌无效；响应里不带内部错误原文
func TestAuth_VerifierFailureIsServerError(t *testing.T) {
	h := newHarness(t, nil)
	h.auth.err = errors.New("database is locked: secret-internal-detail")
	req, _ := http.NewRequest(http.MethodPost, h.url, strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+tokenLevel0)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("状态码 = %d, want 500", resp.StatusCode)
	}
	if strings.Contains(string(body), "secret-internal-detail") {
		t.Errorf("响应里出现了内部错误原文: %s", body)
	}
}

// 工具清单 = 开放名单 ∩ 已注册 ∩ 调用者的等级看得到；每个工具带只读标注，针对集群的工具多一个 cluster_id 参数
func TestListTools_FilteredByLevelAndAnnotated(t *testing.T) {
	h := newHarness(t, nil)

	zero := h.mustConnect(t, tokenLevel0)
	if got, want := toolNames(t, zero), []string{"big", "boom", "clusters", "status"}; !slices.Equal(got, want) {
		t.Errorf("0 级看到 %v, want %v", got, want)
	}
	one := h.mustConnect(t, tokenLevel1)
	if got, want := toolNames(t, one), []string{"big", "boom", "clusters", "detail", "status"}; !slices.Equal(got, want) {
		t.Errorf("1 级看到 %v, want %v（hidden 不在开放名单里）", got, want)
	}

	listed, err := zero.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s 应标为只读: %+v", tool.Name, tool.Annotations)
		}
		raw, _ := json.Marshal(tool.InputSchema)
		var schema struct {
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		}
		if err := json.Unmarshal(raw, &schema); err != nil {
			t.Fatal(err)
		}
		_, hasCluster := schema.Properties[clusterArgument]
		if wantCluster := tool.Name != "clusters"; hasCluster != wantCluster {
			t.Errorf("%s 的 cluster_id 参数: 有=%v, want %v", tool.Name, hasCluster, wantCluster)
		}
		if slices.Contains(schema.Required, clusterArgument) {
			t.Errorf("%s 的 cluster_id 不应必填", tool.Name)
		}
		if _, ok := schema.Properties["query"]; !ok {
			t.Errorf("%s 丢了工具自己的参数: %s", tool.Name, raw)
		}
	}
}

// 集群的确定：只有一个集群时不用给；给了就用给的（不传进工具自己的参数）；多个集群又没给时报错并列出可选项
func TestCallTool_ClusterResolution(t *testing.T) {
	h := newHarness(t, nil)
	session := h.mustConnect(t, tokenLevel0)

	result := callTool(t, session, "status", map[string]any{"query": "q1"})
	if result.IsError || !strings.Contains(resultText(result), `"cluster":"c1"`) {
		t.Fatalf("单集群默认: isError=%v text=%s", result.IsError, resultText(result))
	}

	h.clusters.ids = []string{"c1", "c2"}
	result = callTool(t, session, "status", map[string]any{"query": "q2", clusterArgument: "c2"})
	if result.IsError || !strings.Contains(resultText(result), `"cluster":"c2"`) {
		t.Fatalf("指定集群: isError=%v text=%s", result.IsError, resultText(result))
	}

	result = callTool(t, session, "status", map[string]any{"query": "q3"})
	if !result.IsError || result.Meta[MetaErrorCode] != CodeClusterRequired {
		t.Fatalf("多集群没给应报 %s: isError=%v meta=%v", CodeClusterRequired, result.IsError, result.Meta)
	}
	if text := resultText(result); !strings.Contains(text, "c1") || !strings.Contains(text, "c2") {
		t.Errorf("错误说明应列出可选的集群: %s", text)
	}

	result = callTool(t, session, "status", map[string]any{clusterArgument: "nope"})
	if !result.IsError || result.Meta[MetaErrorCode] != CodeClusterNotFound {
		t.Fatalf("不存在的集群应报 %s: meta=%v", CodeClusterNotFound, result.Meta)
	}

	// 不针对集群的工具：多集群时照样能调，也不认 cluster_id 以外的处理
	result = callTool(t, session, "clusters", map[string]any{})
	if result.IsError {
		t.Fatalf("clusters 应成功: %s", resultText(result))
	}

	if got, want := *h.seen, []string{"c1|q1", "c2|q2", "|"}; !slices.Equal(got, want) {
		t.Errorf("处理函数收到 %v, want %v", got, want)
	}

	// 查集群失败 ≠ 没有集群
	h.clusters.err = errors.New("store down")
	result = callTool(t, session, "status", map[string]any{})
	if !result.IsError || result.Meta[MetaErrorCode] != CodeUnavailable {
		t.Errorf("列不出集群应报 %s: meta=%v", CodeUnavailable, result.Meta)
	}
}

// 调用时再查一遍权限与参数：清单里没有的工具调不到，参数不合法有稳定的错误码
func TestCallTool_PermissionAndParams(t *testing.T) {
	h := newHarness(t, nil)
	zero := h.mustConnect(t, tokenLevel0)

	// 0 级的清单里没有 detail：直接按名字调用是协议层的「没有这个工具」
	if _, err := zero.CallTool(context.Background(), &mcp.CallToolParams{Name: "detail", Arguments: map[string]any{}}); err == nil {
		t.Error("0 级不应调到 detail")
	}
	// 开放名单之外的工具对谁都不存在
	one := h.mustConnect(t, tokenLevel1)
	if _, err := one.CallTool(context.Background(), &mcp.CallToolParams{Name: "hidden", Arguments: map[string]any{}}); err == nil {
		t.Error("不在开放名单里的工具不应调到")
	}

	result := callTool(t, zero, "status", map[string]any{"unknown_field": 1})
	if !result.IsError || result.Meta[MetaErrorCode] != CodeInvalidParams {
		t.Errorf("未知参数应报 %s: meta=%v text=%s", CodeInvalidParams, result.Meta, resultText(result))
	}
	result = callTool(t, zero, "status", map[string]any{clusterArgument: 42})
	if !result.IsError || result.Meta[MetaErrorCode] != CodeInvalidParams {
		t.Errorf("cluster_id 不是字符串应报 %s: meta=%v", CodeInvalidParams, result.Meta)
	}
}

// 工具失败是 isError + 错误码；结果超长时截断并标记
func TestCallTool_FailureAndTruncation(t *testing.T) {
	h := newHarness(t, nil)
	session := h.mustConnect(t, tokenLevel0)

	result := callTool(t, session, "boom", map[string]any{})
	if !result.IsError || result.Meta[MetaErrorCode] != CodeFailed {
		t.Errorf("工具失败应报 %s: isError=%v meta=%v", CodeFailed, result.IsError, result.Meta)
	}
	if !strings.Contains(resultText(result), "底层查询失败") {
		t.Errorf("失败说明应带工具给的原因: %s", resultText(result))
	}

	result = callTool(t, session, "big", map[string]any{})
	if result.IsError {
		t.Fatal("big 应成功")
	}
	if got := len([]rune(resultText(result))); got > 100+20 {
		t.Errorf("结果应截到 100 个字符左右，实际 %d", got)
	}
	if result.Meta[MetaTruncated] != true {
		t.Errorf("截断应在元数据里标出: %v", result.Meta)
	}
	short := callTool(t, session, "status", map[string]any{})
	if _, marked := short.Meta[MetaTruncated]; marked {
		t.Errorf("没截断的结果不应带截断标记: %v", short.Meta)
	}
}

// 每次工具调用记一行审计：谁、哪个工具、哪个集群、成败；令牌不进审计
func TestCallTool_Audited(t *testing.T) {
	h := newHarness(t, nil)
	session := h.mustConnect(t, tokenLevel0)
	callTool(t, session, "status", map[string]any{"query": "q"})
	callTool(t, session, "boom", map[string]any{})

	entries := h.audit.all()
	if len(entries) != 2 {
		t.Fatalf("审计 %d 条, want 2", len(entries))
	}
	ok, failed := entries[0], entries[1]
	if ok.UserID != 7 || ok.Username != "svc-zero" || ok.Role != 0 || ok.Source != auditSource ||
		ok.Action != auditAction || ok.Resource != auditResource || !ok.Success || !strings.HasSuffix(ok.Path, "/status") {
		t.Errorf("成功调用的审计 = %+v", ok)
	}
	if !strings.Contains(ok.RequestBody, `"query":"q"`) || !strings.Contains(ok.RequestBody, "c1") {
		t.Errorf("审计应带参数与集群: %s", ok.RequestBody)
	}
	if failed.Success || failed.ErrorMessage != CodeFailed {
		t.Errorf("失败调用的审计 = %+v", failed)
	}
	for _, entry := range entries {
		if strings.Contains(entry.RequestBody, "atlp_") || strings.Contains(entry.UserAgent, "atlp_") {
			t.Errorf("审计里出现了令牌: %+v", entry)
		}
	}
}

// 按令牌限流：超出后返回 429
func TestRateLimit_PerToken(t *testing.T) {
	h := newHarness(t, func(cfg *Config) { cfg.RateLimitPerMinute = 3 })
	codes := map[int]int{}
	for i := 0; i < 6; i++ {
		req, _ := http.NewRequest(http.MethodPost, h.url, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
		req.Header.Set("Authorization", "Bearer "+tokenLevel0)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		codes[resp.StatusCode]++
	}
	if codes[http.StatusTooManyRequests] != 3 {
		t.Errorf("6 次请求、每分钟 3 次：应有 3 次 429，实际 %v", codes)
	}
}

// 浏览器跨站请求（带别处的 Origin）被拒；请求体超过上限被拒
func TestEntry_RejectsCrossOriginAndOversizedBody(t *testing.T) {
	h := newHarness(t, func(cfg *Config) { cfg.MaxBodyBytes = 256 })

	req, _ := http.NewRequest(http.MethodPost, h.url, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	req.Header.Set("Authorization", "Bearer "+tokenLevel0)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Origin", "https://evil.example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("带 Origin 的请求状态码 = %d, want 403", resp.StatusCode)
	}

	big := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"status","arguments":{"query":"` + strings.Repeat("x", 2048) + `"}}}`
	req, _ = http.NewRequest(http.MethodPost, h.url, strings.NewReader(big))
	req.Header.Set("Authorization", "Bearer "+tokenLevel0)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode < 400 {
		t.Errorf("超长请求体状态码 = %d，应被拒绝", resp.StatusCode)
	}
	if len(*h.seen) != 0 {
		t.Errorf("超长请求不应执行到工具: %v", *h.seen)
	}
}

// 配置有误在启动时就失败：开放名单里的工具没注册、不是只读、参数里自带 cluster_id
func TestNewServer_RejectsBadExposure(t *testing.T) {
	base := func() Config {
		return Config{
			Tools: toolkit.NewRegistry(), Authenticator: &fakeAuthenticator{}, Clusters: &fakeClusters{},
			Audit: &auditRecorder{}, Allowed: testAllowed, Levels: []int{0},
		}
	}
	handler := func(context.Context, string, echoParams) (string, error) { return "{}", nil }

	cfg := base()
	if _, err := newServer(cfg, []string{"missing"}); err == nil {
		t.Error("开放名单里有没注册的工具应失败")
	}

	cfg = base()
	cfg.Tools.Register(toolkit.MustDefine(toolkit.Spec[echoParams]{Name: "writer", Description: "写", Permission: "p0", Effect: toolkit.EffectWrite, Handler: handler}))
	if _, err := newServer(cfg, []string{"writer"}); err == nil {
		t.Error("开放非只读工具应失败")
	}

	type clashParams struct {
		ClusterID string `json:"cluster_id,omitempty"`
	}
	cfg = base()
	cfg.Tools.Register(toolkit.MustDefine(toolkit.Spec[clashParams]{Name: "clash", Description: "撞名", Permission: "p0",
		Handler: func(context.Context, string, clashParams) (string, error) { return "{}", nil }}))
	if _, err := newServer(cfg, []string{"clash"}); err == nil {
		t.Error("工具自己的参数里已有 cluster_id 应失败")
	}

	cfg = base()
	cfg.Authenticator = nil
	if _, err := newServer(cfg, nil); err == nil {
		t.Error("没有令牌校验器应失败")
	}
}

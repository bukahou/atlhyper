package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/service"
)

// stubProviderSvc 模拟 repo 的行为：保存时把 key 原地加密
type stubProviderSvc struct {
	service.Service
	saved *database.AIProvider
}

func (s *stubProviderSvc) CreateAIProvider(_ context.Context, p *database.AIProvider) error {
	p.ID = 1
	p.APIKey = "ENC:" + strings.Repeat("x", 40) // repo 原地加密
	s.saved = p
	return nil
}

func (s *stubProviderSvc) GetAIProviderByID(_ context.Context, _ int64) (*database.AIProvider, error) {
	cp := *s.saved
	return &cp, nil
}

func (s *stubProviderSvc) UpdateAIProvider(_ context.Context, p *database.AIProvider) error {
	p.APIKey = "ENC:" + strings.Repeat("y", 40)
	s.saved = p
	return nil
}

func postJSON(t *testing.T, h http.HandlerFunc, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestCreateProvider_BedrockSupportedAndKeyRequired(t *testing.T) {
	h := NewAIProviderHandler(&stubProviderSvc{})

	rec := postJSON(t, h.ProvidersHandler, http.MethodPost, "/api/v2/ai/providers",
		`{"name":"tokyo","provider":"bedrock","model":"qwen.qwen3-235b-a22b-2507-v1:0"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("bedrock 缺 key 应 400，得到 %d %s", rec.Code, rec.Body.String())
	}

	rec = postJSON(t, h.ProvidersHandler, http.MethodPost, "/api/v2/ai/providers",
		`{"name":"tokyo","provider":"bedrock","apiKey":"ABSKexamplekey1234","model":"qwen.qwen3-235b-a22b-2507-v1:0"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bedrock 应可创建，得到 %d %s", rec.Code, rec.Body.String())
	}
}

// 响应里的掩码必须基于明文 key，而不是 repo 加密后的密文
func TestCreateAndUpdateProvider_MaskUsesPlaintext(t *testing.T) {
	svc := &stubProviderSvc{}
	h := NewAIProviderHandler(svc)

	rec := postJSON(t, h.ProvidersHandler, http.MethodPost, "/api/v2/ai/providers",
		`{"name":"p","provider":"openai","apiKey":"sk-abcd1234efgh5678","model":"gpt-4o"}`)
	var created ProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.APIKeyMasked != "sk-a****5678" {
		t.Errorf("create 掩码 = %q，want sk-a****5678", created.APIKeyMasked)
	}

	rec = postJSON(t, h.ProviderHandler, http.MethodPut, "/api/v2/ai/providers/1",
		`{"apiKey":"sk-zzzz0000yyyy9999"}`)
	var updated ProviderResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.APIKeyMasked != "sk-z****9999" {
		t.Errorf("update 掩码 = %q，want sk-z****9999", updated.APIKeyMasked)
	}
}

func TestUpdateProvider_RejectsUnsupportedType(t *testing.T) {
	svc := &stubProviderSvc{saved: &database.AIProvider{ID: 1, Provider: "openai", APIKey: "ENC:k", Model: "gpt-4o"}}
	h := NewAIProviderHandler(svc)
	rec := postJSON(t, h.ProviderHandler, http.MethodPut, "/api/v2/ai/providers/1", `{"provider":"nope"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("不支持的 provider 类型应 400，得到 %d", rec.Code)
	}
}

// 保存前清理粘贴带入的杂质：首尾空白（所有 provider）、bedrock 的 Bearer 前缀；
// bedrock 的 key 含引号直接拒绝（引号可能来自 `terraform output` 未加 -raw，替用户删风险更大）
func TestNormalizeAPIKey(t *testing.T) {
	cases := []struct {
		name, provider, raw, want string
		wantErr                   bool
	}{
		{"首尾空白与换行", "openai", "  sk-abc123\n", "sk-abc123", false},
		{"bedrock 去 Bearer 前缀", "bedrock", "Bearer ABSKexample123", "ABSKexample123", false},
		{"bedrock 去 bearer 前缀（大小写不敏感）+ 空白", "bedrock", " bearer  ABSKexample123 ", "ABSKexample123", false},
		{"bedrock 含引号 → 拒绝", "bedrock", `"ABSKexample123"`, "", true},
		{"非 bedrock 不动 Bearer 字样", "openai", "Bearer-like-key", "Bearer-like-key", false},
		{"空值保持空（由必填校验处理）", "bedrock", "   ", "", false},
	}
	for _, c := range cases {
		got, err := normalizeAPIKey(c.provider, c.raw)
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("%s: got %q err=%v, want %q wantErr=%v", c.name, got, err, c.want, c.wantErr)
		}
	}
}

func TestCreateProvider_NormalizesKeyBeforeSave(t *testing.T) {
	svc := &stubProviderSvc{}
	h := NewAIProviderHandler(svc)
	rec := postJSON(t, h.ProvidersHandler, http.MethodPost, "/api/v2/ai/providers",
		`{"name":"b","provider":"bedrock","apiKey":"  Bearer ABSKexamplekey1234\n","model":"m"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d %s", rec.Code, rec.Body.String())
	}
	var resp ProviderResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.APIKeyMasked != "ABSK****1234" {
		t.Errorf("应保存清理后的 key，掩码 = %q", resp.APIKeyMasked)
	}

	rec = postJSON(t, h.ProvidersHandler, http.MethodPost, "/api/v2/ai/providers",
		`{"name":"b","provider":"bedrock","apiKey":"\"ABSKexamplekey1234\"","model":"m"}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "引号") {
		t.Errorf("含引号应 400 并提示，得到 %d %s", rec.Code, rec.Body.String())
	}
}

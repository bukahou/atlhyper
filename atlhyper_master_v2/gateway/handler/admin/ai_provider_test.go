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

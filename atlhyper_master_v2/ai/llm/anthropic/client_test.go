package anthropic

import (
	"testing"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// 免密钥（SDK 凭证链 / WIF）必须全进程共用一个 SDK 客户端：AI 服务每次请求都新建 LLMClient，
// 而 SDK 按客户端各存一份令牌，集群令牌带 jti、同一张只能换一次 —— 第二个客户端去换就 401。
func TestNewAnthropicClient_KeylessClientsShareOneSDKClient(t *testing.T) {
	a, err := NewAnthropicClient(llm.Config{Model: "claude-haiku-5-5", BaseURL: "http://keyless-a.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewAnthropicClient(llm.Config{Model: "claude-sonnet-5-5", BaseURL: "http://keyless-a.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	if a.api != b.api {
		t.Error("同一地址的免密钥客户端应共用同一个 SDK 客户端（否则各自换令牌，jti 冲突 401）")
	}
	if a.model != "claude-haiku-5-5" || b.model != "claude-sonnet-5-5" {
		t.Errorf("模型各自独立: %q %q", a.model, b.model)
	}

	other, _ := NewAnthropicClient(llm.Config{BaseURL: "http://keyless-b.invalid"})
	if other.api == a.api {
		t.Error("不同地址应各有一个客户端")
	}

	keyed, _ := NewAnthropicClient(llm.Config{APIKey: "sk-test", BaseURL: "http://keyless-a.invalid"})
	if keyed.api == a.api {
		t.Error("配了 key 的客户端不应复用免密钥的客户端")
	}
}

func TestNewAnthropicClient_DefaultModel(t *testing.T) {
	c, err := NewAnthropicClient(llm.Config{APIKey: "sk-test"})
	if err != nil {
		t.Fatal(err)
	}
	if c.model != defaultModel || defaultModel != "claude-haiku-5-5" {
		t.Errorf("未指定模型时用 %q，实际 %q", defaultModel, c.model)
	}
}

func TestProviderRegistered(t *testing.T) {
	client, err := llm.NewLLMClient(llm.Config{Provider: "anthropic", APIKey: "sk-test", Model: "claude-haiku-5-5"})
	if err != nil {
		t.Fatalf("anthropic provider 未注册或创建失败: %v", err)
	}
	if _, ok := client.(*Client); !ok {
		t.Errorf("got %T", client)
	}
}

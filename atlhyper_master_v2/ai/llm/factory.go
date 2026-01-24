// atlhyper_master_v2/ai/llm/factory.go
// LLM 客户端工厂
// 各 provider 通过 init() 调用 Register 注册自己
// 上层通过 New(cfg) 创建客户端，无需感知底层实现
package llm

import "fmt"

// ProviderFactory 创建 LLMClient 的工厂函数
type ProviderFactory func(apiKey, model string) (LLMClient, error)

// providers 已注册的 provider 工厂
var providers = map[string]ProviderFactory{}

// Register 注册 LLM provider
// 由各 provider 包的 init() 调用
func Register(name string, factory ProviderFactory) {
	providers[name] = factory
}

// New 根据配置创建 LLMClient
// 上层调用此函数，无需直接依赖具体 provider 包
func New(cfg Config) (LLMClient, error) {
	factory, ok := providers[cfg.Provider]
	if !ok {
		available := make([]string, 0, len(providers))
		for k := range providers {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown LLM provider: %q, available: %v", cfg.Provider, available)
	}
	return factory(cfg.APIKey, cfg.Model)
}

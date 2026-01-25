// sdk/registry.go
// 提供者注册机制 - 支持可插拔的基础设施实现
package sdk

import (
	"fmt"
	"sync"
)

// ProviderFactory 提供者工厂函数
type ProviderFactory func(cfg ProviderConfig) (SDKProvider, error)

var (
	providersMu sync.RWMutex
	providers   = make(map[string]ProviderFactory)
)

// RegisterProvider 注册提供者工厂
// 通常在 init() 函数中调用，实现自动注册
//
// 示例：
//
//	func init() {
//	    sdk.RegisterProvider("kubernetes", NewK8sProvider)
//	}
func RegisterProvider(name string, factory ProviderFactory) {
	providersMu.Lock()
	defer providersMu.Unlock()

	if factory == nil {
		panic("sdk: RegisterProvider factory is nil")
	}
	if _, dup := providers[name]; dup {
		panic("sdk: RegisterProvider called twice for " + name)
	}
	providers[name] = factory
}

// NewProvider 创建提供者实例
func NewProvider(name string, cfg ProviderConfig) (SDKProvider, error) {
	providersMu.RLock()
	factory, ok := providers[name]
	providersMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("sdk: unknown provider %q (forgotten import?)", name)
	}
	return factory(cfg)
}

// ListProviders 列出所有已注册的提供者名称
func ListProviders() []string {
	providersMu.RLock()
	defer providersMu.RUnlock()

	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	return names
}

// HasProvider 检查提供者是否已注册
func HasProvider(name string) bool {
	providersMu.RLock()
	defer providersMu.RUnlock()

	_, ok := providers[name]
	return ok
}

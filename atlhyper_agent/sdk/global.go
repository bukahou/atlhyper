// sdk/global.go
// 全局单例管理 - 提供便捷的全局访问方式
package sdk

import (
	"log"
	"sync"
)

var (
	globalProvider SDKProvider
	globalOnce     sync.Once
	globalMu       sync.RWMutex
)

// Init 初始化全局提供者
// 使用 sync.Once 确保只初始化一次
//
// 示例：
//
//	sdk.Init("kubernetes", sdk.ProviderConfig{Kubeconfig: "..."})
func Init(providerName string, cfg ProviderConfig) error {
	var initErr error

	globalOnce.Do(func() {
		provider, err := NewProvider(providerName, cfg)
		if err != nil {
			initErr = err
			return
		}

		globalMu.Lock()
		globalProvider = provider
		globalMu.Unlock()

		log.Printf("[sdk] initialized provider: %s", providerName)
	})

	return initErr
}

// Get 获取全局提供者实例
// 如果未初始化会 panic，确保在 Init() 之后调用
func Get() SDKProvider {
	globalMu.RLock()
	defer globalMu.RUnlock()

	if globalProvider == nil {
		panic("[sdk] provider not initialized, call Init() first")
	}
	return globalProvider
}

// SetProvider 设置全局提供者（用于测试 Mock）
// 警告：仅用于测试，生产环境请使用 Init()
func SetProvider(p SDKProvider) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalProvider = p
}

// IsInitialized 检查是否已初始化
func IsInitialized() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalProvider != nil
}

// Close 关闭全局提供者
func Close() error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalProvider == nil {
		return nil
	}

	err := globalProvider.Close()
	globalProvider = nil
	return err
}

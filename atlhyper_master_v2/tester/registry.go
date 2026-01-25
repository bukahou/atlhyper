// atlhyper_master_v2/tester/registry.go
// 测试器注册表
package tester

import (
	"context"
	"fmt"
	"sync"
)

// Registry 测试器注册表
type Registry struct {
	testers map[string]Tester
	mu      sync.RWMutex
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		testers: make(map[string]Tester),
	}
}

// Register 注册测试器
func (r *Registry) Register(t Tester) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.testers[t.Name()] = t
}

// Get 获取测试器
func (r *Registry) Get(name string) (Tester, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.testers[name]
	return t, ok
}

// Run 执行测试
func (r *Registry) Run(ctx context.Context, name, target string) Result {
	t, ok := r.Get(name)
	if !ok {
		return NewFailureResult(fmt.Sprintf("tester not found: %s", name))
	}
	return t.Test(ctx, target)
}

// List 列出所有测试器
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.testers))
	for name := range r.testers {
		names = append(names, name)
	}
	return names
}

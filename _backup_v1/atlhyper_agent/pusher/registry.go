// logic/pusher/registry.go
// 推送器注册表
package pusher

import (
	"context"
	"log"
	"sync"
)

// Registry 推送器注册表
type Registry struct {
	mu      sync.RWMutex
	pushers map[string]Pusher
}

// NewRegistry 创建注册表
func NewRegistry() *Registry {
	return &Registry{
		pushers: make(map[string]Pusher),
	}
}

// Register 注册推送器
func (r *Registry) Register(p Pusher) {
	r.mu.Lock()
	r.pushers[p.Name()] = p
	r.mu.Unlock()
	log.Printf("[pusher_registry] registered: %s", p.Name())
}

// Get 获取推送器
func (r *Registry) Get(name string) (Pusher, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.pushers[name]
	return p, ok
}

// All 获取所有推送器
func (r *Registry) All() []Pusher {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Pusher, 0, len(r.pushers))
	for _, p := range r.pushers {
		result = append(result, p)
	}
	return result
}

// StartAll 启动所有推送器
func (r *Registry) StartAll(ctx context.Context) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.pushers {
		p.Start(ctx)
	}
	log.Printf("[pusher_registry] started %d pushers", len(r.pushers))
}

// StopAll 停止所有推送器
func (r *Registry) StopAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.pushers {
		p.Stop()
	}
	log.Printf("[pusher_registry] stopped all pushers")
}

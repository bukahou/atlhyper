package agent_store

import (
	"log"
	"sync"
	"time"
)

var bootOnce sync.Once

const (
	defaultMaxAge   = 10 * time.Minute
	defaultInterval = 1 * time.Minute
)

// Bootstrap 使用默认 TTL 与清理间隔启动全局 Store 与清理协程。
func Bootstrap() {
	bootOnce.Do(func() {
		Init()
		StartTTLJanitor(defaultMaxAge, defaultInterval)
		log.Printf("agent_store bootstrap ok (TTL=%s, interval=%s)", defaultMaxAge, defaultInterval)
	})
}

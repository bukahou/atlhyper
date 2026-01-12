// source/abnormal/exception_window.go
// 异常去重/节流控制
package abnormal

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// 内存中的异常追踪缓存（键：异常 ID）
var exceptionWindow sync.Map

// ExceptionEntry 表示单个异常记录的结构体
type ExceptionEntry struct {
	FirstTime time.Time // 首次发生时间
	LastSeen  time.Time // 最近一次发生时间
	Count     int       // 触发次数
	IsActive  bool      // 当前是否仍视为活跃异常
}

// GenerateExceptionID 生成唯一的异常 ID（用于去重）
// 格式：kind:namespace/name#reason
func GenerateExceptionID(kind, name, namespace, reason string) string {
	return fmt.Sprintf("%s:%s/%s#%s", kind, namespace, name, reason)
}

// GeneratePodInstanceExceptionID 替代格式：用于标识特定 Pod 实例（使用 UID）
func GeneratePodInstanceExceptionID(namespace string, uid types.UID, reason string) string {
	return fmt.Sprintf("pod:%s/%s#%s", namespace, uid, reason)
}

// ShouldProcessException 判断是否应该处理该异常（节流控制）
// 如果处于冷却时间内或是重复异常，则返回 false；否则更新状态并返回 true。
func ShouldProcessException(id string, now time.Time, cooldown time.Duration) bool {
	actual, loaded := exceptionWindow.LoadOrStore(id, &ExceptionEntry{
		FirstTime: now,
		LastSeen:  now,
		Count:     1,
		IsActive:  true,
	})

	entry := actual.(*ExceptionEntry)

	if loaded && entry.IsActive && now.Sub(entry.LastSeen) < cooldown {
		return false
	}

	entry.LastSeen = now
	entry.Count++
	entry.IsActive = true

	return true
}

// ResetException 手动标记异常为已恢复
// 可在资源恢复或不再异常时调用。
func ResetException(id string) {
	if v, ok := exceptionWindow.Load(id); ok {
		entry := v.(ExceptionEntry)
		entry.IsActive = false
		exceptionWindow.Store(id, entry)
	}
}

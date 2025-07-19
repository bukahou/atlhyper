// =======================================================================================
// 📄 exception_window.go
//
// ✨ Description:
//     Exception suppression controller to prevent redundant alerting and logging,
//     particularly within reconcile loops. Implements a fingerprint-based mechanism
//     using kind + name + namespace + reason to uniquely track exception states.
//
// 📦 Provided Functions:
//     - GenerateExceptionID(kind, name, namespace, reason): Generate a unique identifier
//     - ShouldProcessException(id, now, cooldown): Determine whether the exception should
//         be processed based on cooldown and activity status
//     - ResetException(id): Mark an exception as resolved, allowing future triggers
//
// 🧠 Use Cases:
//     - Avoid repetitive exception logging
//     - Stabilize controllers by suppressing noisy reprocessing
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package utils

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// 内存中的异常追踪缓存（键：异常 ID）
var exceptionWindow sync.Map

// 表示单个异常记录的结构体
type ExceptionEntry struct {
	FirstTime time.Time // 首次发生时间
	LastSeen  time.Time // 最近一次发生时间
	Count     int       // 触发次数
	IsActive  bool      // 当前是否仍视为活跃异常
}

// =======================================================================================
// ✅ 生成唯一的异常 ID（用于去重）
//
// 格式：kind:namespace/name#reason
func GenerateExceptionID(kind, name, namespace, reason string) string {
	return fmt.Sprintf("%s:%s/%s#%s", kind, namespace, name, reason)
}

// 替代格式：用于标识特定 Pod 实例（使用 UID）
func GeneratePodInstanceExceptionID(namespace string, uid types.UID, reason string) string {
	return fmt.Sprintf("pod:%s/%s#%s", namespace, uid, reason)
}

// =======================================================================================
// ✅ 判断是否应该处理该异常（节流控制）
//
// 如果处于冷却时间内或是重复异常，则返回 false；否则更新状态并返回 true。
func ShouldProcessException(id string, now time.Time, cooldown time.Duration) bool {
	actual, loaded := exceptionWindow.LoadOrStore(id, &ExceptionEntry{
		FirstTime: now,
		LastSeen:  now,
		Count:     1,
		IsActive:  true,
	})

	entry := actual.(*ExceptionEntry)

	// ✅ 调试信息
	// fmt.Printf("🧪 [节流检查] ID=%s | 是否已存在=%v | 上次出现=%s | 当前时间=%s | 时间差=%.fs | 次数=%d\n",
	// 	id, loaded, entry.LastSeen.Format(time.RFC3339), now.Format(time.RFC3339),
	// 	now.Sub(entry.LastSeen).Seconds(), entry.Count)

	if loaded && entry.IsActive && now.Sub(entry.LastSeen) < cooldown {
		// fmt.Printf("⏸️ [节流中] 跳过处理（冷却未结束）: %s（剩余 %.1fs）\n",
		// 	id, cooldown.Seconds()-now.Sub(entry.LastSeen).Seconds())
		return false
	}

	entry.LastSeen = now
	entry.Count++
	entry.IsActive = true

	// fmt.Printf("🚨 [处理异常] 正在处理异常: %s\n", id)
	return true
}

// =======================================================================================
// ✅ 手动标记异常为已恢复
//
// 可在资源恢复或不再异常时调用。
func ResetException(id string) {
	if v, ok := exceptionWindow.Load(id); ok {
		entry := v.(ExceptionEntry)
		entry.IsActive = false
		exceptionWindow.Store(id, entry)
	}
}

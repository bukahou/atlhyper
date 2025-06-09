// =======================================================================================
// 📄 exception_window.go
//
// ✨ 功能说明：
//     异常识别窗口控制器，用于识别“是否为重复异常”，防止 Reconcile 死循环和日志泛滥。
//     支持基于资源类型 + 名称 + 原因的异常指纹（ExceptionID）去重识别。
//
// 📦 提供功能：
//     - GenerateExceptionID(kind, name, namespace, reason): 生成异常指纹
//     - ShouldProcessException(id, now, cooldown): 判断是否允许处理异常
//     - ResetException(id): 手动重置某异常的状态（如异常恢复时）
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package utils

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"
)

// 异常状态缓存（ID → 异常状态）
var exceptionWindow sync.Map

// 异常记录结构
type ExceptionEntry struct {
	FirstTime time.Time // 第一次触发时间
	LastSeen  time.Time // 最近一次触发时间
	Count     int       // 触发次数
	IsActive  bool      // 是否仍处于异常中
}

// =======================================================================================
// ✅ 构造异常指纹 ID（推荐用于 Pod/Node/Deployment/Event 等）
//
// key = kind:namespace/name#reason
func GenerateExceptionID(kind, name, namespace, reason string) string {
	return fmt.Sprintf("%s:%s/%s#%s", kind, namespace, name, reason)
}

func GeneratePodInstanceExceptionID(namespace string, uid types.UID, reason string) string {
	return fmt.Sprintf("pod:%s/%s#%s", namespace, uid, reason)
}

// =======================================================================================
// ✅ 判断异常是否应被处理（用于节流）
//
// 如果处于冷却窗口内，或重复异常 → 返回 false
// 否则记录为活跃异常，更新状态 → 返回 true
func ShouldProcessException(id string, now time.Time, cooldown time.Duration) bool {
	actual, loaded := exceptionWindow.LoadOrStore(id, &ExceptionEntry{
		FirstTime: now,
		LastSeen:  now,
		Count:     1,
		IsActive:  true,
	})

	entry := actual.(*ExceptionEntry)

	// ✅ 打印调试信息
	// fmt.Printf("🧪 [异常节流判断] ID=%s | 已加载=%v | 上次=%s | 当前=%s | 距离=%.fs | 次数=%d\n",
	// 	id, loaded, entry.LastSeen.Format(time.RFC3339), now.Format(time.RFC3339),
	// 	now.Sub(entry.LastSeen).Seconds(), entry.Count)

	if loaded && entry.IsActive && now.Sub(entry.LastSeen) < cooldown {
		// fmt.Printf("⏸️ [异常节流判断] 冷却中，跳过异常：%s（冷却剩余 %.1fs）\n",
		// 	id, cooldown.Seconds()-now.Sub(entry.LastSeen).Seconds())
		return false
	}

	entry.LastSeen = now
	entry.Count++
	entry.IsActive = true

	// fmt.Printf("🚨 [异常节流判断] 允许处理异常：%s\n", id)
	return true
}

// =======================================================================================
// ✅ 手动标记异常已恢复（可在状态正常时调用）
func ResetException(id string) {
	if v, ok := exceptionWindow.Load(id); ok {
		entry := v.(ExceptionEntry)
		entry.IsActive = false
		exceptionWindow.Store(id, entry)
	}
}

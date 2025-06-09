// =======================================================================================
// 📄 exception_window.go
//
// ✨ Description:
//     Exception window controller to suppress repeated exceptions, preventing
//     Reconcile loops and log spamming.
//     Implements exception fingerprinting using kind + name + namespace + reason.
//
// 📦 Provided Functions:
//     - GenerateExceptionID(kind, name, namespace, reason): Generate unique exception ID
//     - ShouldProcessException(id, now, cooldown): Determine whether the exception should be processed
//     - ResetException(id): Manually reset the state of an exception (e.g. after recovery)
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

// In-memory exception tracking cache (key: ExceptionID)
var exceptionWindow sync.Map

// Structure representing an exception entry
type ExceptionEntry struct {
	FirstTime time.Time // First occurrence time
	LastSeen  time.Time // Most recent occurrence
	Count     int       // Number of times triggered
	IsActive  bool      // Whether the exception is still considered active
}

// =======================================================================================
// ✅ Generate a unique Exception ID (fingerprint)
//
// Format: kind:namespace/name#reason
func GenerateExceptionID(kind, name, namespace, reason string) string {
	return fmt.Sprintf("%s:%s/%s#%s", kind, namespace, name, reason)
}

// Alternative ID format for individual Pod instances (using UID)
func GeneratePodInstanceExceptionID(namespace string, uid types.UID, reason string) string {
	return fmt.Sprintf("pod:%s/%s#%s", namespace, uid, reason)
}

// =======================================================================================
// ✅ Determine whether an exception should be processed (rate-limiting)
//
// Returns false if the exception is within the cooldown window or is a duplicate.
// Otherwise, updates the tracking status and returns true.
func ShouldProcessException(id string, now time.Time, cooldown time.Duration) bool {
	actual, loaded := exceptionWindow.LoadOrStore(id, &ExceptionEntry{
		FirstTime: now,
		LastSeen:  now,
		Count:     1,
		IsActive:  true,
	})

	entry := actual.(*ExceptionEntry)

	// ✅ Debug info
	// fmt.Printf("🧪 [Throttle Check] ID=%s | Loaded=%v | LastSeen=%s | Now=%s | Δ=%.fs | Count=%d\n",
	// 	id, loaded, entry.LastSeen.Format(time.RFC3339), now.Format(time.RFC3339),
	// 	now.Sub(entry.LastSeen).Seconds(), entry.Count)

	if loaded && entry.IsActive && now.Sub(entry.LastSeen) < cooldown {
		// fmt.Printf("⏸️ [Throttle] Skipping exception (cooldown active): %s (%.1fs left)\n",
		// 	id, cooldown.Seconds()-now.Sub(entry.LastSeen).Seconds())
		return false
	}

	entry.LastSeen = now
	entry.Count++
	entry.IsActive = true

	// fmt.Printf("🚨 [Throttle] Processing exception: %s\n", id)
	return true
}

// =======================================================================================
// ✅ Manually mark an exception as resolved
//
// Can be called when the resource has recovered or is no longer abnormal.
func ResetException(id string) {
	if v, ok := exceptionWindow.Load(id); ok {
		entry := v.(ExceptionEntry)
		entry.IsActive = false
		exceptionWindow.Store(id, entry)
	}
}

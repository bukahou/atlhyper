package diagnosis

import (
	"fmt"
	"sync"
	"time"
)

// =======================================================================================
// 📄 diagnosis/cleaner.go
//
// ✨ Description:
//     Implements log event cleanup logic, including deduplication and time-based expiration.
//     Maintains a periodically refreshed `cleanedEventPool` that can be used by the matcher module.
//
// 🧼 Responsibilities:
//     - Remove outdated events from the raw event pool
//     - Merge and deduplicate events into the cleaned pool (within retention window)
//     - Provide access to the cleaned pool
//     - Run as a scheduled background cleaner
// =======================================================================================

var (
	mu               sync.Mutex
	cleanedEventPool []LogEvent // Cleaned event pool after deduplication
)

const (
	retentionRawDuration     = 10 * time.Minute
	retentionCleanedDuration = 5 * time.Minute
)

// ✅ Clean the raw event pool: retain only events from the last 10 minutes
func CleanEventPool() {
	now := time.Now()
	newRaw := make([]LogEvent, 0)
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) <= retentionRawDuration {
			newRaw = append(newRaw, ev)
		}
	}
	eventPool = newRaw
}

// ✅ Rebuild the cleaned event pool by merging new and existing entries, with deduplication
func RebuildCleanedEventPool() {
	now := time.Now()
	uniqueMap := make(map[string]LogEvent)
	newCleaned := make([]LogEvent, 0)

	// Add recent events from raw event pool (within cleaned retention)
	for _, ev := range eventPool {
		if now.Sub(ev.Timestamp) > retentionCleanedDuration {
			continue
		}
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
		if _, exists := uniqueMap[key]; !exists {
			uniqueMap[key] = ev
			newCleaned = append(newCleaned, ev)
		}
	}

	// Add remaining non-duplicated events from the previous cleaned pool
	for _, ev := range cleanedEventPool {
		if now.Sub(ev.Timestamp) <= retentionCleanedDuration {
			key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode
			if _, exists := uniqueMap[key]; !exists {
				uniqueMap[key] = ev
				newCleaned = append(newCleaned, ev)
			}
		}
	}

	cleanedEventPool = newCleaned
}

// ✅ Public function: clean both raw and cleaned event pools (thread-safe)
func CleanAndStoreEvents() {
	mu.Lock()
	defer mu.Unlock()
	CleanEventPool()
	RebuildCleanedEventPool()
}

// ✅ Get the current list of cleaned events (thread-safe)
func GetCleanedEvents() []LogEvent {
	mu.Lock()
	defer mu.Unlock()

	copy := make([]LogEvent, len(cleanedEventPool))
	copy = append(copy[:0], cleanedEventPool...)
	return copy
}

// ✅ Start the background loop that periodically cleans the event pools
//
//	(should be called from main.go or the controller entrypoint)
func StartCleanerLoop(interval time.Duration) {
	go func() {
		for {
			CleanAndStoreEvents()
			// 🧪 For debugging only — you can remove this later
			printCleanedEvents()
			time.Sleep(interval)
		}
	}()
}

// ✅ Debug: print the current status of the cleaned event pool
func printCleanedEvents() {
	events := GetCleanedEvents()
	fmt.Println("──────────────────────────────")
	fmt.Println("🧼 Current Cleaned Event Pool:")
	for _, ev := range events {
		fmt.Printf(" - [%s] %s/%s → %s (%s)\n",
			ev.Kind, ev.Namespace, ev.Name, ev.ReasonCode, ev.Timestamp.Format("15:04:05"))
	}
	fmt.Printf("🧮 Total cleaned logs: %d entries\n", len(events))
	fmt.Println("──────────────────────────────")
}

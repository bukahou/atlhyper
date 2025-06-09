package diagnosis

import (
	"fmt"
	"time"
)

// =======================================================================================
// 📄 diagnosis/diagnosis_init.go
//
// ✨ Description:
//     Entry point for starting the diagnosis system.
//     Initializes and launches both the log cleaner and log writer.
//
// 📦 Responsibilities:
//     - Configure intervals for cleaning and writing logs
//     - Start the cleaner loop (deduplication + retention)
//     - Start the file writer loop (deduplicated persistent logs)
// =======================================================================================

// 🕒 Configurable intervals (can be moved to a config package)
var (
	CleanInterval = 30 * time.Second // Interval for cleaning events
	WriteInterval = 30 * time.Second // Interval for writing events to file
)

// ✅ Start the diagnosis system: cleaner + file writer
func StartDiagnosisSystem() {
	// ✅ Startup messages
	fmt.Println("🧠 Starting Diagnosis System ...")
	fmt.Printf("🧼 Clean interval: %v\n", CleanInterval)
	fmt.Printf("📝 Write interval: %v\n", WriteInterval)

	// Start the cleaner (handles deduplication + retention)
	StartCleanerLoop(CleanInterval)

	// Start the log writer (writes deduplicated logs to file)
	go func() {
		for {
			WriteNewCleanedEventsToFile()
			time.Sleep(WriteInterval)
		}
	}()

	fmt.Println("✅ Diagnosis System started successfully.")
}

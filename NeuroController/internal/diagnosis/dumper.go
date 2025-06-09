package diagnosis

import (
	"NeuroController/internal/utils"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// =======================================================================================
// 📄 diagnosis/dumper.go
//
// ✨ Description:
//     Handles deduplicated event log persistence. Only events with meaningful changes
//     are written to disk to avoid redundancy.
//
// 📦 Responsibilities:
//     - Track event content changes using writeRecord cache
//     - Write only updated/unique events from cleaned pool to log file
//     - Support both local and in-cluster paths for writing
// =======================================================================================

type writeRecord struct {
	Message  string
	Severity string
	Category string
}

var (
	writeMu      sync.Mutex
	lastWriteMap = make(map[string]writeRecord)
)

// ✅ Write deduplicated cleaned events to file (only when content changes)
func WriteNewCleanedEventsToFile() {
	writeMu.Lock()
	defer writeMu.Unlock()

	cleaned := GetCleanedEvents()

	// ✅ If the cleaned pool is empty, system is healthy; reset write cache
	if len(cleaned) == 0 {
		lastWriteMap = make(map[string]writeRecord)
		utils.Info(nil, "✅ All alerts cleared, write cache reset")
		return
	}

	newLogs := make([]LogEvent, 0)

	for _, ev := range cleaned {
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message
		last, exists := lastWriteMap[key]

		changed := !exists ||
			ev.Message != last.Message ||
			ev.Severity != last.Severity ||
			ev.Category != last.Category

		if changed {
			newLogs = append(newLogs, ev)
			lastWriteMap[key] = writeRecord{
				Message:  ev.Message,
				Severity: ev.Severity,
				Category: ev.Category,
			}
		}
	}

	if len(newLogs) > 0 {
		DumpEventsToJSONFile(newLogs)
	}
}

// ✅ Dump given events to JSON file (append mode)
func DumpEventsToJSONFile(events []LogEvent) {
	var logDir string

	// ✅ Check if running inside Kubernetes
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		logDir = "/var/log/neurocontroller"
	} else {
		logDir = "./logs"
	}
	logPath := filepath.Join(logDir, "cleaned_events.log")

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		utils.Warn(nil, "⚠️ Failed to create log directory", zap.Error(err))
		return
	}

	// Open file in append mode
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.Warn(nil, "⚠️ Failed to open log file for writing", zap.Error(err))
		return
	}
	defer f.Close()

	for _, ev := range events {
		entry := map[string]interface{}{
			"time":      time.Now().Format(time.RFC3339),
			"kind":      ev.Kind,
			"namespace": ev.Namespace,
			"name":      ev.Name,
			"reason":    ev.ReasonCode,
			"message":   ev.Message,
			"severity":  ev.Severity,
			"category":  ev.Category,
			"eventTime": ev.Timestamp.Format(time.RFC3339),
		}

		data, err := json.Marshal(entry)
		if err != nil {
			utils.Warn(nil, "⚠️ Failed to serialize event", zap.Error(err))
			continue
		}

		f.Write(data)
		f.Write([]byte("\n"))
	}
}

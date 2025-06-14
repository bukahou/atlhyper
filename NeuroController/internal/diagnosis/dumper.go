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

package diagnosis

import (
	"NeuroController/internal/types"
	"NeuroController/internal/utils"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

type writeRecord struct {
	Message  string
	Severity string
	Category string
}

var (
	writeMu      sync.Mutex
	lastWriteMap = make(map[string]writeRecord)
)

// ✅ 将去重后的清理事件写入文件（仅在内容变更时写入）
func WriteNewCleanedEventsToFile() {
	writeMu.Lock()
	defer writeMu.Unlock()

	cleaned := GetCleanedEvents()

	// ✅ 如果清理池为空，说明系统健康，重置写入缓存
	if len(cleaned) == 0 {
		lastWriteMap = make(map[string]writeRecord)
		utils.Info(nil, "✅ 所有告警已清除，写入缓存已重置")
		return
	}

	newLogs := make([]types.LogEvent, 0)

	for _, ev := range cleaned {
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message
		last, exists := lastWriteMap[key]

		// 检查是否有变更（新增或内容变更）
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

// ✅ 将传入的事件追加写入 JSON 文件
func DumpEventsToJSONFile(events []types.LogEvent) {
	var logDir string

	// ✅ 判断是否运行在 Kubernetes 集群中
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		logDir = "/var/log/neurocontroller"
	} else {
		logDir = "./logs"
	}
	logPath := filepath.Join(logDir, "cleaned_events.log")

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		utils.Warn(nil, "⚠️ 创建日志目录失败", zap.Error(err))
		return
	}

	// 以追加模式打开文件
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.Warn(nil, "⚠️ 打开日志文件失败", zap.Error(err))
		return
	}
	defer f.Close()

	for _, ev := range events {
		entry := map[string]interface{}{
			"time":      time.Now().Format(time.RFC3339),
			"kind":      ev.Kind,
			"namespace": ev.Namespace,
			"name":      ev.Name,
			"node":      ev.Node,
			"reason":    ev.ReasonCode,
			"message":   ev.Message,
			"severity":  ev.Severity,
			"category":  ev.Category,
			"eventTime": ev.Timestamp.Format(time.RFC3339),
		}

		data, err := json.Marshal(entry)
		if err != nil {
			utils.Warn(nil, "⚠️ 事件序列化失败", zap.Error(err))
			continue
		}

		f.Write(data)
		f.Write([]byte("\n"))
	}
}

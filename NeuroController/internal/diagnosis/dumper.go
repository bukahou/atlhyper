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

type writeRecord struct {
	Message  string
	Severity string
	Category string
}

var (
	writeMu      sync.Mutex
	lastWriteMap = make(map[string]writeRecord)
)

// ✅ 只在内容变化时写入
func WriteNewCleanedEventsToFile() {
	writeMu.Lock()
	defer writeMu.Unlock()

	cleaned := GetCleanedEvents()

	// ✅ 若清理池为空，表示系统状态恢复，重置写入状态缓存
	if len(cleaned) == 0 {
		lastWriteMap = make(map[string]writeRecord)
		utils.Info(nil, "✅ 所有告警清除，写入状态已重置")
		return
	}

	newLogs := make([]LogEvent, 0)

	for _, ev := range cleaned {
		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message
		last, exists := lastWriteMap[key]

		changed := !exists || ev.Message != last.Message || ev.Severity != last.Severity || ev.Category != last.Category

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

func DumpEventsToJSONFile(events []LogEvent) {
	// ✅ 判断是否运行在 Kubernetes
	var logDir string
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		logDir = "/var/log/neurocontroller"
	} else {
		logDir = "./logs"
	}
	logPath := filepath.Join(logDir, "cleaned_events.log")

	// 创建目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		utils.Warn(nil, "⚠️ 无法创建日志目录", zap.Error(err))
		return
	}

	// 打开文件
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		utils.Warn(nil, "⚠️ 无法写入清理日志文件", zap.Error(err))
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
			utils.Warn(nil, "⚠️ 事件序列化失败", zap.Error(err))
			continue
		}

		f.Write(data)
		f.Write([]byte("\n"))
	}
}

// ✅ 写入日志文件
// func DumpEventsToFile(events []LogEvent) {
// 	logDir := "./logs"
// 	logPath := logDir + "/cleaned_events.log"

// 	if err := os.MkdirAll(logDir, 0755); err != nil {
// 		utils.Warn(nil, "⚠️ 无法创建日志目录", zap.Error(err))
// 		return
// 	}

// 	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		utils.Warn(nil, "⚠️ 无法写入清理日志文件", zap.Error(err))
// 		return
// 	}
// 	defer f.Close()

// 	timestamp := time.Now().Format("2006-01-02 15:04:05")
// 	f.WriteString("🕒 Dump at " + timestamp + "\n")

// 	for _, ev := range events {
// 		line := fmt.Sprintf(" - [%s] %s/%s → %s：%s (%s)\n",
// 			ev.Kind, ev.Namespace, ev.Name, ev.ReasonCode, ev.Message, ev.Timestamp.Format("15:04:05"))
// 		f.WriteString(line)
// 	}

// 	f.WriteString("──────────────────────────────\n")
// }

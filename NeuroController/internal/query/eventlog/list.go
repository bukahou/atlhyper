// 📄 internal/query/eventlog/list.go

package eventlog

import (
	"NeuroController/internal/types"
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// GetRecentEventLogs 返回最近 N 天内的日志（N=1 表示过去 24 小时内）
func GetRecentEventLogs(withinDays int) ([]types.LogEvent, error) {
	var logDir string

	// 判断是否运行在 K8s 中
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		logDir = "/var/log/neurocontroller"
	} else {
		logDir = "./logs"
	}

	logPath := filepath.Join(logDir, "cleaned_events.log")

	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cutoff := time.Now().Add(-time.Duration(withinDays) * 24 * time.Hour)

	var events []types.LogEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		// 解析时间
		eventTime := parseTime(entry["eventTime"])
		if eventTime.Before(cutoff) {
			continue // 忽略早于 N 天的
		}

		ev := types.LogEvent{
			Kind:       getString(entry["kind"]),
			Namespace:  getString(entry["namespace"]),
			Name:       getString(entry["name"]),
			Node:       getString(entry["node"]),
			ReasonCode: getString(entry["reason"]),
			Message:    getString(entry["message"]),
			Severity:   getString(entry["severity"]),
			Category:   getString(entry["category"]),
			Timestamp:  eventTime,
		}
		events = append(events, ev)
	}

	return events, scanner.Err()
}

func getString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseTime(v interface{}) (t time.Time) {
	if s, ok := v.(string); ok {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return
}

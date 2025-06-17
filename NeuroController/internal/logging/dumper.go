// DumpEventsToJSONFile ✅ 将传入的结构化事件列表追加写入 JSON 格式日志文件（换行分隔）
//
// 📦 功能：
//   - 支持在 Kubernetes 容器内或本地开发环境下写入日志文件
//   - 每条事件独立以 JSON 格式序列化并换行写入（方便 Filebeat/Fluentd 解析）
//   - 写入位置根据运行环境自动切换（/var/log/neurocontroller 或 ./logs）
//
// 🚨 错误处理：
//   - 若目录或文件创建失败，会记录日志并跳过写入
//   - 每条事件单独序列化与写入，不影响其他事件持久化

package logging

import (
	"NeuroController/internal/types"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"
)

func DumpEventsToJSONFile(events []types.LogEvent) {
	var logDir string

	// 🔍 判断是否运行在 Kubernetes Pod 内部（通过 serviceaccount 路径判断）
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
		logDir = "/var/log/neurocontroller" // ✅ 正式部署路径（持久卷挂载点）
	} else {
		logDir = "./logs" // ✅ 本地开发调试路径
	}

	// 📄 拼接日志文件路径
	logPath := filepath.Join(logDir, "cleaned_events.log")

	// 📁 确保日志目录存在（权限：0755）
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("❌ 创建日志目录失败: %v", err)
		return
	}

	// ✏️ 打开日志文件（追加模式），若不存在则自动创建（权限：0644）
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("❌ 打开日志文件失败: %v", err)
		return
	}
	defer f.Close()

	// 📦 遍历传入事件列表，逐条写入
	for _, ev := range events {
		// 🧱 构造日志 entry（JSON 格式字段）
		entry := map[string]interface{}{
			"time":      time.Now().Format(time.RFC3339), // 写入时间（记录行为时间）
			"kind":      ev.Kind,
			"namespace": ev.Namespace,
			"name":      ev.Name,
			"node":      ev.Node,
			"reason":    ev.ReasonCode,
			"message":   ev.Message,
			"severity":  ev.Severity,
			"category":  ev.Category,
			"eventTime": ev.Timestamp.Format(time.RFC3339), // 原始事件时间
		}

		// 🔄 序列化为 JSON
		data, err := json.Marshal(entry)
		if err != nil {
			log.Printf("❌ 序列化事件失败: %v", err)
			continue // ⚠️ 序列化失败则跳过当前事件
		}

		// 🖋 写入 JSON 数据（单行）
		if _, err := f.Write(data); err != nil {
			log.Printf("❌ 写入日志文件失败: %v", err)
			continue
		}

		// ➕ 写入换行符（便于日志采集器一行一条记录）
		if _, err := f.Write([]byte("\n")); err != nil {
			log.Printf("❌ 写入换行失败: %v", err)
		}
	}
}

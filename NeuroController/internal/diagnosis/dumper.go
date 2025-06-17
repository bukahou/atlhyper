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

// type writeRecord struct {
// 	Message  string
// 	Severity string
// 	Category string
// }

// var (
// 	writeMu      sync.Mutex
// 	lastWriteMap = make(map[string]writeRecord)
// )

// WriteNewCleanedEventsToFile ✅ 将清理池中“新增或变更”的事件写入 JSON 文件（带写入缓存去重）
//
// ✨ 功能：
//   - 避免重复写入：仅写入与上一次相比内容发生变化的事件
//   - 记录写入缓存（lastWriteMap），用于判断事件是否“真正更新”
//   - 使用互斥锁 writeMu 保证并发安全
//   - 写入时调用 DumpEventsToJSONFile，并用 recover 防止崩溃
//
// 📦 使用场景：
//   - 由定时器周期性触发，将更新过的清理事件持久化
//   - 提供结构化日志供后续分析与查询
// func WriteNewCleanedEventsToFile() {
// 	// 🧵 加锁，避免与其他写入操作并发冲突
// 	writeMu.Lock()
// 	defer writeMu.Unlock()

// 	// 🧪 获取当前清理池快照（已去重 & 时间过滤）
// 	cleaned := GetCleanedEvents()

// 	// ✅ 清理池为空时，表示系统健康或已恢复，清空写入缓存以便后续重建差异状态
// 	if len(cleaned) == 0 {
// 		lastWriteMap = make(map[string]writeRecord)
// 		return
// 	}

// 	// 📥 存放需要写入的新事件
// 	newLogs := make([]types.LogEvent, 0)

// 	// 🔁 遍历清理池，检测是否为“首次写入”或“字段有变化”
// 	for _, ev := range cleaned {
// 		// 生成用于比对的唯一键（包含 Kind + Namespace + Name + ReasonCode + Message）
// 		key := ev.Kind + "|" + ev.Namespace + "|" + ev.Name + "|" + ev.ReasonCode + "|" + ev.Message

// 		// 获取上一轮写入的记录
// 		last, exists := lastWriteMap[key]

// 		// 判断事件是否有变化：
// 		//   - 首次出现
// 		//   - message 字段变更
// 		//   - severity 级别变更
// 		//   - category 分类变更
// 		changed := !exists ||
// 			ev.Message != last.Message ||
// 			ev.Severity != last.Severity ||
// 			ev.Category != last.Category

// 		// 若存在变化，则添加进待写入列表，并更新写入缓存
// 		if changed {
// 			newLogs = append(newLogs, ev)
// 			lastWriteMap[key] = writeRecord{
// 				Message:  ev.Message,
// 				Severity: ev.Severity,
// 				Category: ev.Category,
// 			}
// 		}
// 	}

// 	// ✅ 如果存在变更事件，则触发写入
// 	if len(newLogs) > 0 {
// 		// ⚠️ 用 defer + recover 保护写入流程，防止 JSON 写入崩溃影响主流程
// 		defer func() {
// 			if r := recover(); r != nil {
// 				log.Printf("❌ 写入 JSON 文件过程中发生 panic: %v", r)
// 			}
// 		}()

// 		// ✍️ 调用写入函数（按 JSON 单行格式追加写入）
// 		DumpEventsToJSONFile(newLogs)
// 	}
// }

// // DumpEventsToJSONFile ✅ 将传入的结构化事件列表追加写入 JSON 格式日志文件（换行分隔）
// //
// // 📦 功能：
// //   - 支持在 Kubernetes 容器内或本地开发环境下写入日志文件
// //   - 每条事件独立以 JSON 格式序列化并换行写入（方便 Filebeat/Fluentd 解析）
// //   - 写入位置根据运行环境自动切换（/var/log/neurocontroller 或 ./logs）
// //
// // 🚨 错误处理：
// //   - 若目录或文件创建失败，会记录日志并跳过写入
// //   - 每条事件单独序列化与写入，不影响其他事件持久化
// func DumpEventsToJSONFile(events []types.LogEvent) {
// 	var logDir string

// 	// 🔍 判断是否运行在 Kubernetes Pod 内部（通过 serviceaccount 路径判断）
// 	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
// 		logDir = "/var/log/neurocontroller" // ✅ 正式部署路径（持久卷挂载点）
// 	} else {
// 		logDir = "./logs" // ✅ 本地开发调试路径
// 	}

// 	// 📄 拼接日志文件路径
// 	logPath := filepath.Join(logDir, "cleaned_events.log")

// 	// 📁 确保日志目录存在（权限：0755）
// 	if err := os.MkdirAll(logDir, 0755); err != nil {
// 		log.Printf("❌ 创建日志目录失败: %v", err)
// 		return
// 	}

// 	// ✏️ 打开日志文件（追加模式），若不存在则自动创建（权限：0644）
// 	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		log.Printf("❌ 打开日志文件失败: %v", err)
// 		return
// 	}
// 	defer f.Close()

// 	// 📦 遍历传入事件列表，逐条写入
// 	for _, ev := range events {
// 		// 🧱 构造日志 entry（JSON 格式字段）
// 		entry := map[string]interface{}{
// 			"time":      time.Now().Format(time.RFC3339), // 写入时间（记录行为时间）
// 			"kind":      ev.Kind,
// 			"namespace": ev.Namespace,
// 			"name":      ev.Name,
// 			"node":      ev.Node,
// 			"reason":    ev.ReasonCode,
// 			"message":   ev.Message,
// 			"severity":  ev.Severity,
// 			"category":  ev.Category,
// 			"eventTime": ev.Timestamp.Format(time.RFC3339), // 原始事件时间
// 		}

// 		// 🔄 序列化为 JSON
// 		data, err := json.Marshal(entry)
// 		if err != nil {
// 			log.Printf("❌ 序列化事件失败: %v", err)
// 			continue // ⚠️ 序列化失败则跳过当前事件
// 		}

// 		// 🖋 写入 JSON 数据（单行）
// 		if _, err := f.Write(data); err != nil {
// 			log.Printf("❌ 写入日志文件失败: %v", err)
// 			continue
// 		}

// 		// ➕ 写入换行符（便于日志采集器一行一条记录）
// 		if _, err := f.Write([]byte("\n")); err != nil {
// 			log.Printf("❌ 写入换行失败: %v", err)
// 		}
// 	}
// }

// func DumpEventsToJSONFile(events []types.LogEvent) {
// 	var logDir string

// 	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount"); err == nil {
// 		logDir = "/var/log/neurocontroller"
// 	} else {
// 		logDir = "./logs"
// 	}
// 	logPath := filepath.Join(logDir, "cleaned_events.log")

// 	if err := os.MkdirAll(logDir, 0755); err != nil {
// 		log.Printf("❌ 创建日志目录失败: %v", err)
// 		return
// 	}

// 	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
// 	if err != nil {
// 		log.Printf("❌ 打开日志文件失败: %v", err)
// 		return
// 	}
// 	defer f.Close()

// 	for _, ev := range events {
// 		entry := map[string]interface{}{
// 			"time":      time.Now().Format(time.RFC3339),
// 			"kind":      ev.Kind,
// 			"namespace": ev.Namespace,
// 			"name":      ev.Name,
// 			"node":      ev.Node,
// 			"reason":    ev.ReasonCode,
// 			"message":   ev.Message,
// 			"severity":  ev.Severity,
// 			"category":  ev.Category,
// 			"eventTime": ev.Timestamp.Format(time.RFC3339),
// 		}

// 		data, err := json.Marshal(entry)
// 		if err != nil {
// 			continue
// 		}

// 		f.Write(data)
// 		f.Write([]byte("\n"))
// 	}
// }

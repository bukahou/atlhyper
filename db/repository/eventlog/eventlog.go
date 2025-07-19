package eventlog

import (
	"NeuroController/db/utils"
	"NeuroController/model"

	_ "github.com/mattn/go-sqlite3" // ✅ 引入 SQLite3 驱动（初始化阶段注册）
)

// ==============================================
// ✅ InsertEventLog：插入一条事件日志记录
// ==============================================
// - 参数：model.EventLog 结构体，包含所有事件字段
// - 使用 utils.DB 全局 SQLite 数据库连接对象
// - 返回：执行出错时返回 error，否则为 nil
func InsertEventLog(e model.EventLog) error {
	_, err := utils.DB.Exec(`
		INSERT INTO event_logs (category, eventTime, kind, message, name, namespace, node, reason, severity, time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Category, e.EventTime, e.Kind, e.Message, e.Name,
		e.Namespace, e.Node, e.Reason, e.Severity, e.Time)
	return err
}

// =======================================================
// ✅ GetEventLogsSince：获取指定时间之后的事件日志
// =======================================================
// - 参数：since（字符串，通常为 RFC3339 时间戳）
// - 查询所有 event_logs 中 eventTime >= since 的记录
// - 返回：[]model.EventLog 切片 和可能的 error
func GetEventLogsSince(since string) ([]model.EventLog, error) {
	rows, err := utils.DB.Query(`
		SELECT category, eventTime, kind, message, name, namespace, node, reason, severity, time
		FROM event_logs WHERE eventTime >= ?`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.EventLog

	// 遍历结果集并逐条解析
	for rows.Next() {
		var e model.EventLog
		err := rows.Scan(&e.Category, &e.EventTime, &e.Kind, &e.Message, &e.Name,
			&e.Namespace, &e.Node, &e.Reason, &e.Severity, &e.Time)
		if err != nil {
			continue // 忽略解析错误的行
		}
		logs = append(logs, e)
	}

	return logs, nil
}

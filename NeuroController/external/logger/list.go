// 📄 internal/query/eventlog/list.go

package logger

import (
	"NeuroController/db/repository/eventlog"
	"NeuroController/model"
	"time"
)

// =======================================================================================
// ✅ GetRecentEventLogs - 查询最近 N 天内的事件日志
//
// 📌 用法：
//     - 根据时间范围，从 SQLite 中查询指定天数以内的事件记录（event_logs）
//     - 使用 RFC3339 格式构造 since 时间戳传入底层查询
//
// 🧩 调用链：
//     internal/query/eventlog/list.go → repository/eventlog.GetEventLogsSince()
//
// ⚠️ 注意：
//     - 此函数依赖全局 sqlite.DB（已由 utils 初始化）
//     - 不做分页，如需分页建议在上层增加处理
// =======================================================================================
func GetRecentEventLogs(withinDays int) ([]model.EventLog, error) {
	// 构造起始时间戳：当前时间 - N 天
	since := time.Now().Add(-time.Duration(withinDays) * 24 * time.Hour).Format(time.RFC3339)

	// 调用底层持久层查询函数
	return eventlog.GetEventLogsSince(since)
}

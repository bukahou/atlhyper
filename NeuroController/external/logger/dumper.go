package logger

import (
	"NeuroController/db/repository/eventlog"
	"NeuroController/model"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3" // 引入 SQLite 驱动（供底层使用）
)

// =======================================================================================
// ✅ DumpEventsToSQLite - 批量写入事件日志到 SQLite 数据库
//
// 📌 用法：
//     - 接收处理后的结构化事件列表（LogEvent）
//     - 转换为 EventLog 数据库模型后，逐条插入 SQLite
//
// ⚠️ 注意：
//     - 采用逐条插入（不批量），如需优化性能可考虑事务批量提交
//     - 插入失败时会记录日志，但不会中断循环（容错）
// =======================================================================================
func DumpEventsToSQLite(events []model.LogEvent) {
	for _, ev := range events {
		// 构造用于持久化的事件结构（EventLog）
		err := eventlog.InsertEventLog(model.EventLog{
			Category:  ev.Category,                       // 异常类型分类（如 Pod、Node 等）
			EventTime: ev.Timestamp.Format(time.RFC3339), // 原始事件时间
			Kind:      ev.Kind,                          // 资源类型
			Message:   ev.Message,                       // 事件消息
			Name:      ev.Name,                          // 对象名称
			Namespace: ev.Namespace,                     // 命名空间
			Node:      ev.Node,                          // 所属节点
			Reason:    ev.ReasonCode,                    // 事件原因
			Severity:  ev.Severity,                      // 严重程度（如 Warning / Critical）
			Time:      time.Now().Format(time.RFC3339),  // 写入时间（记录采集时间）
		})

		// 写入失败时记录日志，但不中断
		if err != nil {
			log.Printf("❌ 插入事件到数据库失败: %v", err)
		}
	}
}

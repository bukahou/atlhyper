// repository/sql/event.go
//
// 事件日志仓库 SQL 实现
//
// 本文件实现了 repository.EventRepository 接口，提供 Kubernetes 集群事件的存储和查询。
// 事件日志从各个集群的 Agent 收集，用于集中式事件监控和分析。
//
// 实现的接口方法:
//   - Insert: 插入单条事件
//   - InsertBatch: 批量插入事件 (事务)
//   - GetSince: 查询指定时间之后的事件
//
// 数据库表: event_logs
// 字段: cluster_id, category, eventTime, kind, message, name, namespace, node, reason, severity, time
//
// 事件类型 (kind):
//   - Pod: Pod 相关事件
//   - Node: 节点相关事件
//   - Deployment: 部署相关事件
//   - Service: 服务相关事件
//   - 等等...
//
// 严重程度 (severity):
//   - Normal: 正常事件
//   - Warning: 警告事件
package sql

import (
	"context"

	"AtlHyper/atlhyper_master/store"
	transport "AtlHyper/model/transport"
)

// ============================================================================
// EventRepo 事件日志仓库实现
// ============================================================================

// EventRepo 事件日志仓库的 SQL 实现
// 实现 repository.EventRepository 接口
type EventRepo struct{}

// ============================================================================
// 写入方法
// ============================================================================

// Insert 插入单条事件日志
//
// 参数:
//   - ctx: 上下文
//   - e: 事件日志实体，包含以下字段:
//   - ClusterID: 集群标识
//   - Category: 事件分类
//   - EventTime: 事件发生时间
//   - Kind: 资源类型 (Pod, Node, Deployment 等)
//   - Message: 事件消息
//   - Name: 资源名称
//   - Namespace: 命名空间
//   - Node: 节点名称
//   - Reason: 事件原因
//   - Severity: 严重程度 (Normal, Warning)
//   - Time: 记录时间
//
// 返回:
//   - error: 插入失败时返回错误
//
// 使用场景:
//   - Agent 上报单个事件时使用
//   - 实时事件推送
func (r *EventRepo) Insert(ctx context.Context, e *transport.EventLog) error {
	_, err := store.DB.ExecContext(ctx, Q.Event.Insert,
		e.ClusterID, e.Category, e.EventTime, e.Kind, e.Message, e.Name,
		e.Namespace, e.Node, e.Reason, e.Severity, e.Time)
	return err
}

// InsertBatch 批量插入事件日志
//
// 参数:
//   - ctx: 上下文
//   - events: 事件日志列表
//
// 返回:
//   - error: 插入失败时返回错误，会自动回滚事务
//
// 使用场景:
//   - Agent 定期批量上报事件
//   - 历史事件导入
//
// 实现细节:
//   - 使用事务保证批量插入的原子性
//   - 使用预编译语句提高性能
//   - 任意一条失败则整体回滚
//
// 性能考虑:
//   - 对于大批量数据，建议分批次调用
//   - 每批次建议 100-500 条
func (r *EventRepo) InsertBatch(ctx context.Context, events []transport.EventLog) error {
	// 开启事务
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// 确保失败时回滚
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 预编译 INSERT 语句
	stmt, err := tx.PrepareContext(ctx, Q.Event.Insert)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 批量执行插入
	for _, e := range events {
		if _, err = stmt.ExecContext(ctx,
			e.ClusterID, e.Category, e.EventTime, e.Kind, e.Message, e.Name,
			e.Namespace, e.Node, e.Reason, e.Severity, e.Time); err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit()
}

// ============================================================================
// 查询方法
// ============================================================================

// GetSince 查询指定时间之后的事件
//
// 参数:
//   - ctx: 上下文
//   - clusterID: 集群标识
//   - since: 起始时间 (RFC3339 格式)
//
// 返回:
//   - []transport.EventLog: 事件列表，按事件时间倒序排列
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 前端轮询获取最新事件
//   - 事件时间线展示
//   - 增量事件同步
//
// 时间格式示例:
//   - "2024-01-15T10:30:00Z"
//   - "2024-01-15T10:30:00+08:00"
//
// 注意:
//   - 返回的是指定集群的事件
//   - 最新的事件在前
//   - 扫描错误的行会被跳过，不会中断整个查询
func (r *EventRepo) GetSince(ctx context.Context, clusterID string, since string) ([]transport.EventLog, error) {
	rows, err := store.DB.QueryContext(ctx, Q.Event.GetSince, clusterID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []transport.EventLog
	for rows.Next() {
		var e transport.EventLog
		err := rows.Scan(
			&e.ClusterID, &e.Category, &e.EventTime, &e.Kind, &e.Message, &e.Name,
			&e.Namespace, &e.Node, &e.Reason, &e.Severity, &e.Time,
		)
		if err != nil {
			// 跳过扫描失败的行，继续处理其他行
			continue
		}
		logs = append(logs, e)
	}

	return logs, rows.Err()
}

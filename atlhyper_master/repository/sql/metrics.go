// repository/sql/metrics.go
//
// 节点指标仓库 SQL 实现
//
// 本文件实现了 repository.MetricsRepository 接口，提供节点监控指标的存储和查询。
// 指标数据从各个集群的 Agent 收集，用于实时监控和历史分析。
//
// 实现的接口方法:
//   - UpsertNodeMetrics: 插入或更新节点指标
//   - UpsertTopProcesses: 插入或更新 TOP 进程信息
//   - CleanupBefore: 清理过期数据
//   - GetLatestByNode: 获取节点最新指标
//
// 数据库表:
//   - node_metrics_flat: 节点指标数据 (扁平化存储)
//   - node_top_processes: 节点 TOP 进程信息
//
// 指标类型:
//   - CPU: 使用率、核心数、负载 (1/5/15分钟)
//   - 内存: 总量、已用、可用、使用率
//   - 温度: CPU、GPU、NVMe
//   - 磁盘: 总量、已用、可用、使用率
//   - 网络: 本地环回、eth0 的收发速率
//
// 数据生命周期:
//   定期调用 CleanupBefore 清理过期数据，防止数据库无限增长
package sql

import (
	"context"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/store"
)

// ============================================================================
// MetricsRepo 节点指标仓库实现
// ============================================================================

// MetricsRepo 节点指标仓库的 SQL 实现
// 实现 repository.MetricsRepository 接口
type MetricsRepo struct{}

// ============================================================================
// 写入方法
// ============================================================================

// UpsertNodeMetrics 插入或更新节点指标
//
// 参数:
//   - ctx: 上下文
//   - m: 节点指标数据，包含以下字段:
//   - NodeName: 节点名称 (主键之一)
//   - Timestamp: 采集时间戳 (主键之一)
//   - CPU 相关: CPUUsage, CPUCores, CPULoad1/5/15
//   - 内存相关: MemoryTotal, MemoryUsed, MemoryAvailable, MemoryUsage
//   - 温度相关: TempCPU, TempGPU, TempNVME
//   - 磁盘相关: DiskTotal, DiskUsed, DiskFree, DiskUsage
//   - 网络相关: NetLoRxKBps, NetLoTxKBps, NetEth0RxKBps, NetEth0TxKBps
//
// 返回:
//   - error: 操作失败时返回错误
//
// 实现细节:
//   使用 UPSERT (ON CONFLICT...DO UPDATE) 语法:
//   - 如果 (node_name, ts) 不存在，执行 INSERT
//   - 如果 (node_name, ts) 已存在，执行 UPDATE
//
// 使用场景:
//   - Agent 定期上报节点指标
//   - 补录历史指标数据
func (r *MetricsRepo) UpsertNodeMetrics(ctx context.Context, m *repository.NodeMetricsFlat) error {
	_, err := store.DB.ExecContext(ctx, Q.Metrics.UpsertNodeMetrics,
		// 主键
		m.NodeName, m.Timestamp,
		// CPU 指标
		m.CPUUsage, m.CPUCores, m.CPULoad1, m.CPULoad5, m.CPULoad15,
		// 内存指标
		m.MemoryTotal, m.MemoryUsed, m.MemoryAvailable, m.MemoryUsage,
		// 温度指标
		m.TempCPU, m.TempGPU, m.TempNVME,
		// 磁盘指标
		m.DiskTotal, m.DiskUsed, m.DiskFree, m.DiskUsage,
		// 网络指标
		m.NetLoRxKBps, m.NetLoTxKBps, m.NetEth0RxKBps, m.NetEth0TxKBps)
	return err
}

// UpsertTopProcesses 插入或更新 TOP 进程信息
//
// 参数:
//   - ctx: 上下文
//   - nodeName: 节点名称
//   - ts: 采集时间戳
//   - procs: TOP 进程列表，每个进程包含:
//   - PID: 进程ID
//   - User: 进程所属用户
//   - Command: 进程命令
//   - CPUPercent: CPU 使用率
//   - MemoryMB: 内存使用量 (MB)
//
// 返回:
//   - error: 操作失败时返回错误，会自动回滚事务
//
// 实现细节:
//   - 使用事务保证批量操作的原子性
//   - 使用预编译语句提高性能
//   - 主键为 (node_name, ts, pid)
//   - 使用 UPSERT 处理重复数据
//
// 使用场景:
//   - Agent 定期上报高资源占用进程
//   - 用于问题排查和资源分析
func (r *MetricsRepo) UpsertTopProcesses(ctx context.Context, nodeName, ts string, procs []repository.TopProcess) error {
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

	// 预编译 UPSERT 语句
	stmt, err := tx.PrepareContext(ctx, Q.Metrics.UpsertTopProcesses)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 批量执行
	for _, p := range procs {
		if _, err = stmt.ExecContext(ctx,
			nodeName, ts,
			p.PID, p.User, p.Command, p.CPUPercent, p.MemoryMB); err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit()
}

// ============================================================================
// 数据清理方法
// ============================================================================

// CleanupBefore 清理指定时间之前的过期数据
//
// 参数:
//   - ctx: 上下文
//   - cutoff: 截止时间 (RFC3339 格式)，早于此时间的数据将被删除
//
// 返回:
//   - metricsDeleted: 删除的指标记录数
//   - procsDeleted: 删除的进程记录数
//   - error: 操作失败时返回错误
//
// 实现细节:
//   - 使用事务保证两个表的删除操作原子性
//   - 同时清理 node_metrics_flat 和 node_top_processes 表
//
// 使用场景:
//   - 定期任务调用，清理 N 天前的历史数据
//   - 磁盘空间不足时手动清理
//
// 示例:
//
//	// 清理 7 天前的数据
//	cutoff := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
//	deleted, procsDeleted, err := repo.CleanupBefore(ctx, cutoff)
func (r *MetricsRepo) CleanupBefore(ctx context.Context, cutoff string) (metricsDeleted, procsDeleted int64, err error) {
	// 开启事务
	tx, err := store.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	// 确保失败时回滚
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 删除过期的指标数据
	res1, err := tx.ExecContext(ctx, Q.Metrics.DeleteMetrics, cutoff)
	if err != nil {
		return 0, 0, err
	}
	metricsDeleted, _ = res1.RowsAffected()

	// 删除过期的进程数据
	res2, err := tx.ExecContext(ctx, Q.Metrics.DeleteProcesses, cutoff)
	if err != nil {
		return 0, 0, err
	}
	procsDeleted, _ = res2.RowsAffected()

	// 提交事务
	if err = tx.Commit(); err != nil {
		return 0, 0, err
	}
	return metricsDeleted, procsDeleted, nil
}

// ============================================================================
// 查询方法
// ============================================================================

// GetLatestByNode 获取指定节点的最新指标
//
// 参数:
//   - ctx: 上下文
//   - nodeName: 节点名称
//
// 返回:
//   - *repository.NodeMetricsFlat: 节点最新指标数据
//   - error: 查询失败或节点不存在时返回错误
//
// 使用场景:
//   - 实时监控面板展示当前状态
//   - 健康检查判断节点是否在线
//
// 注意:
//   - 返回时间戳最新的一条记录
//   - 如果节点无数据，返回 sql.ErrNoRows
func (r *MetricsRepo) GetLatestByNode(ctx context.Context, nodeName string) (*repository.NodeMetricsFlat, error) {
	row := store.DB.QueryRowContext(ctx, Q.Metrics.GetLatestByNode, nodeName)

	var m repository.NodeMetricsFlat
	err := row.Scan(
		// 基本信息
		&m.ID, &m.NodeName, &m.Timestamp,
		// CPU 指标
		&m.CPUUsage, &m.CPUCores, &m.CPULoad1, &m.CPULoad5, &m.CPULoad15,
		// 内存指标
		&m.MemoryTotal, &m.MemoryUsed, &m.MemoryAvailable, &m.MemoryUsage,
		// 温度指标
		&m.TempCPU, &m.TempGPU, &m.TempNVME,
		// 磁盘指标
		&m.DiskTotal, &m.DiskUsed, &m.DiskFree, &m.DiskUsage,
		// 网络指标
		&m.NetLoRxKBps, &m.NetLoTxKBps, &m.NetEth0RxKBps, &m.NetEth0TxKBps)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

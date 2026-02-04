// atlhyper_master_v2/database/sqlite/node_metrics.go
// NodeMetrics SQLite 方言实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// nodeMetricsDialect NodeMetrics SQLite 方言
type nodeMetricsDialect struct{}

// UpsertLatest 更新或插入实时数据
func (d *nodeMetricsDialect) UpsertLatest(m *database.NodeMetricsLatest) (string, []any) {
	return `INSERT INTO node_metrics_latest (cluster_id, node_name, snapshot_json, cpu_usage, memory_usage, disk_usage, cpu_temp, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cluster_id, node_name) DO UPDATE SET
			snapshot_json = excluded.snapshot_json,
			cpu_usage = excluded.cpu_usage,
			memory_usage = excluded.memory_usage,
			disk_usage = excluded.disk_usage,
			cpu_temp = excluded.cpu_temp,
			updated_at = excluded.updated_at`,
		[]any{m.ClusterID, m.NodeName, m.SnapshotJSON, m.CPUUsage, m.MemoryUsage, m.DiskUsage, m.CPUTemp, m.UpdatedAt}
}

// SelectLatest 查询单个节点实时数据
func (d *nodeMetricsDialect) SelectLatest(clusterID, nodeName string) (string, []any) {
	return `SELECT id, cluster_id, node_name, snapshot_json, cpu_usage, memory_usage, disk_usage, cpu_temp, updated_at
		FROM node_metrics_latest
		WHERE cluster_id = ? AND node_name = ?`, []any{clusterID, nodeName}
}

// SelectAllLatest 查询集群所有节点实时数据
func (d *nodeMetricsDialect) SelectAllLatest(clusterID string) (string, []any) {
	return `SELECT id, cluster_id, node_name, snapshot_json, cpu_usage, memory_usage, disk_usage, cpu_temp, updated_at
		FROM node_metrics_latest
		WHERE cluster_id = ?
		ORDER BY node_name`, []any{clusterID}
}

// DeleteLatest 删除节点实时数据
func (d *nodeMetricsDialect) DeleteLatest(clusterID, nodeName string) (string, []any) {
	return `DELETE FROM node_metrics_latest WHERE cluster_id = ? AND node_name = ?`, []any{clusterID, nodeName}
}

// ScanLatest 扫描实时数据行
func (d *nodeMetricsDialect) ScanLatest(rows *sql.Rows) (*database.NodeMetricsLatest, error) {
	m := &database.NodeMetricsLatest{}
	var updatedAt string
	err := rows.Scan(&m.ID, &m.ClusterID, &m.NodeName, &m.SnapshotJSON,
		&m.CPUUsage, &m.MemoryUsage, &m.DiskUsage, &m.CPUTemp, &updatedAt)
	if err != nil {
		return nil, err
	}
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return m, nil
}

// InsertHistory 插入历史数据
func (d *nodeMetricsDialect) InsertHistory(m *database.NodeMetricsHistory) (string, []any) {
	return `INSERT INTO node_metrics_history
		(cluster_id, node_name, timestamp, cpu_usage, memory_usage, disk_usage,
		 disk_io_read, disk_io_write, network_rx, network_tx, cpu_temp, load_1)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		[]any{m.ClusterID, m.NodeName, m.Timestamp, m.CPUUsage, m.MemoryUsage, m.DiskUsage,
			m.DiskIORead, m.DiskIOWrite, m.NetworkRx, m.NetworkTx, m.CPUTemp, m.Load1}
}

// SelectHistory 查询历史数据
func (d *nodeMetricsDialect) SelectHistory(clusterID, nodeName string, start, end time.Time) (string, []any) {
	return `SELECT id, cluster_id, node_name, timestamp, cpu_usage, memory_usage, disk_usage,
		disk_io_read, disk_io_write, network_rx, network_tx, cpu_temp, load_1
		FROM node_metrics_history
		WHERE cluster_id = ? AND node_name = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC`, []any{clusterID, nodeName, start, end}
}

// DeleteHistoryBefore 清理历史数据
func (d *nodeMetricsDialect) DeleteHistoryBefore(before time.Time) (string, []any) {
	return `DELETE FROM node_metrics_history WHERE timestamp < ?`, []any{before}
}

// ScanHistory 扫描历史数据行
func (d *nodeMetricsDialect) ScanHistory(rows *sql.Rows) (*database.NodeMetricsHistory, error) {
	m := &database.NodeMetricsHistory{}
	var timestamp string
	err := rows.Scan(&m.ID, &m.ClusterID, &m.NodeName, &timestamp,
		&m.CPUUsage, &m.MemoryUsage, &m.DiskUsage,
		&m.DiskIORead, &m.DiskIOWrite, &m.NetworkRx, &m.NetworkTx, &m.CPUTemp, &m.Load1)
	if err != nil {
		return nil, err
	}
	// SQLite 存储的时间格式: "2006-01-02 15:04:05.999999999+09:00"
	m.Timestamp, _ = time.Parse("2006-01-02 15:04:05.999999999-07:00", timestamp)
	if m.Timestamp.IsZero() {
		m.Timestamp, _ = time.Parse(time.RFC3339Nano, timestamp)
	}
	return m, nil
}

// SelectAllNodeNames 查询所有节点名
func (d *nodeMetricsDialect) SelectAllNodeNames(clusterID string) (string, []any) {
	return `SELECT DISTINCT node_name FROM node_metrics_latest WHERE cluster_id = ? ORDER BY node_name`, []any{clusterID}
}

// CreateLatestTable 创建实时数据表 DDL
func (d *nodeMetricsDialect) CreateLatestTable() string {
	return `CREATE TABLE IF NOT EXISTS node_metrics_latest (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cluster_id TEXT NOT NULL,
		node_name TEXT NOT NULL,
		snapshot_json TEXT NOT NULL,
		cpu_usage REAL DEFAULT 0,
		memory_usage REAL DEFAULT 0,
		disk_usage REAL DEFAULT 0,
		cpu_temp REAL DEFAULT 0,
		updated_at DATETIME NOT NULL,
		UNIQUE(cluster_id, node_name)
	)`
}

// CreateHistoryTable 创建历史数据表 DDL
func (d *nodeMetricsDialect) CreateHistoryTable() string {
	return `CREATE TABLE IF NOT EXISTS node_metrics_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cluster_id TEXT NOT NULL,
		node_name TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		cpu_usage REAL DEFAULT 0,
		memory_usage REAL DEFAULT 0,
		disk_usage REAL DEFAULT 0,
		disk_io_read REAL DEFAULT 0,
		disk_io_write REAL DEFAULT 0,
		network_rx REAL DEFAULT 0,
		network_tx REAL DEFAULT 0,
		cpu_temp REAL DEFAULT 0,
		load_1 REAL DEFAULT 0
	)`
}

// CreateHistoryIndexes 创建历史数据索引 DDL
func (d *nodeMetricsDialect) CreateHistoryIndexes() []string {
	return []string{
		`CREATE INDEX IF NOT EXISTS idx_node_metrics_history_cluster_node ON node_metrics_history(cluster_id, node_name)`,
		`CREATE INDEX IF NOT EXISTS idx_node_metrics_history_timestamp ON node_metrics_history(timestamp)`,
	}
}

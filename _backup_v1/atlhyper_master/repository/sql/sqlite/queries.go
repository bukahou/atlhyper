// repository/sql/sqlite/queries.go
//
// SQLite 专用 SQL 语句
//
// 本文件包含所有针对 SQLite 数据库优化的 SQL 语句。
// SQLite 是默认的嵌入式数据库，适用于单节点部署场景。
//
// SQLite 特有语法说明:
//   - 占位符: 使用 ? 作为参数占位符 (而非 PostgreSQL 的 $1, $2)
//   - UPSERT: 使用 ON CONFLICT(...) DO UPDATE SET col=excluded.col 语法
//   - 自增ID: 通过 LastInsertId() 获取 (而非 RETURNING id)
//   - 布尔值: 存储为 INTEGER (0/1)
//   - 时间戳: 存储为 TEXT (RFC3339 格式)
//
// 表结构概览:
//   - users: 用户信息表
//   - user_audit_logs: 用户操作审计日志
//   - event_logs: Kubernetes 集群事件日志
//   - notify_slack: Slack 通知配置 (单行表)
//   - notify_mail: 邮件通知配置 (单行表)
//   - node_metrics_flat: 节点指标数据 (扁平化存储)
//   - node_top_processes: 节点 TOP 进程信息
package sqlite

import "AtlHyper/atlhyper_master/repository/sql/schema"

// Queries 是 SQLite 数据库的 SQL 语句集合
// 通过 sql.SetQueries(sqlite.Queries) 激活
var Queries = &schema.Queries{

	// ========================================================================
	// User 用户表相关查询
	// 表结构: id INTEGER PRIMARY KEY, username TEXT, password_hash TEXT,
	//         display_name TEXT, email TEXT, role INTEGER, created_at TEXT, last_login TEXT
	// ========================================================================
	User: schema.UserQueries{
		// GetByID 根据用户ID查询
		// 用于获取用户详情、权限校验等场景
		GetByID: `
			SELECT id, username, password_hash, display_name, email, role, created_at, last_login
			FROM users WHERE id = ?`,

		// GetByUsername 根据用户名查询
		// 主要用于登录验证，通过用户名获取密码哈希进行比对
		GetByUsername: `
			SELECT id, username, password_hash, display_name, email, role, created_at, last_login
			FROM users WHERE username = ?`,

		// GetAll 获取所有用户列表
		// 用于管理界面展示用户列表
		GetAll: `
			SELECT id, username, password_hash, display_name, email, role, created_at, last_login
			FROM users ORDER BY id ASC`,

		// ExistsByUsername 检查用户名是否存在
		// 用于注册时防止用户名重复
		ExistsByUsername: `SELECT COUNT(*) FROM users WHERE username = ?`,

		// ExistsByID 检查用户ID是否存在
		// 用于验证用户是否有效
		ExistsByID: `SELECT COUNT(*) FROM users WHERE id = ?`,

		// Count 统计用户总数
		// 用于判断是否需要创建初始管理员
		Count: `SELECT COUNT(*) FROM users`,

		// Insert 创建新用户
		// 注意: SQLite 使用 LastInsertId() 获取新记录ID
		// 参数顺序: username, password_hash, display_name, email, role, created_at
		Insert: `
			INSERT INTO users (username, password_hash, display_name, email, role, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,

		// UpdateRole 更新用户角色
		// role 取值: 1=管理员, 2=普通用户
		UpdateRole: `UPDATE users SET role = ? WHERE id = ?`,

		// UpdateLastLogin 更新最后登录时间
		// 每次成功登录后调用
		UpdateLastLogin: `UPDATE users SET last_login = ? WHERE id = ?`,

		// Delete 删除用户
		// 物理删除，谨慎使用
		Delete: `DELETE FROM users WHERE id = ?`,
	},

	// ========================================================================
	// Audit 审计日志表相关查询
	// 表结构: id INTEGER PRIMARY KEY, timestamp TEXT, user_id INTEGER,
	//         username TEXT, role INTEGER, action TEXT, success INTEGER,
	//         ip TEXT, method TEXT, status INTEGER
	// ========================================================================
	Audit: schema.AuditQueries{
		// Insert 插入审计日志
		// 记录用户的每一次操作，用于安全审计和问题追溯
		// success 字段: 0=失败, 1=成功
		Insert: `
			INSERT INTO user_audit_logs
				(timestamp, user_id, username, role, action, success, ip, method, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,

		// GetAll 获取所有审计日志
		// 按时间倒序排列，最新的日志在前
		GetAll: `
			SELECT id, user_id, username, role, action, success, ip, method, status, timestamp
			FROM user_audit_logs
			ORDER BY timestamp DESC`,

		// GetByUserID 获取指定用户的审计日志
		// 用于查看单个用户的操作历史
		// 第二个参数为返回条数限制
		GetByUserID: `
			SELECT id, user_id, username, role, action, success, ip, method, status, timestamp
			FROM user_audit_logs
			WHERE user_id = ?
			ORDER BY timestamp DESC
			LIMIT ?`,
	},

	// ========================================================================
	// Event 事件日志表相关查询
	// 表结构: cluster_id TEXT, category TEXT, eventTime TEXT, kind TEXT,
	//         message TEXT, name TEXT, namespace TEXT, node TEXT,
	//         reason TEXT, severity TEXT, time TEXT
	// ========================================================================
	Event: schema.EventQueries{
		// Insert 插入事件日志
		// 存储从 Kubernetes 集群收集的事件
		Insert: `
			INSERT INTO event_logs (cluster_id, category, eventTime, kind, message, name, namespace, node, reason, severity, time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,

		// GetSince 查询指定时间之后的事件
		// 用于前端轮询获取最新事件
		// 参数: cluster_id, 起始时间
		GetSince: `
			SELECT cluster_id, category, eventTime, kind, message, name, namespace, node, reason, severity, time
			FROM event_logs
			WHERE cluster_id = ? AND eventTime >= ?
			ORDER BY eventTime DESC`,
	},

	// ========================================================================
	// Config 配置表相关查询
	// 包含 notify_slack 和 notify_mail 两个单行配置表
	// ========================================================================
	Config: schema.ConfigQueries{
		// -------------------- Slack 通知配置 --------------------
		// 表结构: id INTEGER, enable INTEGER, webhook TEXT,
		//         interval_sec INTEGER, updated_at TEXT

		// GetSlack 获取 Slack 配置
		// 固定查询 id=1 的记录
		GetSlack: `
			SELECT id, enable, webhook, interval_sec, updated_at
			FROM notify_slack WHERE id = 1`,

		// CountSlack 检查配置是否存在
		// 用于判断是执行 INSERT 还是 UPDATE
		CountSlack: `SELECT COUNT(*) FROM notify_slack WHERE id = 1`,

		// InsertSlack 插入 Slack 配置
		// 首次配置时使用
		InsertSlack: `
			INSERT INTO notify_slack (id, enable, webhook, interval_sec, updated_at)
			VALUES (1, ?, ?, ?, ?)`,

		// UpdateSlack 更新 Slack 配置
		// 修改现有配置时使用
		UpdateSlack: `
			UPDATE notify_slack SET enable=?, webhook=?, interval_sec=?, updated_at=?
			WHERE id=1`,

		// -------------------- 邮件通知配置 --------------------
		// 表结构: id INTEGER, enable INTEGER, smtp_host TEXT, smtp_port TEXT,
		//         username TEXT, password TEXT, from_addr TEXT, to_addrs TEXT,
		//         interval_sec INTEGER, updated_at TEXT

		// GetMail 获取邮件配置
		GetMail: `
			SELECT id, enable, smtp_host, smtp_port, username, password, from_addr, to_addrs, interval_sec, updated_at
			FROM notify_mail WHERE id = 1`,

		// CountMail 检查配置是否存在
		CountMail: `SELECT COUNT(*) FROM notify_mail WHERE id = 1`,

		// InsertMail 插入邮件配置
		InsertMail: `
			INSERT INTO notify_mail (id, enable, smtp_host, smtp_port, username, password, from_addr, to_addrs, interval_sec, updated_at)
			VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,

		// UpdateMail 更新邮件配置
		UpdateMail: `
			UPDATE notify_mail SET enable=?, smtp_host=?, smtp_port=?, username=?, password=?, from_addr=?, to_addrs=?, interval_sec=?, updated_at=?
			WHERE id=1`,
	},

	// ========================================================================
	// Metrics 节点指标表相关查询
	// 包含 node_metrics_flat (指标) 和 node_top_processes (进程)
	// ========================================================================
	Metrics: schema.MetricsQueries{
		// UpsertNodeMetrics 插入或更新节点指标
		// 使用 SQLite 的 ON CONFLICT...DO UPDATE 语法实现 UPSERT
		// 主键: (node_name, ts)
		// 当主键冲突时更新所有指标字段
		//
		// 字段分组:
		//   - CPU: usage, cores, load1, load5, load15
		//   - 内存: total, used, available, usage
		//   - 温度: cpu, gpu, nvme
		//   - 磁盘: total, used, free, usage
		//   - 网络: lo_rx, lo_tx, eth0_rx, eth0_tx
		UpsertNodeMetrics: `
			INSERT INTO node_metrics_flat
			(node_name, ts,
			 cpu_usage, cpu_cores, cpu_load1, cpu_load5, cpu_load15,
			 memory_total, memory_used, memory_available, memory_usage,
			 temp_cpu, temp_gpu, temp_nvme,
			 disk_total, disk_used, disk_free, disk_usage,
			 net_lo_rx_kbps, net_lo_tx_kbps, net_eth0_rx_kbps, net_eth0_tx_kbps)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(node_name, ts) DO UPDATE SET
			 cpu_usage=excluded.cpu_usage, cpu_cores=excluded.cpu_cores,
			 cpu_load1=excluded.cpu_load1, cpu_load5=excluded.cpu_load5, cpu_load15=excluded.cpu_load15,
			 memory_total=excluded.memory_total, memory_used=excluded.memory_used,
			 memory_available=excluded.memory_available, memory_usage=excluded.memory_usage,
			 temp_cpu=excluded.temp_cpu, temp_gpu=excluded.temp_gpu, temp_nvme=excluded.temp_nvme,
			 disk_total=excluded.disk_total, disk_used=excluded.disk_used, disk_free=excluded.disk_free, disk_usage=excluded.disk_usage,
			 net_lo_rx_kbps=excluded.net_lo_rx_kbps, net_lo_tx_kbps=excluded.net_lo_tx_kbps,
			 net_eth0_rx_kbps=excluded.net_eth0_rx_kbps, net_eth0_tx_kbps=excluded.net_eth0_tx_kbps`,

		// UpsertTopProcesses 插入或更新 TOP 进程
		// 主键: (node_name, ts, pid)
		// 记录每个时间点各节点的高资源占用进程
		UpsertTopProcesses: `
			INSERT INTO node_top_processes
			(node_name, ts, pid, user, command, cpu_percent, memory_mb)
			VALUES (?,?,?,?,?,?,?)
			ON CONFLICT(node_name, ts, pid) DO UPDATE SET
			 user=excluded.user, command=excluded.command,
			 cpu_percent=excluded.cpu_percent, memory_mb=excluded.memory_mb`,

		// DeleteMetrics 清理过期指标数据
		// 删除时间戳早于指定时间的记录
		// 用于定期清理历史数据，防止数据库过大
		DeleteMetrics: `DELETE FROM node_metrics_flat WHERE ts < ?`,

		// DeleteProcesses 清理过期进程数据
		// 与 DeleteMetrics 配合使用
		DeleteProcesses: `DELETE FROM node_top_processes WHERE ts < ?`,

		// GetLatestByNode 获取节点最新指标
		// 按时间倒序取第一条
		// 用于实时监控展示
		GetLatestByNode: `
			SELECT id, node_name, ts,
				cpu_usage, cpu_cores, cpu_load1, cpu_load5, cpu_load15,
				memory_total, memory_used, memory_available, memory_usage,
				temp_cpu, temp_gpu, temp_nvme,
				disk_total, disk_used, disk_free, disk_usage,
				net_lo_rx_kbps, net_lo_tx_kbps, net_eth0_rx_kbps, net_eth0_tx_kbps
			FROM node_metrics_flat
			WHERE node_name = ?
			ORDER BY ts DESC
			LIMIT 1`,
	},
}

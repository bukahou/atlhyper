// repository/sql/pg/queries.go
//
// PostgreSQL 专用 SQL 语句
//
// 本文件包含所有针对 PostgreSQL 数据库优化的 SQL 语句。
// PostgreSQL 适用于需要高并发、高可用的生产环境部署。
//
// PostgreSQL 特有语法说明:
//   - 占位符: 使用 $1, $2, $3... 作为参数占位符 (而非 SQLite 的 ?)
//   - UPSERT: 使用 ON CONFLICT(...) DO UPDATE SET col=excluded.col 语法 (与 SQLite 相同)
//   - 自增ID: 使用 RETURNING id 在 INSERT 后返回新记录ID
//   - 布尔值: 原生支持 BOOLEAN 类型，但为兼容性仍使用 INTEGER (0/1)
//   - 时间戳: 可使用 TIMESTAMP WITH TIME ZONE，但为兼容性使用 TEXT (RFC3339)
//
// 与 SQLite 的主要差异:
//   - 占位符格式不同: PostgreSQL 使用 $1, $2 而非 ?
//   - INSERT 返回ID方式不同: PostgreSQL 使用 RETURNING id
//   - 某些数据类型更丰富 (JSONB, ARRAY 等，本项目暂未使用)
//
// 表结构与 SQLite 版本完全相同，仅 SQL 语法有差异
package pg

import "AtlHyper/atlhyper_master/repository/sql/schema"

// Queries 是 PostgreSQL 数据库的 SQL 语句集合
// 通过 sql.SetQueries(pg.Queries) 激活
var Queries = &schema.Queries{

	// ========================================================================
	// User 用户表相关查询
	// 表结构: id SERIAL PRIMARY KEY, username VARCHAR, password_hash VARCHAR,
	//         display_name VARCHAR, email VARCHAR, role INTEGER,
	//         created_at TIMESTAMP, last_login TIMESTAMP
	// ========================================================================
	User: schema.UserQueries{
		// GetByID 根据用户ID查询
		// 参数: $1=用户ID
		GetByID: `
			SELECT id, username, password_hash, display_name, email, role, created_at, last_login
			FROM users WHERE id = $1`,

		// GetByUsername 根据用户名查询
		// 参数: $1=用户名
		// 主要用于登录验证
		GetByUsername: `
			SELECT id, username, password_hash, display_name, email, role, created_at, last_login
			FROM users WHERE username = $1`,

		// GetAll 获取所有用户列表
		// 无参数，按ID升序排列
		GetAll: `
			SELECT id, username, password_hash, display_name, email, role, created_at, last_login
			FROM users ORDER BY id ASC`,

		// ExistsByUsername 检查用户名是否存在
		// 参数: $1=用户名
		ExistsByUsername: `SELECT COUNT(*) FROM users WHERE username = $1`,

		// ExistsByID 检查用户ID是否存在
		// 参数: $1=用户ID
		ExistsByID: `SELECT COUNT(*) FROM users WHERE id = $1`,

		// Count 统计用户总数
		Count: `SELECT COUNT(*) FROM users`,

		// Insert 创建新用户
		// 参数: $1=username, $2=password_hash, $3=display_name, $4=email, $5=role, $6=created_at
		// 返回: 使用 RETURNING id 返回新创建用户的ID
		// 注意: 调用方需要使用 QueryRowContext 而非 ExecContext 来获取返回的ID
		Insert: `
			INSERT INTO users (username, password_hash, display_name, email, role, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`,

		// UpdateRole 更新用户角色
		// 参数: $1=新角色, $2=用户ID
		UpdateRole: `UPDATE users SET role = $1 WHERE id = $2`,

		// UpdateLastLogin 更新最后登录时间
		// 参数: $1=登录时间, $2=用户ID
		UpdateLastLogin: `UPDATE users SET last_login = $1 WHERE id = $2`,

		// Delete 删除用户
		// 参数: $1=用户ID
		Delete: `DELETE FROM users WHERE id = $1`,
	},

	// ========================================================================
	// Audit 审计日志表相关查询
	// 表结构与 SQLite 相同
	// ========================================================================
	Audit: schema.AuditQueries{
		// Insert 插入审计日志
		// 参数: $1=timestamp, $2=user_id, $3=username, $4=role, $5=action,
		//       $6=success, $7=ip, $8=method, $9=status
		Insert: `
			INSERT INTO user_audit_logs
				(timestamp, user_id, username, role, action, success, ip, method, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,

		// GetAll 获取所有审计日志
		// 按时间倒序排列
		GetAll: `
			SELECT id, user_id, username, role, action, success, ip, method, status, timestamp
			FROM user_audit_logs
			ORDER BY timestamp DESC`,

		// GetByUserID 获取指定用户的审计日志
		// 参数: $1=用户ID, $2=限制条数
		GetByUserID: `
			SELECT id, user_id, username, role, action, success, ip, method, status, timestamp
			FROM user_audit_logs
			WHERE user_id = $1
			ORDER BY timestamp DESC
			LIMIT $2`,
	},

	// ========================================================================
	// Event 事件日志表相关查询
	// ========================================================================
	Event: schema.EventQueries{
		// Insert 插入事件日志
		// 参数: $1=cluster_id, $2=category, $3=eventTime, $4=kind, $5=message,
		//       $6=name, $7=namespace, $8=node, $9=reason, $10=severity, $11=time
		Insert: `
			INSERT INTO event_logs (cluster_id, category, eventTime, kind, message, name, namespace, node, reason, severity, time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,

		// GetSince 查询指定时间之后的事件
		// 参数: $1=cluster_id, $2=起始时间
		GetSince: `
			SELECT cluster_id, category, eventTime, kind, message, name, namespace, node, reason, severity, time
			FROM event_logs
			WHERE cluster_id = $1 AND eventTime >= $2
			ORDER BY eventTime DESC`,
	},

	// ========================================================================
	// Config 配置表相关查询
	// ========================================================================
	Config: schema.ConfigQueries{
		// -------------------- Slack 通知配置 --------------------

		// GetSlack 获取 Slack 配置
		GetSlack: `
			SELECT id, enable, webhook, interval_sec, updated_at
			FROM notify_slack WHERE id = 1`,

		// CountSlack 检查配置是否存在
		CountSlack: `SELECT COUNT(*) FROM notify_slack WHERE id = 1`,

		// InsertSlack 插入 Slack 配置
		// 参数: $1=enable, $2=webhook, $3=interval_sec, $4=updated_at
		InsertSlack: `
			INSERT INTO notify_slack (id, enable, webhook, interval_sec, updated_at)
			VALUES (1, $1, $2, $3, $4)`,

		// UpdateSlack 更新 Slack 配置
		// 参数: $1=enable, $2=webhook, $3=interval_sec, $4=updated_at
		UpdateSlack: `
			UPDATE notify_slack SET enable=$1, webhook=$2, interval_sec=$3, updated_at=$4
			WHERE id=1`,

		// -------------------- 邮件通知配置 --------------------

		// GetMail 获取邮件配置
		GetMail: `
			SELECT id, enable, smtp_host, smtp_port, username, password, from_addr, to_addrs, interval_sec, updated_at
			FROM notify_mail WHERE id = 1`,

		// CountMail 检查配置是否存在
		CountMail: `SELECT COUNT(*) FROM notify_mail WHERE id = 1`,

		// InsertMail 插入邮件配置
		// 参数: $1=enable, $2=smtp_host, $3=smtp_port, $4=username, $5=password,
		//       $6=from_addr, $7=to_addrs, $8=interval_sec, $9=updated_at
		InsertMail: `
			INSERT INTO notify_mail (id, enable, smtp_host, smtp_port, username, password, from_addr, to_addrs, interval_sec, updated_at)
			VALUES (1, $1, $2, $3, $4, $5, $6, $7, $8, $9)`,

		// UpdateMail 更新邮件配置
		// 参数: $1=enable, $2=smtp_host, $3=smtp_port, $4=username, $5=password,
		//       $6=from_addr, $7=to_addrs, $8=interval_sec, $9=updated_at
		UpdateMail: `
			UPDATE notify_mail SET enable=$1, smtp_host=$2, smtp_port=$3, username=$4, password=$5, from_addr=$6, to_addrs=$7, interval_sec=$8, updated_at=$9
			WHERE id=1`,
	},

	// ========================================================================
	// Metrics 节点指标表相关查询
	// ========================================================================
	Metrics: schema.MetricsQueries{
		// UpsertNodeMetrics 插入或更新节点指标
		// 使用 PostgreSQL 的 ON CONFLICT...DO UPDATE 语法
		// 主键: (node_name, ts)
		//
		// 参数:
		//   $1=node_name, $2=ts
		//   $3-$7: CPU (usage, cores, load1, load5, load15)
		//   $8-$11: Memory (total, used, available, usage)
		//   $12-$14: Temperature (cpu, gpu, nvme)
		//   $15-$18: Disk (total, used, free, usage)
		//   $19-$22: Network (lo_rx, lo_tx, eth0_rx, eth0_tx)
		UpsertNodeMetrics: `
			INSERT INTO node_metrics_flat
			(node_name, ts,
			 cpu_usage, cpu_cores, cpu_load1, cpu_load5, cpu_load15,
			 memory_total, memory_used, memory_available, memory_usage,
			 temp_cpu, temp_gpu, temp_nvme,
			 disk_total, disk_used, disk_free, disk_usage,
			 net_lo_rx_kbps, net_lo_tx_kbps, net_eth0_rx_kbps, net_eth0_tx_kbps)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
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
		// 参数: $1=node_name, $2=ts, $3=pid, $4=user, $5=command, $6=cpu_percent, $7=memory_mb
		// 主键: (node_name, ts, pid)
		UpsertTopProcesses: `
			INSERT INTO node_top_processes
			(node_name, ts, pid, user, command, cpu_percent, memory_mb)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT(node_name, ts, pid) DO UPDATE SET
			 user=excluded.user, command=excluded.command,
			 cpu_percent=excluded.cpu_percent, memory_mb=excluded.memory_mb`,

		// DeleteMetrics 清理过期指标数据
		// 参数: $1=截止时间
		DeleteMetrics: `DELETE FROM node_metrics_flat WHERE ts < $1`,

		// DeleteProcesses 清理过期进程数据
		// 参数: $1=截止时间
		DeleteProcesses: `DELETE FROM node_top_processes WHERE ts < $1`,

		// GetLatestByNode 获取节点最新指标
		// 参数: $1=node_name
		GetLatestByNode: `
			SELECT id, node_name, ts,
				cpu_usage, cpu_cores, cpu_load1, cpu_load5, cpu_load15,
				memory_total, memory_used, memory_available, memory_usage,
				temp_cpu, temp_gpu, temp_nvme,
				disk_total, disk_used, disk_free, disk_usage,
				net_lo_rx_kbps, net_lo_tx_kbps, net_eth0_rx_kbps, net_eth0_tx_kbps
			FROM node_metrics_flat
			WHERE node_name = $1
			ORDER BY ts DESC
			LIMIT 1`,
	},
}

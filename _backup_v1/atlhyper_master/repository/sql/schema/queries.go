// repository/sql/schema/queries.go
//
// SQL 查询语句结构定义
//
// 本文件定义了所有 SQL 查询语句的类型结构，作为独立包存在以避免循环导入。
// 具体的 SQL 语句实现位于 sqlite/ 和 pg/ 子目录中。
//
// 架构说明:
//   schema/     - 查询结构定义 (本文件，无外部依赖)
//   sqlite/     - SQLite 专用 SQL 语句 (使用 ? 占位符)
//   pg/         - PostgreSQL 专用 SQL 语句 (使用 $1, $2 占位符)
//   *.go        - 业务逻辑层 (通过全局变量 Q 访问 SQL 语句)
//
// 使用方式:
//   1. 业务逻辑代码通过 sql.Q.User.GetByID 等方式访问 SQL 语句
//   2. 初始化时根据数据库类型加载对应的 Queries 实例
//   3. 新增 SQL 语句时，需要:
//      a. 在对应的 XxxQueries 结构中添加字段
//      b. 在 sqlite/queries.go 中添加 SQLite 版本
//      c. 在 pg/queries.go 中添加 PostgreSQL 版本
package schema

// ============================================================================
// Queries 所有 SQL 语句的聚合结构
// ============================================================================

// Queries 是所有 SQL 语句的顶层容器
// 包含各个业务领域的查询语句集合
type Queries struct {
	User    UserQueries    // 用户相关 SQL
	Audit   AuditQueries   // 审计日志相关 SQL
	Event   EventQueries   // 事件日志相关 SQL
	Config  ConfigQueries  // 系统配置相关 SQL
	Metrics MetricsQueries // 节点指标相关 SQL
}

// ============================================================================
// UserQueries 用户管理相关 SQL
// ============================================================================

// UserQueries 用户表 (users) 的所有查询语句
// 对应表结构: id, username, password_hash, display_name, email, role, created_at, last_login
type UserQueries struct {
	// GetByID 根据用户 ID 查询单个用户
	// 参数: $1=用户ID
	// 返回: 用户完整信息
	GetByID string

	// GetByUsername 根据用户名查询单个用户
	// 参数: $1=用户名
	// 返回: 用户完整信息
	// 用于登录验证
	GetByUsername string

	// GetAll 查询所有用户列表
	// 参数: 无
	// 返回: 按 ID 升序排列的用户列表
	GetAll string

	// ExistsByUsername 检查用户名是否已存在
	// 参数: $1=用户名
	// 返回: COUNT(*) 结果 (0 或 1)
	// 用于注册时的用户名唯一性校验
	ExistsByUsername string

	// ExistsByID 检查用户 ID 是否存在
	// 参数: $1=用户ID
	// 返回: COUNT(*) 结果 (0 或 1)
	ExistsByID string

	// Count 统计用户总数
	// 参数: 无
	// 返回: COUNT(*) 结果
	// 用于判断是否需要初始化管理员账户
	Count string

	// Insert 插入新用户
	// 参数: $1=username, $2=password_hash, $3=display_name, $4=email, $5=role, $6=created_at
	// 返回: SQLite 使用 LastInsertId(), PostgreSQL 使用 RETURNING id
	Insert string

	// UpdateRole 更新用户角色
	// 参数: $1=新角色, $2=用户ID
	// 角色定义: 1=管理员, 2=普通用户
	UpdateRole string

	// UpdateLastLogin 更新用户最后登录时间
	// 参数: $1=登录时间(RFC3339格式), $2=用户ID
	UpdateLastLogin string

	// Delete 删除用户
	// 参数: $1=用户ID
	// 注意: 物理删除，不可恢复
	Delete string
}

// ============================================================================
// AuditQueries 审计日志相关 SQL
// ============================================================================

// AuditQueries 审计日志表 (user_audit_logs) 的所有查询语句
// 对应表结构: id, timestamp, user_id, username, role, action, success, ip, method, status
type AuditQueries struct {
	// Insert 插入审计日志
	// 参数: $1=timestamp, $2=user_id, $3=username, $4=role, $5=action,
	//       $6=success(0/1), $7=ip, $8=method, $9=status
	// 用于记录用户操作行为
	Insert string

	// GetAll 查询所有审计日志
	// 参数: 无
	// 返回: 按时间倒序排列的日志列表
	GetAll string

	// GetByUserID 查询指定用户的审计日志
	// 参数: $1=用户ID, $2=限制条数
	// 返回: 按时间倒序排列的日志列表
	GetByUserID string
}

// ============================================================================
// EventQueries 事件日志相关 SQL
// ============================================================================

// EventQueries 事件日志表 (event_logs) 的所有查询语句
// 对应表结构: cluster_id, category, eventTime, kind, message, name, namespace, node, reason, severity, time
// 用于存储 Kubernetes 集群事件
type EventQueries struct {
	// Insert 插入单条事件日志
	// 参数: $1=cluster_id, $2=category, $3=eventTime, $4=kind, $5=message,
	//       $6=name, $7=namespace, $8=node, $9=reason, $10=severity, $11=time
	Insert string

	// GetSince 查询指定时间之后的事件
	// 参数: $1=cluster_id, $2=起始时间
	// 返回: 按事件时间倒序排列的事件列表
	GetSince string
}

// ============================================================================
// ConfigQueries 系统配置相关 SQL
// ============================================================================

// ConfigQueries 配置表的所有查询语句
// 包含 Slack 通知配置 (notify_slack) 和邮件通知配置 (notify_mail)
// 两个表都使用 id=1 的单条记录模式
type ConfigQueries struct {
	// -------------------- Slack 配置 --------------------

	// GetSlack 获取 Slack 通知配置
	// 参数: 无 (固定查询 id=1)
	// 返回: id, enable, webhook, interval_sec, updated_at
	GetSlack string

	// CountSlack 检查 Slack 配置是否存在
	// 参数: 无 (固定查询 id=1)
	// 返回: COUNT(*) 结果 (0 或 1)
	// 用于判断是插入还是更新
	CountSlack string

	// InsertSlack 插入 Slack 配置
	// 参数: $1=enable(0/1), $2=webhook, $3=interval_sec, $4=updated_at
	// 注意: id 固定为 1
	InsertSlack string

	// UpdateSlack 更新 Slack 配置
	// 参数: $1=enable(0/1), $2=webhook, $3=interval_sec, $4=updated_at
	UpdateSlack string

	// -------------------- 邮件配置 --------------------

	// GetMail 获取邮件通知配置
	// 参数: 无 (固定查询 id=1)
	// 返回: id, enable, smtp_host, smtp_port, username, password, from_addr, to_addrs, interval_sec, updated_at
	GetMail string

	// CountMail 检查邮件配置是否存在
	// 参数: 无 (固定查询 id=1)
	// 返回: COUNT(*) 结果 (0 或 1)
	CountMail string

	// InsertMail 插入邮件配置
	// 参数: $1=enable, $2=smtp_host, $3=smtp_port, $4=username, $5=password,
	//       $6=from_addr, $7=to_addrs, $8=interval_sec, $9=updated_at
	InsertMail string

	// UpdateMail 更新邮件配置
	// 参数: $1=enable, $2=smtp_host, $3=smtp_port, $4=username, $5=password,
	//       $6=from_addr, $7=to_addrs, $8=interval_sec, $9=updated_at
	UpdateMail string
}

// ============================================================================
// MetricsQueries 节点指标相关 SQL
// ============================================================================

// MetricsQueries 节点指标表的所有查询语句
// 包含 node_metrics_flat (指标数据) 和 node_top_processes (进程数据)
type MetricsQueries struct {
	// UpsertNodeMetrics 插入或更新节点指标
	// 参数: $1=node_name, $2=ts(时间戳),
	//       $3-$7: CPU 相关 (usage, cores, load1, load5, load15)
	//       $8-$11: 内存相关 (total, used, available, usage)
	//       $12-$14: 温度相关 (cpu, gpu, nvme)
	//       $15-$18: 磁盘相关 (total, used, free, usage)
	//       $19-$22: 网络相关 (lo_rx, lo_tx, eth0_rx, eth0_tx)
	// 使用 UPSERT 语法，主键冲突时更新
	UpsertNodeMetrics string

	// UpsertTopProcesses 插入或更新 TOP 进程信息
	// 参数: $1=node_name, $2=ts, $3=pid, $4=user, $5=command, $6=cpu_percent, $7=memory_mb
	// 使用 UPSERT 语法，主键 (node_name, ts, pid) 冲突时更新
	UpsertTopProcesses string

	// DeleteMetrics 清理过期的指标数据
	// 参数: $1=截止时间
	// 删除时间戳早于截止时间的记录
	DeleteMetrics string

	// DeleteProcesses 清理过期的进程数据
	// 参数: $1=截止时间
	// 删除时间戳早于截止时间的记录
	DeleteProcesses string

	// GetLatestByNode 获取指定节点的最新指标
	// 参数: $1=node_name
	// 返回: 按时间倒序的第一条记录
	GetLatestByNode string
}

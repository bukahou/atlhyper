package sqlite

import (
	"NeuroController/db/utils"
	"log"
)

// ============================================================
// ✅ CreateTables：初始化 SQLite 表结构
// - 使用全局 utils.DB 数据库连接
// - 如表已存在则不会重复创建（IF NOT EXISTS）
// ============================================================
func CreateTables() error {

	// 1️⃣ 创建 event_logs 表（用于记录告警/事件日志）
	_, err := utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS event_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT,         -- 事件来源类别，如 kube-event / apm / custom
			eventTime TEXT,        -- 实际事件发生时间（ISO 8601 字符串）
			kind TEXT,             -- 资源类型，如 Pod / Node / Deployment
			message TEXT,          -- 告警/事件的详细消息
			name TEXT,             -- 对应的资源名称
			namespace TEXT,        -- 所属命名空间
			node TEXT,             -- 所属节点名称（可为空）
			reason TEXT,           -- 原因（如 K8s 事件中的 reason）
			severity TEXT,         -- 严重程度：info / warning / critical
			time TEXT              -- 入库时间戳（记录时间，区别于 eventTime）
		)
	`)
	if err != nil {
		log.Printf("❌ 创建 event_logs 表失败: %v", err)
		return err
	}

	// 2️⃣ 创建 users 表（用于 Web 登录认证）
	_, err = utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,      -- 用户名，唯一
			password_hash TEXT NOT NULL,        -- 加密后的密码（bcrypt）
			display_name TEXT,                  -- 展示用名称
			email TEXT,                         -- 邮箱地址（可选）
			role INTEGER NOT NULL,              -- 角色标识（如 1=普通用户，3=管理员）
			created_at TEXT NOT NULL,           -- 创建时间（ISO 字符串）
			last_login TEXT                     -- 最近一次登录时间（可为空）
		)
	`)
	if err != nil {
		log.Printf("❌ 创建 users 表失败: %v", err)
	}
	

	// 3️⃣ 创建用户审计表格。主要记录对集群进行的操作细节
	// username	TEXT	执行操作的用户名
	// role	INTEGER	用户角色（如 1=普通用户，2=运维，3=管理员）
	// action	TEXT	操作内容（如 "cordon_node", "delete_pod" 等）
	// success	BOOLEAN	操作是否成功（true / false）
	// timestamp	TEXT	操作发生时间（ISO8601 格式字符串）
	// _, err = utils.DB.Exec(`
	// 	CREATE TABLE IF NOT EXISTS user_audit_logs (
	// 		id INTEGER PRIMARY KEY AUTOINCREMENT,
	// 		user_id INTEGER NOT NULL,		   -- 操作用户 ID
	// 		username TEXT NOT NULL,            -- 执行操作的用户名
	// 		role INTEGER NOT NULL,             -- 用户角色（如 1=普通用户，2=运维，3=管理员）
	// 		action TEXT NOT NULL,              -- 操作内容（如 "cordon_node", "delete_pod" 等）
	// 		success BOOLEAN NOT NULL,          -- 操作是否成功（true / false）
	// 		timestamp TEXT NOT NULL DEFAULT  (datetime('now', 'localtime'))            -- 操作发生时间（ISO8601 格式字符串）
	// 	)
	// `)
	// if err != nil {
	// 	log.Printf("❌ 创建 user_audit_logs 表失败: %v", err)
	// 	return err
	// }


	_, err = utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_audit_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,                         -- 自增主键
			timestamp   TEXT    NOT NULL DEFAULT (datetime('now','localtime')),    -- 事件发生时间（本地时区）

			user_id     INTEGER NOT NULL,                                          -- 操作用户 ID（未登录写 0）
			username    TEXT    NOT NULL,                                          -- 用户名（未登录写 'anonymous'）
			role        INTEGER NOT NULL CHECK (role IN (1,2,3)),                  -- 用户角色（1=Viewer，2=Operator，3=Admin）

			action      TEXT    NOT NULL,                                          -- 动作：pod.restart / auth.login / ...
			success     INTEGER NOT NULL CHECK (success IN (0,1)),                 -- 是否成功（0=失败，1=成功）

			ip          TEXT,                                                       -- 客户端 IP（优先 X-Forwarded-For）
			method      TEXT,                                                       -- HTTP 方法（GET/POST/...）
			status      INTEGER                                                     -- HTTP 状态码
		);

		-- 常用检索索引（对齐当前字段集）
		CREATE INDEX IF NOT EXISTS idx_audit_time           ON user_audit_logs(timestamp DESC);               -- 按时间倒序
		CREATE INDEX IF NOT EXISTS idx_audit_user_time      ON user_audit_logs(username, timestamp DESC);     -- 按用户+时间
		CREATE INDEX IF NOT EXISTS idx_audit_action_time    ON user_audit_logs(action, timestamp DESC);       -- 按动作+时间
		CREATE INDEX IF NOT EXISTS idx_audit_role_time      ON user_audit_logs(role, timestamp DESC);         -- 按角色+时间
		CREATE INDEX IF NOT EXISTS idx_audit_status_time    ON user_audit_logs(status, timestamp DESC);       -- 按状态码+时间
		CREATE INDEX IF NOT EXISTS idx_audit_success_action ON user_audit_logs(success, action, timestamp DESC); -- 成功/失败分布
	`)
	if err != nil {
		log.Printf("❌ 创建/升级 user_audit_logs 表失败: %v", err)
		return err
	}




	// 4️⃣ 创建 node_metrics_flat 表（每个 节点+时间戳 一行的汇总）
	_, err = utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_metrics_flat (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_name        TEXT NOT NULL,     -- 节点名称
			ts               TEXT NOT NULL,     -- 采集时间（ISO8601 字符串）
			cpu_usage        REAL,              -- CPU 使用率（0.0~1.0）
			cpu_cores        INTEGER,           -- CPU 核心数
			cpu_load1        REAL,              -- 1 分钟平均负载
			cpu_load5        REAL,              -- 5 分钟平均负载
			cpu_load15       REAL,              -- 15 分钟平均负载
			memory_total     INTEGER,           -- 总内存（字节）
			memory_used      INTEGER,           -- 已用内存（字节）
			memory_available INTEGER,           -- 可用内存（字节）
			memory_usage     REAL,              -- 内存使用率（0.0~1.0）
			temp_cpu         INTEGER,           -- CPU 温度（摄氏度）
			temp_gpu         INTEGER,           -- GPU 温度（摄氏度）
			temp_nvme        INTEGER,           -- NVMe 磁盘温度（摄氏度）
			disk_total       INTEGER,           -- 磁盘总容量（字节）
			disk_used        INTEGER,           -- 已用磁盘容量（字节）
			disk_free        INTEGER,           -- 可用磁盘容量（字节）
			disk_usage       REAL,              -- 磁盘使用率（0.0~1.0）
			net_lo_rx_kbps   REAL,              -- lo 网卡接收速率（KB/s）
			net_lo_tx_kbps   REAL,              -- lo 网卡发送速率（KB/s）
			net_eth0_rx_kbps REAL,              -- eth0 网卡接收速率（KB/s）
			net_eth0_tx_kbps REAL,              -- eth0 网卡发送速率（KB/s）
			UNIQUE(node_name, ts)
		);
		CREATE INDEX IF NOT EXISTS idx_node_metrics_flat_ts
			ON node_metrics_flat(ts DESC);
		CREATE INDEX IF NOT EXISTS idx_node_metrics_flat_node_ts
			ON node_metrics_flat(node_name, ts DESC);
	`)
	if err != nil {
		log.Printf("❌ 创建 node_metrics_flat 表失败: %v", err)
		return err
	}

	// 5️⃣ 创建 node_top_processes 表（每个 节点+时间戳+PID 一行）
	_, err = utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS node_top_processes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_name   TEXT NOT NULL,          -- 节点名称
			ts          TEXT NOT NULL,          -- 采集时间（ISO8601 字符串）
			pid         INTEGER NOT NULL,       -- 进程 ID
			user        TEXT,                   -- 所属用户
			command     TEXT,                   -- 命令名
			cpu_percent REAL,                   -- CPU 占用百分比
			memory_mb   REAL,                   -- 内存占用（MB）
			UNIQUE(node_name, ts, pid)
		);
		CREATE INDEX IF NOT EXISTS idx_node_top_processes_ts
			ON node_top_processes(ts DESC);
		CREATE INDEX IF NOT EXISTS idx_node_topproc_node_ts
			ON node_top_processes(node_name, ts DESC);
	`)
	if err != nil {
		log.Printf("❌ 创建 node_top_processes 表失败: %v", err)
		return err
	}


	return nil
}

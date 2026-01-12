// store/sqlite/init.go
// SQLite 存储引擎初始化
package sqlite

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DefaultDBPath 默认数据库路径
const DefaultDBPath = "atlhyper_master/store/sqlite/data/atlhyper.db"

// Init 初始化 SQLite 数据库，返回 DB 连接
func Init(dbPath string) (*sql.DB, error) {
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// 打开连接
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// 创建表结构
	if err = createTables(db); err != nil {
		return nil, err
	}

	log.Println("✅ SQLite 存储引擎初始化完成")
	return db, nil
}

// ============================================================
// 表结构创建
// ============================================================

func createTables(db *sql.DB) error {
	tables := []struct {
		name   string
		create func(*sql.DB) error
	}{
		{"event_logs", createEventLogsTable},
		{"users", createUsersTable},
		{"user_audit_logs", createUserAuditLogsTable},
		{"node_metrics_flat", createNodeMetricsTable},
		{"node_top_processes", createTopProcessesTable},
		{"notify_slack", createNotifySlackTable},
		{"notify_mail", createNotifyMailTable},
	}

	for _, t := range tables {
		if err := t.create(db); err != nil {
			log.Printf("❌ 创建 %s 表失败: %v", t.name, err)
			return err
		}
	}
	return nil
}

func createEventLogsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS event_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT,
			category TEXT,
			eventTime TEXT,
			kind TEXT,
			message TEXT,
			name TEXT,
			namespace TEXT,
			node TEXT,
			reason TEXT,
			severity TEXT,
			time TEXT
		)
	`)
	return err
}

func createUsersTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			email TEXT,
			role INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			last_login TEXT
		)
	`)
	return err
}

func createUserAuditLogsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_audit_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp   TEXT    NOT NULL DEFAULT (datetime('now','localtime')),
			user_id     INTEGER NOT NULL,
			username    TEXT    NOT NULL,
			role        INTEGER NOT NULL CHECK (role IN (1,2,3)),
			action      TEXT    NOT NULL,
			success     INTEGER NOT NULL CHECK (success IN (0,1)),
			ip          TEXT,
			method      TEXT,
			status      INTEGER
		);

		CREATE INDEX IF NOT EXISTS idx_audit_time           ON user_audit_logs(timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_user_time      ON user_audit_logs(username, timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_action_time    ON user_audit_logs(action, timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_role_time      ON user_audit_logs(role, timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_status_time    ON user_audit_logs(status, timestamp DESC);
		CREATE INDEX IF NOT EXISTS idx_audit_success_action ON user_audit_logs(success, action, timestamp DESC);
	`)
	return err
}

func createNodeMetricsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS node_metrics_flat (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_name        TEXT NOT NULL,
			ts               TEXT NOT NULL,
			cpu_usage        REAL,
			cpu_cores        INTEGER,
			cpu_load1        REAL,
			cpu_load5        REAL,
			cpu_load15       REAL,
			memory_total     INTEGER,
			memory_used      INTEGER,
			memory_available INTEGER,
			memory_usage     REAL,
			temp_cpu         INTEGER,
			temp_gpu         INTEGER,
			temp_nvme        INTEGER,
			disk_total       INTEGER,
			disk_used        INTEGER,
			disk_free        INTEGER,
			disk_usage       REAL,
			net_lo_rx_kbps   REAL,
			net_lo_tx_kbps   REAL,
			net_eth0_rx_kbps REAL,
			net_eth0_tx_kbps REAL,
			UNIQUE(node_name, ts)
		);
		CREATE INDEX IF NOT EXISTS idx_node_metrics_flat_ts
			ON node_metrics_flat(ts DESC);
		CREATE INDEX IF NOT EXISTS idx_node_metrics_flat_node_ts
			ON node_metrics_flat(node_name, ts DESC);
	`)
	return err
}

func createTopProcessesTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS node_top_processes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			node_name   TEXT NOT NULL,
			ts          TEXT NOT NULL,
			pid         INTEGER NOT NULL,
			user        TEXT,
			command     TEXT,
			cpu_percent REAL,
			memory_mb   REAL,
			UNIQUE(node_name, ts, pid)
		);
		CREATE INDEX IF NOT EXISTS idx_node_top_processes_ts
			ON node_top_processes(ts DESC);
		CREATE INDEX IF NOT EXISTS idx_node_topproc_node_ts
			ON node_top_processes(node_name, ts DESC);
	`)
	return err
}

func createNotifySlackTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notify_slack (
			id           INTEGER PRIMARY KEY CHECK (id = 1),
			enable       INTEGER NOT NULL DEFAULT 0,
			webhook      TEXT,
			interval_sec INTEGER NOT NULL DEFAULT 5,
			updated_at   TEXT
		);
	`)
	return err
}

func createNotifyMailTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS notify_mail (
			id           INTEGER PRIMARY KEY CHECK (id = 1),
			enable       INTEGER NOT NULL DEFAULT 0,
			smtp_host    TEXT,
			smtp_port    TEXT DEFAULT '587',
			username     TEXT,
			password     TEXT,
			from_addr    TEXT,
			to_addrs     TEXT,
			interval_sec INTEGER NOT NULL DEFAULT 60,
			updated_at   TEXT
		);
	`)
	return err
}

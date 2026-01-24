// atlhyper_master_v2/database/sqlite/migrations.go
// 数据库迁移
package sqlite

import (
	"database/sql"
	"log"
	"time"

	"AtlHyper/atlhyper_master_v2/config"

	"golang.org/x/crypto/bcrypt"
)

// migrate 执行数据库迁移
func migrate(db *sql.DB) error {
	migrations := []string{
		// ==================== 集群事件表（只存 Warning，业务去重）====================
		// dedup_key = MD5(cluster_id + involved_kind + involved_namespace + involved_name + reason)
		// 同一资源的同一 Reason 只存一条，更新 count/last_timestamp/message
		`CREATE TABLE IF NOT EXISTS cluster_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dedup_key TEXT UNIQUE NOT NULL,
			cluster_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'Warning',
			reason TEXT NOT NULL,
			message TEXT,
			source_component TEXT,
			source_host TEXT,
			involved_kind TEXT,
			involved_name TEXT,
			involved_namespace TEXT,
			first_timestamp TEXT NOT NULL,
			last_timestamp TEXT NOT NULL,
			count INTEGER DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_cluster_time ON cluster_events(cluster_id, last_timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_involved ON cluster_events(cluster_id, involved_kind, involved_namespace, involved_name)`,
		`CREATE INDEX IF NOT EXISTS idx_events_reason ON cluster_events(cluster_id, reason, last_timestamp DESC)`,

		// ==================== 通知渠道表 ====================
		`CREATE TABLE IF NOT EXISTS notify_channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			enabled INTEGER DEFAULT 0,
			config TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		// ==================== 用户表 ====================
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			email TEXT,
			role INTEGER DEFAULT 3,
			status INTEGER DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_login_at TEXT,
			last_login_ip TEXT
		)`,

		// ==================== 集群表 ====================
		`CREATE TABLE IF NOT EXISTS clusters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_uid TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			environment TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		// ==================== 审计日志表 ====================
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			user_id INTEGER,
			username TEXT,
			role INTEGER,
			source TEXT,
			action TEXT,
			resource TEXT,
			method TEXT,
			request_body TEXT,
			status_code INTEGER,
			success INTEGER,
			error_message TEXT,
			ip TEXT,
			user_agent TEXT,
			duration_ms INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id, timestamp DESC)`,

		// ==================== 指令历史表 ====================
		`CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command_id TEXT UNIQUE NOT NULL,
			cluster_id TEXT NOT NULL,
			source TEXT,
			user_id INTEGER,
			action TEXT NOT NULL,
			target_kind TEXT,
			target_namespace TEXT,
			target_name TEXT,
			params TEXT,
			status TEXT NOT NULL,
			result TEXT,
			error_message TEXT,
			created_at TEXT NOT NULL,
			started_at TEXT,
			finished_at TEXT,
			duration_ms INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_cluster ON command_history(cluster_id, created_at DESC)`,

		// ==================== 系统设置表 ====================
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			description TEXT,
			updated_at TEXT NOT NULL,
			updated_by INTEGER
		)`,

		// ==================== AI 对话表 ====================
		`CREATE TABLE IF NOT EXISTS ai_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			cluster_id TEXT NOT NULL,
			title TEXT,
			message_count INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_conv_user ON ai_conversations(user_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_conv_cluster ON ai_conversations(cluster_id)`,

		// ==================== AI 消息表 ====================
		`CREATE TABLE IF NOT EXISTS ai_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			tool_calls TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_msg_conv ON ai_messages(conversation_id, created_at ASC)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Printf("[SQLiteDB] 数据库迁移失败: %v\nSQL: %s", err, m)
			return err
		}
	}

	// 初始化默认管理员（从配置读取）
	if err := initDefaultAdmin(db); err != nil {
		return err
	}

	log.Println("[SQLiteDB] 数据库迁移完成")
	return nil
}

// initDefaultAdmin 初始化默认管理员用户
// 从 config.GlobalConfig.Admin 读取配置
// 如果用户已存在则跳过
func initDefaultAdmin(db *sql.DB) error {
	adminCfg := config.GlobalConfig.Admin
	if adminCfg.Username == "" || adminCfg.Password == "" {
		log.Println("[SQLiteDB] 未配置默认管理员，跳过创建")
		return nil
	}

	// 检查用户是否已存在
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", adminCfg.Username).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Printf("[SQLiteDB] 管理员用户 %s 已存在，跳过创建", adminCfg.Username)
		return nil
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminCfg.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[SQLiteDB] 密码加密失败: %v", err)
		return err
	}

	// 插入默认管理员
	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, display_name, role, status, created_at, updated_at)
		VALUES (?, ?, ?, 3, 1, ?, ?)`,
		adminCfg.Username, string(hashedPassword), adminCfg.DisplayName, now, now)
	if err != nil {
		log.Printf("[SQLiteDB] 创建默认管理员失败: %v", err)
		return err
	}

	log.Printf("[SQLiteDB] 已创建默认管理员: %s", adminCfg.Username)
	return nil
}

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
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tool_calls INTEGER DEFAULT 0,
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

		// ==================== AI 提供商配置表 ====================
		`CREATE TABLE IF NOT EXISTS ai_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			provider TEXT NOT NULL,
			api_key TEXT NOT NULL,
			model TEXT NOT NULL,
			description TEXT,
			total_requests INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
			last_used_at TEXT,
			last_error TEXT,
			last_error_at TEXT,
			status TEXT DEFAULT 'unknown',
			status_checked_at TEXT,
			created_at TEXT NOT NULL,
			created_by INTEGER,
			updated_at TEXT NOT NULL,
			updated_by INTEGER,
			deleted_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_provider_deleted ON ai_providers(deleted_at)`,

		// ==================== AI 当前配置表 (单行) ====================
		`CREATE TABLE IF NOT EXISTS ai_active_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			enabled INTEGER DEFAULT 0,
			provider_id INTEGER,
			tool_timeout INTEGER DEFAULT 30,
			updated_at TEXT NOT NULL,
			updated_by INTEGER,
			FOREIGN KEY (provider_id) REFERENCES ai_providers(id)
		)`,
		// 注意: ai_active_config 初始化由 InitAIActiveConfig() 从配置文件读取并写入
		// 不在此处硬编码默认值，保证配置可追溯

		// ==================== AI 提供商模型表 ====================
		// 各提供商支持的模型列表 (DB管理)
		`CREATE TABLE IF NOT EXISTS ai_provider_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			display_name TEXT,
			is_default INTEGER DEFAULT 0,
			sort_order INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(provider, model)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_models_provider ON ai_provider_models(provider, sort_order)`,

		// ==================== SLO: 入口原始增量数据表 ====================
		// JSON bucket 格式，保留 48 小时
		`CREATE TABLE IF NOT EXISTS slo_metrics_raw (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			host TEXT NOT NULL,
			domain TEXT,
			path_prefix TEXT DEFAULT '/',
			timestamp TEXT NOT NULL,
			total_requests INTEGER NOT NULL DEFAULT 0,
			error_requests INTEGER NOT NULL DEFAULT 0,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			latency_buckets TEXT,
			method_get INTEGER NOT NULL DEFAULT 0,
			method_post INTEGER NOT NULL DEFAULT 0,
			method_put INTEGER NOT NULL DEFAULT 0,
			method_delete INTEGER NOT NULL DEFAULT 0,
			method_other INTEGER NOT NULL DEFAULT 0,
			is_missing INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_raw_cluster_host_ts ON slo_metrics_raw(cluster_id, host, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_raw_timestamp ON slo_metrics_raw(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_raw_domain ON slo_metrics_raw(cluster_id, domain, path_prefix, timestamp)`,

		// ==================== SLO: 入口小时聚合数据表 ====================
		// JSON bucket 格式，保留 90 天
		`CREATE TABLE IF NOT EXISTS slo_metrics_hourly (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			host TEXT NOT NULL,
			domain TEXT,
			path_prefix TEXT DEFAULT '/',
			hour_start TEXT NOT NULL,
			total_requests INTEGER NOT NULL DEFAULT 0,
			error_requests INTEGER NOT NULL DEFAULT 0,
			availability REAL,
			p50_latency_ms INTEGER NOT NULL DEFAULT 0,
			p95_latency_ms INTEGER NOT NULL DEFAULT 0,
			p99_latency_ms INTEGER NOT NULL DEFAULT 0,
			avg_latency_ms INTEGER NOT NULL DEFAULT 0,
			avg_rps REAL NOT NULL DEFAULT 0,
			latency_buckets TEXT,
			method_get INTEGER NOT NULL DEFAULT 0,
			method_post INTEGER NOT NULL DEFAULT 0,
			method_put INTEGER NOT NULL DEFAULT 0,
			method_delete INTEGER NOT NULL DEFAULT 0,
			method_other INTEGER NOT NULL DEFAULT 0,
			sample_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(cluster_id, host, hour_start)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_hourly_hour ON slo_metrics_hourly(hour_start)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_hourly_domain ON slo_metrics_hourly(cluster_id, domain, path_prefix, hour_start)`,

		// ==================== SLO: 服务网格原始数据表 ====================
		// Linkerd inbound 数据，保留 48 小时
		`CREATE TABLE IF NOT EXISTS slo_service_raw (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			name TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			total_requests INTEGER NOT NULL DEFAULT 0,
			error_requests INTEGER NOT NULL DEFAULT 0,
			status_2xx INTEGER NOT NULL DEFAULT 0,
			status_3xx INTEGER NOT NULL DEFAULT 0,
			status_4xx INTEGER NOT NULL DEFAULT 0,
			status_5xx INTEGER NOT NULL DEFAULT 0,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0,
			latency_buckets TEXT,
			tls_request_delta INTEGER NOT NULL DEFAULT 0,
			total_request_delta INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_service_raw_cluster ON slo_service_raw(cluster_id, namespace, name, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_service_raw_ts ON slo_service_raw(timestamp)`,

		// ==================== SLO: 服务网格小时聚合表 ====================
		// Linkerd inbound 聚合，保留 90 天
		`CREATE TABLE IF NOT EXISTS slo_service_hourly (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			name TEXT NOT NULL,
			hour_start TEXT NOT NULL,
			total_requests INTEGER NOT NULL DEFAULT 0,
			error_requests INTEGER NOT NULL DEFAULT 0,
			availability REAL,
			p50_latency_ms INTEGER NOT NULL DEFAULT 0,
			p95_latency_ms INTEGER NOT NULL DEFAULT 0,
			p99_latency_ms INTEGER NOT NULL DEFAULT 0,
			avg_latency_ms INTEGER NOT NULL DEFAULT 0,
			avg_rps REAL NOT NULL DEFAULT 0,
			status_2xx INTEGER NOT NULL DEFAULT 0,
			status_3xx INTEGER NOT NULL DEFAULT 0,
			status_4xx INTEGER NOT NULL DEFAULT 0,
			status_5xx INTEGER NOT NULL DEFAULT 0,
			latency_buckets TEXT,
			mtls_percent REAL,
			sample_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(cluster_id, namespace, name, hour_start)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_service_hourly_hour ON slo_service_hourly(hour_start)`,

		// ==================== SLO: 拓扑边原始数据表 ====================
		// Linkerd outbound 数据，保留 48 小时
		`CREATE TABLE IF NOT EXISTS slo_edge_raw (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			src_namespace TEXT NOT NULL,
			src_name TEXT NOT NULL,
			dst_namespace TEXT NOT NULL,
			dst_name TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			request_delta INTEGER NOT NULL DEFAULT 0,
			failure_delta INTEGER NOT NULL DEFAULT 0,
			latency_sum REAL NOT NULL DEFAULT 0,
			latency_count INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_edge_raw_cluster ON slo_edge_raw(cluster_id, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_edge_raw_ts ON slo_edge_raw(timestamp)`,

		// ==================== SLO: 拓扑边小时聚合表 ====================
		// Linkerd outbound 聚合，保留 90 天
		`CREATE TABLE IF NOT EXISTS slo_edge_hourly (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			src_namespace TEXT NOT NULL,
			src_name TEXT NOT NULL,
			dst_namespace TEXT NOT NULL,
			dst_name TEXT NOT NULL,
			hour_start TEXT NOT NULL,
			total_requests INTEGER NOT NULL DEFAULT 0,
			error_requests INTEGER NOT NULL DEFAULT 0,
			avg_latency_ms INTEGER,
			avg_rps REAL,
			error_rate REAL,
			sample_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(cluster_id, src_namespace, src_name, dst_namespace, dst_name, hour_start)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_edge_hourly_hour ON slo_edge_hourly(hour_start)`,

		// ==================== SLO: 目标配置表 ====================
		// SLO 目标配置，用户可配置不同周期的可用性和延迟目标
		`CREATE TABLE IF NOT EXISTS slo_targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			host TEXT NOT NULL,
			ingress_name TEXT NOT NULL,
			ingress_class TEXT NOT NULL DEFAULT 'nginx',
			namespace TEXT NOT NULL,
			tls INTEGER NOT NULL DEFAULT 1,
			time_range TEXT NOT NULL,
			availability_target REAL NOT NULL DEFAULT 95.00,
			p95_latency_target INTEGER NOT NULL DEFAULT 300,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(cluster_id, host, time_range)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_targets_cluster ON slo_targets(cluster_id)`,

		// ==================== SLO: 状态变更历史表 ====================
		// 记录 SLO 状态变更历史，保留 180 天
		`CREATE TABLE IF NOT EXISTS slo_status_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			host TEXT NOT NULL,
			time_range TEXT NOT NULL,
			old_status TEXT NOT NULL,
			new_status TEXT NOT NULL,
			availability REAL NOT NULL,
			p95_latency INTEGER NOT NULL,
			error_budget_remaining REAL NOT NULL,
			changed_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_status_cluster_host ON slo_status_history(cluster_id, host)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_status_changed_at ON slo_status_history(changed_at)`,

		// ==================== SLO: 路由映射表 ====================
		// 存储 Traefik service 名称到域名/路径的映射关系
		// Agent 采集 IngressRoute CRD 后上报，用于将 service 维度的指标转换为 domain/path 维度
		// 注意：同一个 service 可能服务于多个路径，所以唯一约束基于 domain + path
		`CREATE TABLE IF NOT EXISTS slo_route_mapping (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			domain TEXT NOT NULL,
			path_prefix TEXT NOT NULL DEFAULT '/',
			ingress_name TEXT NOT NULL,
			namespace TEXT NOT NULL,
			tls INTEGER NOT NULL DEFAULT 1,
			service_key TEXT NOT NULL,
			service_name TEXT NOT NULL,
			service_port INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(cluster_id, domain, path_prefix)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_route_mapping_domain ON slo_route_mapping(cluster_id, domain)`,
		`CREATE INDEX IF NOT EXISTS idx_route_mapping_service ON slo_route_mapping(cluster_id, service_key)`,

		// ==================== NodeMetrics: 实时数据表 ====================
		// 每个节点一条记录，覆盖式更新
		`CREATE TABLE IF NOT EXISTS node_metrics_latest (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			node_name TEXT NOT NULL,
			snapshot_json TEXT NOT NULL,
			cpu_usage REAL DEFAULT 0,
			memory_usage REAL DEFAULT 0,
			disk_usage REAL DEFAULT 0,
			cpu_temp REAL DEFAULT 0,
			updated_at TEXT NOT NULL,
			UNIQUE(cluster_id, node_name)
		)`,

		// ==================== NodeMetrics: 历史数据表 ====================
		// 每 5 分钟采样一次，保留 30 天
		`CREATE TABLE IF NOT EXISTS node_metrics_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			node_name TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			cpu_usage REAL DEFAULT 0,
			memory_usage REAL DEFAULT 0,
			disk_usage REAL DEFAULT 0,
			disk_io_read REAL DEFAULT 0,
			disk_io_write REAL DEFAULT 0,
			network_rx REAL DEFAULT 0,
			network_tx REAL DEFAULT 0,
			cpu_temp REAL DEFAULT 0,
			load_1 REAL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_node_metrics_history_cluster_node ON node_metrics_history(cluster_id, node_name)`,
		`CREATE INDEX IF NOT EXISTS idx_node_metrics_history_timestamp ON node_metrics_history(timestamp)`,

		// ==================== AIOps: 基线状态表 ====================
		// 持久化 EMA 状态，用于重启恢复
		`CREATE TABLE IF NOT EXISTS aiops_baseline_states (
			entity_key  TEXT NOT NULL,
			metric_name TEXT NOT NULL,
			ema         REAL NOT NULL,
			variance    REAL NOT NULL,
			count       INTEGER NOT NULL,
			updated_at  INTEGER NOT NULL,
			PRIMARY KEY (entity_key, metric_name)
		)`,

		// ==================== AIOps: 依赖图快照表 ====================
		// 定期持久化图快照，用于重启恢复（每集群一条，覆盖式更新）
		`CREATE TABLE IF NOT EXISTS aiops_dependency_graph_snapshots (
			cluster_id TEXT PRIMARY KEY,
			snapshot   BLOB NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	// 增量迁移：为已存在的表添加新列（忽略错误，可能列已存在）
	alterMigrations := []string{
		// AI 对话表添加累计统计字段
		`ALTER TABLE ai_conversations ADD COLUMN total_input_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_conversations ADD COLUMN total_output_tokens INTEGER DEFAULT 0`,
		`ALTER TABLE ai_conversations ADD COLUMN total_tool_calls INTEGER DEFAULT 0`,
		// 入口指标表添加状态码列
		`ALTER TABLE slo_metrics_raw ADD COLUMN status_2xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_raw ADD COLUMN status_3xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_raw ADD COLUMN status_4xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_raw ADD COLUMN status_5xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_hourly ADD COLUMN status_2xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_hourly ADD COLUMN status_3xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_hourly ADD COLUMN status_4xx INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE slo_metrics_hourly ADD COLUMN status_5xx INTEGER NOT NULL DEFAULT 0`,
	}

	// 删除旧的 snapshot 表（OTel 迁移后不再需要）
	dropMigrations := []string{
		`DROP TABLE IF EXISTS ingress_counter_snapshot`,
		`DROP TABLE IF EXISTS ingress_histogram_snapshot`,
	}

	// 删除旧的 SLO 表（OTel 迁移重建）
	rebuildSLOTables(db)

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Printf("[SQLiteDB] 数据库迁移失败: %v\nSQL: %s", err, m)
			return err
		}
	}

	// 执行增量迁移（忽略错误，可能列已存在）
	for _, m := range alterMigrations {
		if _, err := db.Exec(m); err != nil {
			// 忽略 "duplicate column name" 错误
			log.Printf("[SQLiteDB] 增量迁移跳过（可能列已存在）: %v", err)
		}
	}

	// 删除旧 snapshot 表
	for _, m := range dropMigrations {
		if _, err := db.Exec(m); err != nil {
			log.Printf("[SQLiteDB] 删除旧表跳过: %v", err)
		}
	}

	// 初始化默认管理员（从配置读取）
	if err := initDefaultAdmin(db); err != nil {
		return err
	}

	// 初始化默认 AI 模型列表
	if err := initDefaultAIModels(db); err != nil {
		return err
	}

	// 清理无效的 entrypoint 级别数据
	if err := cleanupEntrypointData(db); err != nil {
		log.Printf("[SQLiteDB] 清理 entrypoint 数据失败: %v", err)
		// 不中断启动，只记录警告
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

// initDefaultAIModels 初始化默认 AI 模型列表
func initDefaultAIModels(db *sql.DB) error {
	// 检查是否已有数据
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM ai_provider_models").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有数据，跳过
	}

	now := time.Now().Format(time.RFC3339)
	models := []struct {
		provider    string
		model       string
		displayName string
		isDefault   int
		sortOrder   int
	}{
		// Gemini 2.5 系列
		{"gemini", "gemini-2.5-flash", "Gemini 2.5 Flash", 1, 1},
		{"gemini", "gemini-2.5-flash-lite", "Gemini 2.5 Flash Lite", 0, 2},
		{"gemini", "gemini-2.5-pro", "Gemini 2.5 Pro", 0, 3},
		// OpenAI
		{"openai", "gpt-4o", "GPT-4o", 1, 1},
		{"openai", "gpt-4o-mini", "GPT-4o Mini", 0, 2},
		{"openai", "gpt-4-turbo", "GPT-4 Turbo", 0, 3},
		{"openai", "gpt-4", "GPT-4", 0, 4},
		{"openai", "o1", "o1", 0, 5},
		{"openai", "o1-mini", "o1 Mini", 0, 6},
		// Anthropic
		{"anthropic", "claude-sonnet-4-20250514", "Claude Sonnet 4", 1, 1},
		{"anthropic", "claude-opus-4-5-20251101", "Claude Opus 4.5", 0, 2},
		{"anthropic", "claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet", 0, 3},
		{"anthropic", "claude-3-5-haiku-20241022", "Claude 3.5 Haiku", 0, 4},
		{"anthropic", "claude-3-opus-20240229", "Claude 3 Opus", 0, 5},
		{"anthropic", "claude-3-haiku-20240307", "Claude 3 Haiku", 0, 6},
	}

	stmt, err := db.Prepare(`INSERT INTO ai_provider_models (provider, model, display_name, is_default, sort_order, created_at) VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range models {
		if _, err := stmt.Exec(m.provider, m.model, m.displayName, m.isDefault, m.sortOrder, now); err != nil {
			log.Printf("[SQLiteDB] 插入默认模型失败: %v", err)
		}
	}

	log.Println("[SQLiteDB] 已初始化默认 AI 模型列表")
	return nil
}

// cleanupEntrypointData 清理无效的 entrypoint 级别 SLO 数据
func cleanupEntrypointData(db *sql.DB) error {
	cleanupQueries := []struct {
		table string
		query string
	}{
		{"slo_metrics_raw", `DELETE FROM slo_metrics_raw WHERE host LIKE '%@entrypoint%'`},
		{"slo_metrics_hourly", `DELETE FROM slo_metrics_hourly WHERE host LIKE '%@entrypoint%'`},
		{"slo_targets", `DELETE FROM slo_targets WHERE host LIKE '%@entrypoint%'`},
		{"slo_status_history", `DELETE FROM slo_status_history WHERE host LIKE '%@entrypoint%'`},
	}

	totalDeleted := int64(0)
	for _, q := range cleanupQueries {
		result, err := db.Exec(q.query)
		if err != nil {
			log.Printf("[SQLiteDB] 清理 %s entrypoint 数据失败: %v", q.table, err)
			continue
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			log.Printf("[SQLiteDB] 已清理 %s 表 %d 条 entrypoint 数据", q.table, rowsAffected)
			totalDeleted += rowsAffected
		}
	}

	if totalDeleted > 0 {
		log.Printf("[SQLiteDB] 共清理 %d 条无效 entrypoint 数据", totalDeleted)
	}
	return nil
}

// rebuildSLOTables OTel 迁移：重建旧格式的 SLO 表
// 旧表使用 12 列固定 bucket，新表使用 JSON bucket + method 分类
func rebuildSLOTables(db *sql.DB) {
	tablesToRebuild := []string{"slo_metrics_raw", "slo_metrics_hourly", "slo_route_mapping"}

	for _, table := range tablesToRebuild {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
		if err != nil || count == 0 {
			continue // 表不存在，会由主迁移创建
		}

		// 检查是否已迁移过（新表有 latency_buckets 列）
		if table == "slo_metrics_raw" || table == "slo_metrics_hourly" {
			var colCount int
			db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='latency_buckets'`, table).Scan(&colCount)
			if colCount > 0 {
				continue // 已经是新格式
			}
		}

		// 删除旧表，由主迁移重建
		if _, err := db.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			log.Printf("[SQLiteDB] 删除旧 %s 表失败: %v", table, err)
		} else {
			log.Printf("[SQLiteDB] 已删除旧 %s 表（将重建为新格式）", table)
		}
	}
}

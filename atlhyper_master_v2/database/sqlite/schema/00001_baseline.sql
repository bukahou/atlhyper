-- 第 1 版：引入迁移工具时已有的全部表结构（与此前每次启动执行的建表语句逐字相同）。
-- 全部 IF NOT EXISTS：已有的库执行后不变，只记下版本号；空库在这里建表。

-- +goose Up
-- ==================== 集群事件表（只存 Warning，业务去重）====================
-- dedup_key = MD5(cluster_id + involved_kind + involved_namespace + involved_name + reason)
-- 同一资源的同一 Reason 只存一条，更新 count/last_timestamp/message
CREATE TABLE IF NOT EXISTS cluster_events (
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
);

CREATE INDEX IF NOT EXISTS idx_events_cluster_time ON cluster_events(cluster_id, last_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_events_involved ON cluster_events(cluster_id, involved_kind, involved_namespace, involved_name);

CREATE INDEX IF NOT EXISTS idx_events_reason ON cluster_events(cluster_id, reason, last_timestamp DESC);

-- ==================== 通知：全局策略（单行，id 恒为 1）====================
CREATE TABLE IF NOT EXISTS notify_policy (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	incident_alerts_enabled INTEGER NOT NULL DEFAULT 1,
	min_severity TEXT NOT NULL DEFAULT 'low',
	notify_resolved INTEGER NOT NULL DEFAULT 1,
	warning_flap_threshold INTEGER NOT NULL DEFAULT 3,
	group_wait_seconds INTEGER NOT NULL DEFAULT 30,
	group_interval_seconds INTEGER NOT NULL DEFAULT 300,
	repeat_interval_seconds INTEGER NOT NULL DEFAULT 14400,
	k8s_event_alerts_enabled INTEGER NOT NULL DEFAULT 0,
	default_receiver_ids TEXT NOT NULL DEFAULT '[]',
	updated_at TEXT NOT NULL
);

-- ==================== 通知：告警（事件 / 心跳 / K8s 事件的统一对象）====================
-- last_notified_kind / last_notified_severity：本次持续期间最近一次已通知的变化与当时的严重度，去重用
CREATE TABLE IF NOT EXISTS alerts (
	fingerprint TEXT PRIMARY KEY,
	source TEXT NOT NULL,
	cluster_id TEXT NOT NULL,
	severity TEXT NOT NULL,
	status TEXT NOT NULL,
	kind TEXT NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	labels TEXT NOT NULL DEFAULT '{}',
	details TEXT NOT NULL DEFAULT '{}',
	starts_at TEXT NOT NULL,
	ends_at TEXT,
	updated_at TEXT NOT NULL,
	last_notified_at TEXT,
	last_notified_kind TEXT NOT NULL DEFAULT '',
	last_notified_severity TEXT NOT NULL DEFAULT '',
	notify_count INTEGER NOT NULL DEFAULT 0,
	suppressed_by TEXT NOT NULL DEFAULT '',
	silenced_by INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_alerts_status_cluster ON alerts(status, cluster_id);

-- ==================== 通知：送达记录（一条告警发给一个接收人的一次通知）====================
-- 接收人的名称与类型冗余存一份：接收人删除后记录仍可读
CREATE TABLE IF NOT EXISTS notify_deliveries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	batch_id TEXT NOT NULL,
	fingerprint TEXT NOT NULL,
	alert_title TEXT NOT NULL,
	receiver_id INTEGER NOT NULL DEFAULT 0,
	receiver_name TEXT NOT NULL DEFAULT '',
	receiver_type TEXT NOT NULL DEFAULT '',
	kind TEXT NOT NULL,
	status TEXT NOT NULL,
	attempts INTEGER NOT NULL DEFAULT 0,
	last_error TEXT NOT NULL DEFAULT '',
	next_attempt_at TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notify_deliveries_created ON notify_deliveries(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notify_deliveries_fingerprint ON notify_deliveries(fingerprint);

CREATE INDEX IF NOT EXISTS idx_notify_deliveries_status_next ON notify_deliveries(status, next_attempt_at);

CREATE INDEX IF NOT EXISTS idx_notify_deliveries_batch ON notify_deliveries(batch_id);

-- ==================== 通知：接收人（config 是整体加密后的 JSON，含 Webhook 地址 / SMTP 密码）====================
CREATE TABLE IF NOT EXISTS notify_receivers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT UNIQUE NOT NULL,
	type TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	config TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

-- ==================== 通知：路由规则（按 position 从小到大匹配，第一条命中生效）====================
CREATE TABLE IF NOT EXISTS notify_rules (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	position INTEGER NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	cluster_ids TEXT NOT NULL DEFAULT '[]',
	sources TEXT NOT NULL DEFAULT '[]',
	min_severity TEXT NOT NULL DEFAULT 'low',
	receiver_ids TEXT NOT NULL DEFAULT '[]',
	group_wait_seconds INTEGER,
	group_interval_seconds INTEGER,
	repeat_interval_seconds INTEGER,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

-- ==================== 通知：静默（不删除，只会过期）====================
-- 全局发件设置（单行）：配置 JSON 整体加密。独立成表：已有库执行 CREATE TABLE IF NOT EXISTS 即可，无需清库
CREATE TABLE IF NOT EXISTS notify_smtp (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	config TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS notify_silences (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	cluster_id TEXT NOT NULL DEFAULT '',
	namespace TEXT NOT NULL DEFAULT '',
	subject TEXT NOT NULL DEFAULT '',
	starts_at TEXT NOT NULL,
	ends_at TEXT NOT NULL,
	comment TEXT NOT NULL,
	created_by TEXT NOT NULL,
	created_at TEXT NOT NULL
);

-- ==================== 用户表 ====================
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	display_name TEXT,
	email TEXT,
	role INTEGER DEFAULT 0,
	status INTEGER DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	last_login_at TEXT,
	last_login_ip TEXT
);

-- ==================== 外部身份表（OIDC issuer + subject → 用户） ====================
-- 独立成表而非给 users 加列：已有库执行 CREATE TABLE IF NOT EXISTS 即可，无需迁移
CREATE TABLE IF NOT EXISTS user_identities (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	issuer TEXT NOT NULL,
	subject TEXT NOT NULL,
	created_at TEXT NOT NULL,
	UNIQUE (issuer, subject)
);

CREATE INDEX IF NOT EXISTS idx_user_identities_user ON user_identities(user_id);

-- ==================== 集群表 ====================
-- 集群注册表：管理员登记的集群与它的 Agent 令牌。令牌只存哈希，明文只在签发时返回一次。
CREATE TABLE IF NOT EXISTS clusters (
	cluster_id      TEXT PRIMARY KEY,
	name            TEXT NOT NULL DEFAULT '',
	description     TEXT NOT NULL DEFAULT '',
	token_hash      TEXT NOT NULL DEFAULT '',
	token_prefix    TEXT NOT NULL DEFAULT '',
	token_issued_at TEXT,
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL
);

-- 每个请求都按令牌哈希查一次集群；没签发令牌的行哈希为空串，不进索引
CREATE UNIQUE INDEX IF NOT EXISTS idx_clusters_token_hash ON clusters(token_hash) WHERE token_hash != '';

-- ==================== 审计日志表 ====================
CREATE TABLE IF NOT EXISTS audit_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp TEXT NOT NULL,
	user_id INTEGER,
	username TEXT,
	role INTEGER,
	source TEXT,
	action TEXT,
	resource TEXT,
	method TEXT,
	path TEXT NOT NULL DEFAULT '',
	request_body TEXT,
	status_code INTEGER,
	success INTEGER,
	error_message TEXT,
	ip TEXT,
	user_agent TEXT,
	duration_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id, timestamp DESC);

-- ==================== 指令历史表 ====================
CREATE TABLE IF NOT EXISTS command_history (
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
);

CREATE INDEX IF NOT EXISTS idx_cmd_cluster ON command_history(cluster_id, created_at DESC);

-- ==================== 系统设置表 ====================
CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT,
	description TEXT,
	updated_at TEXT NOT NULL,
	updated_by INTEGER
);

-- ==================== AI 对话表 ====================
CREATE TABLE IF NOT EXISTS ai_conversations (
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
);

CREATE INDEX IF NOT EXISTS idx_ai_conv_user ON ai_conversations(user_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_conv_cluster ON ai_conversations(cluster_id);

-- ==================== AI 消息表 ====================
CREATE TABLE IF NOT EXISTS ai_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	conversation_id INTEGER NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	tool_calls TEXT,
	created_at TEXT NOT NULL,
	FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ai_msg_conv ON ai_messages(conversation_id, created_at ASC);

-- ==================== AI 提供商配置表 ====================
CREATE TABLE IF NOT EXISTS ai_providers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	provider TEXT NOT NULL,
	api_key TEXT NOT NULL,
	model TEXT NOT NULL,
	base_url TEXT DEFAULT '',
	description TEXT,
	roles TEXT DEFAULT '[]',
	context_window_override INTEGER DEFAULT 0,
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
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_deleted ON ai_providers(deleted_at);

-- ==================== AI 全局设置表 (单行) ====================
CREATE TABLE IF NOT EXISTS ai_settings (
	id INTEGER PRIMARY KEY CHECK (id = 1),
	tool_timeout INTEGER DEFAULT 30,
	updated_at TEXT NOT NULL,
	updated_by INTEGER
);

-- ==================== AI 提供商模型表 ====================
-- 各提供商支持的模型列表 (DB管理)
CREATE TABLE IF NOT EXISTS ai_provider_models (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	provider TEXT NOT NULL,
	model TEXT NOT NULL,
	display_name TEXT,
	is_default INTEGER DEFAULT 0,
	sort_order INTEGER DEFAULT 0,
	context_window INTEGER DEFAULT 0,
	created_at TEXT NOT NULL,
	UNIQUE(provider, model)
);

CREATE INDEX IF NOT EXISTS idx_ai_models_provider ON ai_provider_models(provider, sort_order);

-- ==================== AI 角色预算表 ====================
CREATE TABLE IF NOT EXISTS ai_role_budget (
	role TEXT PRIMARY KEY,
	daily_input_token_limit INTEGER DEFAULT 0,
	daily_output_token_limit INTEGER DEFAULT 0,
	daily_call_limit INTEGER DEFAULT 0,
	daily_input_tokens_used INTEGER DEFAULT 0,
	daily_output_tokens_used INTEGER DEFAULT 0,
	daily_calls_used INTEGER DEFAULT 0,
	daily_reset_at TEXT,
	monthly_input_token_limit INTEGER DEFAULT 0,
	monthly_output_token_limit INTEGER DEFAULT 0,
	monthly_call_limit INTEGER DEFAULT 0,
	monthly_input_tokens_used INTEGER DEFAULT 0,
	monthly_output_tokens_used INTEGER DEFAULT 0,
	monthly_calls_used INTEGER DEFAULT 0,
	monthly_reset_at TEXT,
	fallback_provider_id INTEGER,
	auto_trigger_min_severity TEXT DEFAULT 'critical',
	auto_trigger_mode TEXT DEFAULT 'auto',
	schedule_start_time TEXT DEFAULT '',
	schedule_end_time TEXT DEFAULT '',
	updated_at TEXT NOT NULL
);

-- ==================== AI 分析报告表 ====================
CREATE TABLE IF NOT EXISTS ai_reports (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	incident_id TEXT,
	cluster_id TEXT NOT NULL,
	role TEXT NOT NULL,
	trigger TEXT NOT NULL,
	summary TEXT,
	root_cause_analysis TEXT,
	recommendations TEXT,
	similar_incidents TEXT,
	investigation_steps TEXT,
	evidence_chain TEXT,
	provider_name TEXT,
	model TEXT,
	input_tokens INTEGER DEFAULT 0,
	output_tokens INTEGER DEFAULT 0,
	duration_ms INTEGER DEFAULT 0,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_reports_incident ON ai_reports(incident_id);

CREATE INDEX IF NOT EXISTS idx_ai_reports_cluster ON ai_reports(cluster_id, role, created_at DESC);

-- ==================== SLO: 目标配置表 ====================
-- SLO 目标配置，用户可配置不同周期的可用性和延迟目标
CREATE TABLE IF NOT EXISTS slo_targets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	cluster_id TEXT NOT NULL,
	host TEXT NOT NULL,
	ingress_name TEXT NOT NULL,
	ingress_class TEXT NOT NULL DEFAULT 'nginx',
	namespace TEXT NOT NULL,
	tls INTEGER NOT NULL DEFAULT 1,
	window_days INTEGER NOT NULL DEFAULT 7,
	availability_target REAL NOT NULL DEFAULT 99.00,
	p95_latency_target INTEGER NOT NULL DEFAULT 300,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	UNIQUE(cluster_id, host)
);

CREATE INDEX IF NOT EXISTS idx_slo_targets_cluster ON slo_targets(cluster_id);

-- ==================== SLO: 路由映射表 ====================
-- 存储 Traefik service 名称到域名/路径的映射关系
-- Agent 采集 IngressRoute CRD 后上报，用于将 service 维度的指标转换为 domain/path 维度
-- 注意：同一个 service 可能服务于多个路径，所以唯一约束基于 domain + path
CREATE TABLE IF NOT EXISTS slo_route_mapping (
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
);

CREATE INDEX IF NOT EXISTS idx_route_mapping_domain ON slo_route_mapping(cluster_id, domain);

CREATE INDEX IF NOT EXISTS idx_route_mapping_service ON slo_route_mapping(cluster_id, service_key);

-- ==================== AIOps: 基线状态表 ====================
-- 持久化 EMA 状态，用于重启恢复
CREATE TABLE IF NOT EXISTS aiops_baseline_states (
	cluster_id  TEXT NOT NULL,
	entity_key  TEXT NOT NULL,
	metric_name TEXT NOT NULL,
	ema         REAL NOT NULL,
	variance    REAL NOT NULL,
	count       INTEGER NOT NULL,
	updated_at  INTEGER NOT NULL,
	PRIMARY KEY (cluster_id, entity_key, metric_name)
);

-- ==================== AIOps: 依赖图快照表 ====================
-- 定期持久化图快照，用于重启恢复（每集群一条，覆盖式更新）
CREATE TABLE IF NOT EXISTS aiops_dependency_graph_snapshots (
	cluster_id TEXT PRIMARY KEY,
	snapshot   BLOB NOT NULL,
	created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

-- ==================== AIOps: 事件表 ====================
-- 异常事件记录，关联集群和受影响实体
CREATE TABLE IF NOT EXISTS aiops_incidents (
	id TEXT PRIMARY KEY,
	cluster_id TEXT NOT NULL,
	state TEXT NOT NULL DEFAULT 'firing',
	severity TEXT NOT NULL DEFAULT 'low',
	subject TEXT NOT NULL,
	root_cause TEXT,
	peak_risk REAL NOT NULL DEFAULT 0,
	started_at TEXT NOT NULL,
	escalated_at TEXT,
	resolved_at TEXT,
	close_reason TEXT NOT NULL DEFAULT '',
	duration_s INTEGER NOT NULL DEFAULT 0,
	recurrence INTEGER NOT NULL DEFAULT 0,
	summary TEXT,
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_aiops_incidents_cluster ON aiops_incidents(cluster_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_aiops_incidents_state ON aiops_incidents(state);

CREATE INDEX IF NOT EXISTS idx_aiops_incidents_subject ON aiops_incidents(cluster_id, subject, started_at DESC);

-- ==================== AIOps: 事件受影响实体表 ====================
-- 事件关联的实体信息（风险分数、角色等）
CREATE TABLE IF NOT EXISTS aiops_incident_entities (
	incident_id TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	entity_type TEXT NOT NULL,
	r_local REAL NOT NULL DEFAULT 0,
	r_final REAL NOT NULL DEFAULT 0,
	role TEXT NOT NULL DEFAULT 'affected',
	PRIMARY KEY (incident_id, entity_key),
	FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_aiops_incident_entities_entity ON aiops_incident_entities(entity_key);

-- ==================== AIOps: 事件时间线表 ====================
-- 事件生命周期中的关键事件记录
CREATE TABLE IF NOT EXISTS aiops_incident_timeline (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	incident_id TEXT NOT NULL,
	timestamp TEXT NOT NULL,
	event_type TEXT NOT NULL,
	entity_key TEXT,
	detail TEXT,
	FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_aiops_incident_timeline_inc ON aiops_incident_timeline(incident_id, timestamp ASC);

-- 拓扑归并：受影响事件归到源头事件（以受影响事件为键）。新表，已有库执行 CREATE TABLE IF NOT EXISTS 即可，无需清库
CREATE TABLE IF NOT EXISTS aiops_incident_links (
	incident_id TEXT PRIMARY KEY,
	cluster_id TEXT NOT NULL,
	root_incident_id TEXT NOT NULL,
	path TEXT NOT NULL DEFAULT '[]',
	linked_at TEXT NOT NULL,
	unlinked_at TEXT,
	FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_aiops_incident_links_active ON aiops_incident_links(cluster_id, unlinked_at);

-- 事件的信号累计（关键证据）：按（事件, 成员, 指标）一行。新表，已有库执行 CREATE TABLE IF NOT EXISTS 即可，无需清库
CREATE TABLE IF NOT EXISTS aiops_incident_signals (
	incident_id TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	metric TEXT NOT NULL,
	first_seen TEXT NOT NULL,
	last_seen TEXT NOT NULL,
	hits INTEGER NOT NULL DEFAULT 1,
	max_score REAL NOT NULL DEFAULT 0,
	detail TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (incident_id, entity_key, metric),
	FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
);

-- ==================== GitHub App 安装记录（单行）====================
CREATE TABLE IF NOT EXISTS github_installations (
	id              INTEGER PRIMARY KEY,
	installation_id INTEGER NOT NULL UNIQUE,
	account_login   TEXT NOT NULL,
	created_at      TEXT NOT NULL
);

-- ==================== 仓库映射配置 ====================
CREATE TABLE IF NOT EXISTS repo_config (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	repo            TEXT NOT NULL UNIQUE,
	mapping_enabled INTEGER DEFAULT 0,
	created_at      TEXT NOT NULL,
	updated_at      TEXT NOT NULL
);

-- ==================== 部署配置（每集群一条）====================
CREATE TABLE IF NOT EXISTS deploy_config (
	id            INTEGER PRIMARY KEY,
	cluster_id    TEXT NOT NULL UNIQUE,
	repo_url      TEXT NOT NULL,
	paths         TEXT NOT NULL DEFAULT '[]',
	interval_sec  INTEGER DEFAULT 60,
	auto_deploy   INTEGER DEFAULT 1,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL
);

-- ==================== 部署历史 ====================
CREATE TABLE IF NOT EXISTS deploy_history (
	id                INTEGER PRIMARY KEY AUTOINCREMENT,
	cluster_id        TEXT NOT NULL,
	path              TEXT NOT NULL,
	namespace         TEXT NOT NULL,
	commit_sha        TEXT NOT NULL,
	commit_message    TEXT,
	commit_author     TEXT DEFAULT '',
	commit_avatar_url TEXT DEFAULT '',
	pr_number         INTEGER DEFAULT 0,
	pr_title          TEXT DEFAULT '',
	pr_url            TEXT DEFAULT '',
	changed_files     TEXT DEFAULT '[]',
	compare_url       TEXT DEFAULT '',
	source_repo       TEXT DEFAULT '',
	source_commit_sha TEXT DEFAULT '',
	deployed_at       TEXT NOT NULL,
	trigger           TEXT NOT NULL DEFAULT 'auto',
	status            TEXT DEFAULT 'pending',
	duration_ms       INTEGER DEFAULT 0,
	resource_total    INTEGER DEFAULT 0,
	resource_changed  INTEGER DEFAULT 0,
	error_message     TEXT
);

CREATE INDEX IF NOT EXISTS idx_deploy_history_lookup ON deploy_history(cluster_id, path, deployed_at DESC);

-- +goose Down
-- 第 1 版是起点，不提供回退。

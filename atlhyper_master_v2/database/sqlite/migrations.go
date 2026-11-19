// atlhyper_master_v2/database/sqlite/migrations.go
// 数据库迁移
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"time"

	"github.com/pressly/goose/v3"

	"AtlHyper/atlhyper_master_v2/database/sqlite/schema"

	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/common/logger"

	"golang.org/x/crypto/bcrypt"
)

var log = logger.Module("Migrations")

// migrate 先按版本执行表结构迁移（schema/*.sql，goose），再补齐种子数据。
// 改表一律在 schema/ 下新增迁移文件；种子函数只补缺的默认值，可重复执行。
func migrate(db *sql.DB) error {
	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, schema.FS, goose.WithDisableGlobalRegistry(true))
	if err != nil {
		return fmt.Errorf("加载表结构迁移: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("执行表结构迁移: %w", err)
	}
	for _, r := range results {
		log.Info("已执行表结构迁移", "version", r.Source.Version, "file", path.Base(r.Source.Path), "duration", r.Duration.String())
	}
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("读取表结构版本: %w", err)
	}

	// 初始化默认管理员（从配置读取）
	if err := initDefaultAdmin(db); err != nil {
		return err
	}

	// 初始化默认 AI 模型列表
	if err := initDefaultAIModels(db); err != nil {
		return err
	}

	// 初始化默认角色预算
	if err := initDefaultRoleBudgets(db); err != nil {
		return err
	}

	log.Info("数据库迁移完成", "schemaVersion", version)
	return nil
}

// initDefaultAdmin 初始化默认管理员用户
// 从 config.GlobalConfig.Admin 读取配置
// 如果用户已存在则跳过
func initDefaultAdmin(db *sql.DB) error {
	adminCfg := config.GlobalConfig.Admin
	if adminCfg.Username == "" || adminCfg.Password == "" {
		log.Info("未配置默认管理员，跳过创建")
		return nil
	}

	// 检查用户是否已存在
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", adminCfg.Username).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Info("管理员用户已存在，跳过创建", "username", adminCfg.Username)
		return nil
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminCfg.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("密码加密失败", "err", err)
		return err
	}

	// 插入默认管理员
	now := utcStamp(time.Now())
	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, display_name, role, status, created_at, updated_at)
		VALUES (?, ?, ?, 3, 1, ?, ?)`,
		adminCfg.Username, string(hashedPassword), adminCfg.DisplayName, now, now)
	if err != nil {
		log.Error("创建默认管理员失败", "err", err)
		return err
	}

	log.Info("已创建默认管理员", "username", adminCfg.Username)
	return nil
}

// initDefaultAIModels 初始化默认 AI 模型列表
// 逐行 INSERT OR IGNORE（用户 2026-10-12 裁定 D2）：每次启动补齐缺失的内置模型，
// 已有行（含用户改过的）不动。依赖 UNIQUE(provider, model)。
func initDefaultAIModels(db *sql.DB) error {
	now := utcStamp(time.Now())
	models := []struct {
		provider      string
		model         string
		displayName   string
		isDefault     int
		sortOrder     int
		contextWindow int // tokens
	}{
		// Gemini 2.5 系列
		{"gemini", "gemini-2.5-flash", "Gemini 2.5 Flash", 1, 1, 1048576},
		{"gemini", "gemini-2.5-flash-lite", "Gemini 2.5 Flash Lite", 0, 2, 1048576},
		{"gemini", "gemini-2.5-pro", "Gemini 2.5 Pro", 0, 3, 1048576},
		// OpenAI
		{"openai", "gpt-4o", "GPT-4o", 1, 1, 128000},
		{"openai", "gpt-4o-mini", "GPT-4o Mini", 0, 2, 128000},
		{"openai", "gpt-4-turbo", "GPT-4 Turbo", 0, 3, 128000},
		{"openai", "gpt-4", "GPT-4", 0, 4, 8192},
		{"openai", "o1", "o1", 0, 5, 200000},
		{"openai", "o1-mini", "o1 Mini", 0, 6, 128000},
		// Anthropic（模型 ID 不带日期后缀）
		{"anthropic", "claude-haiku-5-5", "Claude Haiku 5.5", 1, 1, 1000000},
		{"anthropic", "claude-sonnet-5-5", "Claude Sonnet 5.5", 0, 2, 1000000},
		{"anthropic", "claude-opus-5-5", "Claude Opus 5.5", 0, 3, 1000000},
		{"anthropic", "claude-fable-5-2", "Claude Fable 5.2", 0, 4, 1000000},
		// Ollama (本地部署)
		{"ollama", "qwen2.5:14b", "Qwen 2.5 14B", 1, 1, 32768},
		{"ollama", "qwen2.5:7b", "Qwen 2.5 7B", 0, 2, 32768},
		{"ollama", "qwen2.5:32b", "Qwen 2.5 32B", 0, 3, 32768},
		{"ollama", "llama3.1:8b", "Llama 3.1 8B", 0, 4, 131072},
		{"ollama", "deepseek-r1:14b", "DeepSeek R1 14B", 0, 5, 65536},
		// AWS Bedrock（东京 In-Region on-demand；context_window 取模型卡标称值的保守整数）
		// ⚠️ 冒烟（流式 + 多轮 tool call）不通过的模型须从此处移除
		{"bedrock", "qwen.qwen3-235b-a22b-2507-v1:0", "Qwen3 235B A22B", 1, 1, 256000},
		{"bedrock", "openai.gpt-oss-120b-1:0", "gpt-oss 120B", 0, 2, 128000},
		{"bedrock", "minimax.minimax-m2.5", "MiniMax M2.5", 0, 3, 196000},
	}

	stmt, err := db.Prepare(`INSERT OR IGNORE INTO ai_provider_models (provider, model, display_name, is_default, sort_order, context_window, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	added := 0
	for _, m := range models {
		res, err := stmt.Exec(m.provider, m.model, m.displayName, m.isDefault, m.sortOrder, m.contextWindow, now)
		if err != nil {
			log.Warn("插入默认模型失败", "provider", m.provider, "model", m.model, "err", err)
			continue
		}
		if n, _ := res.RowsAffected(); n > 0 {
			added++
		}
	}

	if added > 0 {
		log.Info("已补齐内置 AI 模型", "added", added)
	}
	return nil
}

// initDefaultRoleBudgets 初始化默认角色预算（种子数据）
func initDefaultRoleBudgets(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM ai_role_budget").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := utcStamp(time.Now())
	budgets := []struct {
		role                    string
		dailyInputTokenLimit    int
		dailyOutputTokenLimit   int
		dailyCallLimit          int
		monthlyInputTokenLimit  int
		monthlyOutputTokenLimit int
		monthlyCallLimit        int
		autoTriggerMinSeverity  string
	}{
		{"background", 400000, 100000, 50, 4000000, 1000000, 500, "low"},
		{"chat", 800000, 200000, 100, 8000000, 2000000, 1000, "off"},
		{"analysis", 1600000, 400000, 20, 8000000, 2000000, 100, "high"},
	}

	stmt, err := db.Prepare(`INSERT INTO ai_role_budget
		(role, daily_input_token_limit, daily_output_token_limit, daily_call_limit,
		 monthly_input_token_limit, monthly_output_token_limit, monthly_call_limit,
		 auto_trigger_min_severity, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range budgets {
		if _, err := stmt.Exec(b.role,
			b.dailyInputTokenLimit, b.dailyOutputTokenLimit, b.dailyCallLimit,
			b.monthlyInputTokenLimit, b.monthlyOutputTokenLimit, b.monthlyCallLimit,
			b.autoTriggerMinSeverity, now); err != nil {
			log.Warn("插入默认角色预算失败", "role", b.role, "err", err)
		}
	}

	log.Info("已初始化默认角色预算")
	return nil
}

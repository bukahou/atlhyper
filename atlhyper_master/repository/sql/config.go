// repository/sql/config.go
//
// 配置仓库 SQL 实现
//
// 本文件实现了 repository.ConfigRepository 接口，提供系统配置的存储和查询。
// 当前支持两种通知配置: Slack 通知和邮件通知。
//
// 实现的接口方法:
//   - GetSlackConfig: 获取 Slack 通知配置
//   - UpdateSlackConfig: 更新 Slack 通知配置
//   - GetMailConfig: 获取邮件通知配置
//   - UpdateMailConfig: 更新邮件通知配置
//
// 数据库表:
//   - notify_slack: Slack 通知配置 (单行表，id=1)
//   - notify_mail: 邮件通知配置 (单行表，id=1)
//
// 设计说明:
//   配置表采用"单行表"模式，每种配置类型只有一条记录 (id=1)。
//   更新操作会先检查记录是否存在，不存在则 INSERT，存在则 UPDATE。
package sql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/store"
)

// ============================================================================
// ConfigRepo 配置仓库实现
// ============================================================================

// ConfigRepo 配置仓库的 SQL 实现
// 实现 repository.ConfigRepository 接口
type ConfigRepo struct{}

// ============================================================================
// Slack 配置
// ============================================================================

// GetSlackConfig 获取 Slack 通知配置
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - *repository.SlackConfig: Slack 配置实体
//   - error: 查询失败时返回错误
//
// 特殊处理:
//   - 如果配置不存在 (ErrNoRows)，返回默认配置而非错误
//   - 默认配置: 禁用状态，空 webhook，5秒间隔
//
// 默认值:
//   - Enable: false
//   - Webhook: ""
//   - IntervalSec: 5
func (r *ConfigRepo) GetSlackConfig(ctx context.Context) (*repository.SlackConfig, error) {
	// 使用 NullXxx 类型处理可能为空的字段
	var (
		id  sql.NullInt64
		en  sql.NullInt64
		wh  sql.NullString
		iv  sql.NullInt64
		upd sql.NullString
	)

	err := store.DB.QueryRowContext(ctx, Q.Config.GetSlack).Scan(&id, &en, &wh, &iv, &upd)

	// 准备默认配置
	cfg := &repository.SlackConfig{
		ID:          1,
		Enable:      false,
		Webhook:     "",
		IntervalSec: 5,
		UpdatedAt:   time.Time{},
	}

	// 记录不存在时返回默认配置
	if err == sql.ErrNoRows {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	// 解析字段值
	cfg.Enable = en.Valid && en.Int64 != 0
	if wh.Valid {
		cfg.Webhook = strings.TrimSpace(wh.String)
	}
	if iv.Valid && iv.Int64 > 0 {
		cfg.IntervalSec = iv.Int64
	}
	if upd.Valid {
		if t, e := time.Parse(time.RFC3339, upd.String); e == nil {
			cfg.UpdatedAt = t
		}
	}
	return cfg, nil
}

// UpdateSlackConfig 更新 Slack 通知配置
//
// 参数:
//   - ctx: 上下文
//   - cfg: Slack 配置实体，包含:
//   - Enable: 是否启用
//   - Webhook: Slack Webhook URL
//   - IntervalSec: 发送间隔 (秒)
//
// 返回:
//   - error: 更新失败时返回错误
//
// 执行逻辑:
//  1. 检查 id=1 的记录是否存在
//  2. 不存在则 INSERT，存在则 UPDATE
//  3. 自动设置 updated_at 为当前时间
//
// 注意:
//   - Webhook URL 会被 TrimSpace 处理
//   - Enable 布尔值会被转换为整数 (0/1)
func (r *ConfigRepo) UpdateSlackConfig(ctx context.Context, cfg *repository.SlackConfig) error {
	// 检查记录是否存在
	var exists int
	store.DB.QueryRowContext(ctx, Q.Config.CountSlack).Scan(&exists)

	if exists == 0 {
		// 记录不存在，执行 INSERT
		_, err := store.DB.ExecContext(ctx, Q.Config.InsertSlack,
			boolToInt(cfg.Enable),
			cfg.Webhook,
			cfg.IntervalSec,
			time.Now().Format(time.RFC3339))
		return err
	}

	// 记录存在，执行 UPDATE
	_, err := store.DB.ExecContext(ctx, Q.Config.UpdateSlack,
		boolToInt(cfg.Enable),
		strings.TrimSpace(cfg.Webhook),
		cfg.IntervalSec,
		time.Now().Format(time.RFC3339))
	return err
}

// ============================================================================
// 邮件配置
// ============================================================================

// GetMailConfig 获取邮件通知配置
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - *repository.MailConfig: 邮件配置实体
//   - error: 查询失败时返回错误
//
// 特殊处理:
//   - 如果配置不存在 (ErrNoRows)，返回默认配置而非错误
//   - 默认配置: 禁用状态，端口 587，60秒间隔
//
// 默认值:
//   - Enable: false
//   - SMTPPort: "587"
//   - IntervalSec: 60
func (r *ConfigRepo) GetMailConfig(ctx context.Context) (*repository.MailConfig, error) {
	// 使用 NullXxx 类型处理可能为空的字段
	var (
		id       sql.NullInt64
		en       sql.NullInt64
		host     sql.NullString
		port     sql.NullString
		user     sql.NullString
		pass     sql.NullString
		fromAddr sql.NullString
		toAddrs  sql.NullString
		iv       sql.NullInt64
		upd      sql.NullString
	)

	err := store.DB.QueryRowContext(ctx, Q.Config.GetMail).Scan(&id, &en, &host, &port, &user, &pass, &fromAddr, &toAddrs, &iv, &upd)

	// 准备默认配置
	cfg := &repository.MailConfig{
		ID:          1,
		Enable:      false,
		SMTPHost:    "",
		SMTPPort:    "587", // 默认使用 TLS 端口
		Username:    "",
		Password:    "",
		FromAddr:    "",
		ToAddrs:     "",
		IntervalSec: 60,
		UpdatedAt:   time.Time{},
	}

	// 记录不存在时返回默认配置
	if err == sql.ErrNoRows {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	// 解析字段值
	cfg.Enable = en.Valid && en.Int64 != 0
	if host.Valid {
		cfg.SMTPHost = strings.TrimSpace(host.String)
	}
	if port.Valid && strings.TrimSpace(port.String) != "" {
		cfg.SMTPPort = strings.TrimSpace(port.String)
	}
	if user.Valid {
		cfg.Username = strings.TrimSpace(user.String)
	}
	if pass.Valid {
		cfg.Password = pass.String // 密码不做 TrimSpace
	}
	if fromAddr.Valid {
		cfg.FromAddr = strings.TrimSpace(fromAddr.String)
	}
	if toAddrs.Valid {
		cfg.ToAddrs = strings.TrimSpace(toAddrs.String)
	}
	if iv.Valid && iv.Int64 > 0 {
		cfg.IntervalSec = iv.Int64
	}
	if upd.Valid {
		if t, e := time.Parse(time.RFC3339, upd.String); e == nil {
			cfg.UpdatedAt = t
		}
	}
	return cfg, nil
}

// UpdateMailConfig 更新邮件通知配置
//
// 参数:
//   - ctx: 上下文
//   - cfg: 邮件配置实体，包含:
//   - Enable: 是否启用
//   - SMTPHost: SMTP 服务器地址
//   - SMTPPort: SMTP 端口 (通常 25/465/587)
//   - Username: SMTP 认证用户名
//   - Password: SMTP 认证密码
//   - FromAddr: 发件人地址
//   - ToAddrs: 收件人地址 (多个用逗号分隔)
//   - IntervalSec: 发送间隔 (秒)
//
// 返回:
//   - error: 更新失败时返回错误
//
// 执行逻辑:
//  1. 检查 id=1 的记录是否存在
//  2. 不存在则 INSERT，存在则 UPDATE
//  3. 自动设置 updated_at 为当前时间
//
// 安全注意:
//   - 密码以明文存储，建议在应用层加密
//   - 生产环境应考虑使用密钥管理服务
func (r *ConfigRepo) UpdateMailConfig(ctx context.Context, cfg *repository.MailConfig) error {
	// 检查记录是否存在
	var exists int
	store.DB.QueryRowContext(ctx, Q.Config.CountMail).Scan(&exists)

	if exists == 0 {
		// 记录不存在，执行 INSERT
		_, err := store.DB.ExecContext(ctx, Q.Config.InsertMail,
			boolToInt(cfg.Enable),
			cfg.SMTPHost,
			cfg.SMTPPort,
			cfg.Username,
			cfg.Password,
			cfg.FromAddr,
			cfg.ToAddrs,
			cfg.IntervalSec,
			time.Now().Format(time.RFC3339))
		return err
	}

	// 记录存在，执行 UPDATE
	_, err := store.DB.ExecContext(ctx, Q.Config.UpdateMail,
		boolToInt(cfg.Enable),
		strings.TrimSpace(cfg.SMTPHost),
		strings.TrimSpace(cfg.SMTPPort),
		strings.TrimSpace(cfg.Username),
		cfg.Password, // 密码不做 TrimSpace
		strings.TrimSpace(cfg.FromAddr),
		strings.TrimSpace(cfg.ToAddrs),
		cfg.IntervalSec,
		time.Now().Format(time.RFC3339))
	return err
}

// ============================================================================
// 辅助函数
// ============================================================================

// boolToInt 将布尔值转换为整数
//
// SQLite 不支持原生 BOOLEAN 类型，使用 INTEGER 存储:
//   - true  -> 1
//   - false -> 0
//
// 参数:
//   - b: 布尔值
//
// 返回:
//   - int: 0 或 1
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

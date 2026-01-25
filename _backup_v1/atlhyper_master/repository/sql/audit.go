// repository/sql/audit.go
//
// 审计日志仓库 SQL 实现
//
// 本文件实现了 repository.AuditRepository 接口，提供用户操作审计日志的存储和查询。
// 审计日志用于记录用户的每一次操作，支持安全审计和问题追溯。
//
// 实现的接口方法:
//   - Insert: 插入审计日志
//   - GetAll: 获取所有审计日志
//   - GetByUserID: 获取指定用户的审计日志
//
// 数据库表: user_audit_logs
// 字段: id, timestamp, user_id, username, role, action, success, ip, method, status
//
// 记录的操作类型 (action):
//   - login: 用户登录
//   - logout: 用户登出
//   - api_call: API 调用
//   - config_change: 配置变更
//   - user_create: 创建用户
//   - user_delete: 删除用户
//   - 等等...
package sql

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master/repository"
	"AtlHyper/atlhyper_master/store"
)

// ============================================================================
// AuditRepo 审计日志仓库实现
// ============================================================================

// AuditRepo 审计日志仓库的 SQL 实现
// 实现 repository.AuditRepository 接口
type AuditRepo struct{}

// ============================================================================
// 写入方法
// ============================================================================

// Insert 插入审计日志
//
// 参数:
//   - ctx: 上下文
//   - log: 审计日志实体，包含以下字段:
//   - Timestamp: 操作时间 (可选，为空时自动填充当前时间)
//   - UserID: 用户ID
//   - Username: 用户名
//   - Role: 用户角色
//   - Action: 操作类型 (如 "login", "api_call")
//   - Success: 操作是否成功
//   - IP: 客户端IP地址 (可选)
//   - Method: HTTP 方法 (可选，如 "GET", "POST")
//   - Status: HTTP 状态码 (可选)
//
// 返回:
//   - error: 插入失败时返回错误
//
// 使用场景:
//   - 用户登录/登出时记录
//   - API 请求完成后记录
//   - 敏感操作 (配置变更、用户管理) 时记录
func (r *AuditRepo) Insert(ctx context.Context, log *repository.AuditLog) error {
	// 如果没有提供时间戳，使用当前时间
	ts := log.Timestamp
	if ts == "" {
		ts = time.Now().Format("2006-01-02T15:04:05Z07:00")
	}

	// 将布尔值转换为整数 (SQLite 兼容)
	successInt := 0
	if log.Success {
		successInt = 1
	}

	// 执行插入
	_, err := store.DB.ExecContext(ctx, Q.Audit.Insert,
		ts,
		log.UserID,
		log.Username,
		log.Role,
		log.Action,
		successInt,
		nullString(log.IP),     // IP 可能为空
		nullString(log.Method), // Method 可能为空
		nullInt(log.Status),    // Status 为 0 时视为空
	)
	return err
}

// ============================================================================
// 查询方法
// ============================================================================

// GetAll 获取所有审计日志
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []repository.AuditLog: 审计日志列表，按时间倒序排列
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 管理界面审计日志列表展示
//   - 安全审计报告生成
//
// 注意:
//   - 返回所有记录，大数据量时应考虑分页
//   - 最新的日志在前
func (r *AuditRepo) GetAll(ctx context.Context) ([]repository.AuditLog, error) {
	rows, err := store.DB.QueryContext(ctx, Q.Audit.GetAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// GetByUserID 获取指定用户的审计日志
//
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - limit: 返回记录数限制
//
// 返回:
//   - []repository.AuditLog: 审计日志列表，按时间倒序排列
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 用户详情页展示该用户的操作历史
//   - 分析单个用户的行为模式
func (r *AuditRepo) GetByUserID(ctx context.Context, userID int, limit int) ([]repository.AuditLog, error) {
	rows, err := store.DB.QueryContext(ctx, Q.Audit.GetByUserID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

// ============================================================================
// 内部辅助函数
// ============================================================================

// scanAuditLogs 从查询结果扫描审计日志列表
//
// 参数:
//   - rows: sql.Rows 查询结果集
//
// 返回:
//   - []repository.AuditLog: 审计日志列表
//   - error: 扫描失败时返回错误
//
// 处理逻辑:
//   - success 字段从 INTEGER (0/1) 转换为 bool
//   - ip, method 可能为 NULL，使用 sql.NullString 处理
//   - status 可能为 NULL，使用 sql.NullInt64 处理
//   - IPv6 回环地址 ::1 会被规范化为 127.0.0.1
func scanAuditLogs(rows *sql.Rows) ([]repository.AuditLog, error) {
	var logs []repository.AuditLog

	for rows.Next() {
		var (
			log        repository.AuditLog
			successInt int
			ipNS       sql.NullString
			methodNS   sql.NullString
			statusNI64 sql.NullInt64
		)

		// 扫描所有字段
		if err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Username,
			&log.Role,
			&log.Action,
			&successInt,
			&ipNS,
			&methodNS,
			&statusNI64,
			&log.Timestamp,
		); err != nil {
			return nil, err
		}

		// 转换字段类型
		log.Success = (successInt == 1)
		if ipNS.Valid {
			// 规范化 IPv6 回环地址
			log.IP = normalizeLoopback(ipNS.String)
		}
		if methodNS.Valid {
			log.Method = methodNS.String
		}
		if statusNI64.Valid {
			log.Status = int(statusNI64.Int64)
		}

		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// normalizeLoopback 规范化回环地址
//
// 将 IPv6 回环地址 "::1" 转换为 IPv4 格式 "127.0.0.1"
// 便于前端展示和日志分析
//
// 参数:
//   - ip: 原始 IP 地址字符串
//
// 返回:
//   - string: 规范化后的 IP 地址
func normalizeLoopback(ip string) string {
	if ip == "::1" {
		return "127.0.0.1"
	}
	return ip
}

// nullString 将字符串转换为 sql.NullString
//
// 空字符串会被转换为 NULL，非空字符串保持原值
//
// 参数:
//   - s: 原始字符串
//
// 返回:
//   - sql.NullString: 可空字符串
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullInt 将整数转换为 sql.NullInt64
//
// 值为 0 时会被转换为 NULL，非零值保持原值
// 主要用于 HTTP 状态码，0 表示未记录
//
// 参数:
//   - i: 原始整数
//
// 返回:
//   - sql.NullInt64: 可空整数
func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

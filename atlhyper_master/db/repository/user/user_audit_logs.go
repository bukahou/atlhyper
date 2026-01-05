package user

import (
	"AtlHyper/atlhyper_master/db/utils"
	"AtlHyper/atlhyper_master/model"
	"database/sql"
)

// =======================================================================
// ✅ GetUserAuditLogs：查询所有用户审计日志
//
// 参数：
//   - 不需要输入任何参数获取全部日志
// 返回 error：执行成功返回 nil，否则返回错误信息
// =======================================================================

func GetUserAuditLogs() ([]model.GetUserAuditLogsResponse, error) {
	rows, err := utils.DB.Query(`
		SELECT
			id,
			user_id,
			username,
			role,
			action,
			success,    -- 0/1
			ip,         -- nullable
			method,     -- nullable
			status,     -- nullable
			timestamp
		FROM user_audit_logs
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	normalizeLoopback := func(ip string) string {
		if ip == "::1" {
			return "127.0.0.1"
		}
		return ip
	}

	var logs []model.GetUserAuditLogsResponse
	for rows.Next() {
		var (
			item       model.GetUserAuditLogsResponse
			successInt int
			ipNS       sql.NullString
			methodNS   sql.NullString
			statusNI64 sql.NullInt64
		)

		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Username,
			&item.Role,
			&item.Action,
			&successInt,
			&ipNS,
			&methodNS,
			&statusNI64,
			&item.Timestamp,
		); err != nil {
			return nil, err
		}

		// 映射/归一化
		item.Success = (successInt == 1)
		if ipNS.Valid {
			item.IP = normalizeLoopback(ipNS.String)
		}
		if methodNS.Valid {
			item.Method = methodNS.String
		}
		if statusNI64.Valid {
			item.Status = int(statusNI64.Int64)
		}

		logs = append(logs, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return logs, nil
}



// =======================================================================
// ✅ InsertUserAuditLog：用户审计日志插入函数
//
// 参数：
//   - id：自增id
//   - userID：操作用户 ID
//   - username：执行操作的用户名
//   - role：用户角色
//   - action：操作内容
//   - success：操作是否成功
//   - timestamp：操作发生时间(服务器事件)
// 返回 error：执行成功返回 nil，否则返回错误信息
// =======================================================================
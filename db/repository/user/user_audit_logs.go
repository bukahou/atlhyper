package user

import (
	"NeuroController/db/utils"
	"NeuroController/model"
)

// =======================================================================
// ✅ GetUserAuditLogs：查询所有用户审计日志
//
// 参数：
//   - 不需要输入任何参数获取全部日志
// 返回 error：执行成功返回 nil，否则返回错误信息
// =======================================================================

func GetUserAuditLogs() ([]model.GetUserAuditLogsResponse, error) {
	//查询函数
	rows, err := utils.DB.Query(`
		SELECT id, user_id, username, role, action, success, timestamp
		FROM user_audit_logs ORDER BY timestamp DESC
		`)
	
	if err !=nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.GetUserAuditLogsResponse

	for rows.Next() {
		var log model.GetUserAuditLogsResponse
		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Username,
			&log.Role,
			&log.Action,
			&log.Success,
			&log.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

		return logs,nil 

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
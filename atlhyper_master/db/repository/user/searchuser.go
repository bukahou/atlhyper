package user

import (
	"AtlHyper/atlhyper_master/db/utils"
	"AtlHyper/atlhyper_master/model"
	"database/sql"
	"time"
)

// =======================================================================
// ✅ GetAllUsers：查询所有用户信息
//
// 功能说明：
//   - 连接 SQLite 数据库，读取 users 表中所有用户的基本信息
//   - 不返回 password_hash 等敏感字段
//   - 结果按用户 ID 升序排序
//
// 返回值：
//   - []model.User：用户信息切片（不包含密码）
//   - error：查询出错时返回错误
//
// 使用示例：
//   users, err := GetAllUsers()
// =======================================================================
func GetAllUsers() ([]model.User, error) {
	// 🔍 执行查询，排除敏感字段（password_hash）
	rows, err := utils.DB.Query(`
		SELECT id, username, display_name, email, role, created_at, last_login
		FROM users ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User

	// 📦 遍历查询结果，逐行解析
	for rows.Next() {
		var u model.User                        // 用户结构体
		var createdAtStr, lastLoginStr sql.NullString // 用于接收字符串时间（兼容 NULL）

		// 🧩 绑定每一行数据到变量（注意顺序要匹配 SQL 字段）
		err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.DisplayName,
			&u.Email,
			&u.Role,
			&createdAtStr,
			&lastLoginStr,
		)
		if err != nil {
			return nil, err
		}

		// 🕒 解析创建时间字符串为 time.Time 类型
		if createdAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, createdAtStr.String)
			u.CreatedAt = t
		}

		// 🕒 解析上次登录时间（可为空）
		if lastLoginStr.Valid {
			t, _ := time.Parse(time.RFC3339, lastLoginStr.String)
			u.LastLogin = &t
		}

		// 📥 添加到用户列表
		users = append(users, u)
	}

	// ✅ 返回用户列表
	return users, nil
}

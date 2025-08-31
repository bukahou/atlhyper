package user

import (
	"AtlHyper/atlhyper_master/db/utils"
	"AtlHyper/model"

	"golang.org/x/crypto/bcrypt"
)

// ===========================================
// ✅ User：用户数据模型结构体
// ===========================================
// 对应 SQLite 表 users 的字段结构，用于查询与认证
// type User struct {
// 	ID           int    // 主键，自增
// 	Username     string // 登录用户名，唯一
// 	PasswordHash string // 加密后的密码（bcrypt）
// 	DisplayName  string // 显示名（可选）
// 	Email        string // 邮箱地址（可选）
// 	Role         int    // 用户角色（如 1=普通用户，3=管理员）
// }

// =================================================
// ✅ GetUserByUsername：根据用户名查询用户信息
// =================================================
// - 参数：用户名（string）
// - 返回：用户结构体指针 + 错误信息
// - 说明：使用全局 SQLite 连接（utils.DB）查询用户表
func GetUserByUsername(username string) (*model.User, error) {
	row := utils.DB.QueryRow(`
		SELECT id, username, password_hash, display_name, email, role
		FROM users WHERE username = ?`, username)

	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName, &u.Email, &u.Role)
	if err != nil {
		return nil, err // 未找到或扫描失败
	}
	return &u, nil
}

// ==============================================
// ✅ CheckPassword：校验明文密码与加密哈希是否匹配
// ==============================================
// - 参数：用户输入的明文密码、数据库中的 hash
// - 返回：true 表示匹配成功，false 表示密码错误
func CheckPassword(plain, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

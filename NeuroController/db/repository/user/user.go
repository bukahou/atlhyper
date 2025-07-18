package user

import (
	"NeuroController/db/utils"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ============================================================
// ✅ EnsureAdminUser：初始化默认管理员账户
// ============================================================
// 功能：
// - 如果用户表为空（首次启动或数据库为空），则自动创建一个默认管理员账户。
// - 避免首次登录时因无用户导致系统不可用。
// - 使用全局 SQLite 连接（utils.DB）执行 SQL 操作。
//
// 默认账户信息：
// - 用户名：wuxiafeng
// - 密码：wuxiafeng（⚠️ 建议生产中从环境变量读取并设置强密码）
func EnsureAdminUser() error {
	// 1️⃣ 查询当前 users 表中的记录数量
	row := utils.DB.QueryRow(`SELECT COUNT(*) FROM users`)
	var count int
	if err := row.Scan(&count); err != nil {
		return err // 查询失败，返回错误
	}
	if count > 0 {
		// 如果已有用户，则跳过管理员初始化
		log.Println("ℹ️ 用户表已存在用户，跳过管理员初始化")
		return nil
	}

	// 2️⃣ 构造默认用户信息
	username := "admin"
	password := "admin"
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err // 密码加密失败
	}

	// 3️⃣ 插入默认用户记录
	_, err = utils.DB.Exec(`
		INSERT INTO users (username, password_hash, display_name, email, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		username,
		string(hashed),
		"武夏锋",              // 显示名
		"",                     // 邮箱为空
		3,                      // 管理员权限标识（例如 3）
		time.Now().Format(time.RFC3339), // 创建时间
	)
	if err != nil {
		return err // 插入失败
	}

	log.Println("✅ 默认管理员已创建: 用户名 admin / 密码 admin")
	return nil
}

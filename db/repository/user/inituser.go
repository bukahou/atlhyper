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
		"atlhyper",              // 显示名
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

// InsertTestAuditLog 插入一条用于测试的审计记录
func InsertTestAuditLog() error {
	 //先查询是否有数据
	 row := utils.DB.QueryRow(`SELECT COUNT(*) FROM user_audit_logs`)

	 var count int

	 //获取是是数据是否成功
	 if err := row.Scan(&count); err != nil {
		log.Panicf("查询用户审计日志失败: %v", err)
	}

	//如果存在数据则不再插入测试记录
	if count > 0 {
		log.Println("ℹ️ 用户审计日志已存在，跳过插入测试记录")
		return nil
	}

	// 插入一条测试审计记录
	_, err := utils.DB.Exec(`
		INSERT INTO user_audit_logs (user_id, username, role, action, success)
		VALUES (?, ?, ?, ?, ?)`,
		1,                      // 假设用户ID为1
		"wuxiafeng", //用户名
		3,                      // 管理员角色
		"restart pod", // 操作描述
		1,                      // 成功标识（1表示成功）
	)
	if err != nil {
		log.Println("测试数据插入失败:", err)
	}
	log.Println("✅ 测试审计记录已插入")

	return nil
}
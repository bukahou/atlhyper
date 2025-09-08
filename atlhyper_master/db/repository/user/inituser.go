package user

import (
	"AtlHyper/atlhyper_master/db/utils"
	"AtlHyper/config"
	"log"
	"os"
	"strconv"
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
func EnsureAdminUser() error {
	// 1️⃣ 查询当前 users 表中的记录数量
	row := utils.DB.QueryRow(`SELECT COUNT(*) FROM users`)
	var count int
	if err := row.Scan(&count); err != nil {
		return err // 查询失败，返回错误
	}
	if count > 0 {
		log.Println("ℹ️ 用户表已存在用户，跳过管理员初始化")
		return nil
	}

	// 2️⃣ 构造默认用户信息
	username := config.GlobalConfig.Admin.Username
	password := config.GlobalConfig.Admin.Password
	displayName := config.GlobalConfig.Admin.DisplayName
	email := config.GlobalConfig.Admin.Email

	// 🔐 加密密码
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err // 密码加密失败
	}

	// 3️⃣ 解析权限等级（从字符串转为 int）
	roleStr := config.GlobalConfig.Admin.Role
	role := 1 // 默认：普通用户
	if parsed, err := strconv.Atoi(roleStr); err == nil {
		role = parsed
	} else {
		log.Printf("⚠️ DEFAULT_ADMIN_ROLE=%q 无法解析为整数，使用默认值 role=1", roleStr)
	}

	// 4️⃣ 插入默认用户记录
	_, err = utils.DB.Exec(`
		INSERT INTO users (username, password_hash, display_name, email, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		username,
		string(hashed),
		displayName,
		email,
		role,
		time.Now().Format(time.RFC3339),
	)
	if err != nil {
		return err // 插入失败
	}

	log.Printf("✅ 默认管理员已创建: 用户名 %s / 密码 %s / 角色 %d", username, password, role)
	return nil
}

func EnsureDefaultUsers() error {
	countStr := os.Getenv("DEFAULT_USER_COUNT")
	if countStr == "" {
		log.Println("ℹ️ 未设置 DEFAULT_USER_COUNT，跳过默认用户初始化")
		return nil
	}
	n, err := strconv.Atoi(countStr)
	if err != nil || n <= 0 {
		log.Printf("⚠️ DEFAULT_USER_COUNT=%q 无效，跳过初始化", countStr)
		return nil
	}

	for i := 1; i <= n; i++ {
		prefix := "USER_" + strconv.Itoa(i) + "_"

		username := os.Getenv(prefix + "USERNAME")
		password := os.Getenv(prefix + "PASSWORD")
		displayName := os.Getenv(prefix + "DISPLAY_NAME")
		email := os.Getenv(prefix + "EMAIL")
		roleStr := os.Getenv(prefix + "ROLE")

		// 👀 跳过字段不完整的用户
		if username == "" || password == "" || email == "" {
			log.Printf("⚠️ 用户 %d 信息不完整（用户名/密码/邮箱缺失），跳过", i)
			continue
		}

		// 检查是否已存在该用户名
		row := utils.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username)
		var count int
		if err := row.Scan(&count); err != nil {
			log.Printf("❌ 查询用户 %q 失败: %v", username, err)
			continue
		}
		if count > 0 {
			log.Printf("ℹ️ 用户 %q 已存在，跳过创建", username)
			continue
		}

		// 解析角色
		role := 1 // 默认普通用户
		if parsed, err := strconv.Atoi(roleStr); err == nil {
			role = parsed
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("❌ 加密密码失败（用户 %q）: %v", username, err)
			continue
		}

		_, err = utils.DB.Exec(`
			INSERT INTO users (username, password_hash, display_name, email, role, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			username,
			string(hashed),
			displayName,
			email,
			role,
			time.Now().Format(time.RFC3339),
		)
		if err != nil {
			log.Printf("❌ 创建用户 %q 失败: %v", username, err)
			continue
		}

		log.Printf("✅ 创建用户 %d: 用户名=%q，角色=%d", i, username, role)
	}
	return nil
}



// ============================================================
// ✅ EnsureAdminTodo：初始化默认代办事项
// ============================================================
// 功能：
// - 检查 todos 表中是否已有 admin 用户的待办事项。
// - 如果没有，则插入一条默认待办任务，用于开发验证。
func EnsureAdminTodo() error {
	// 1️⃣ 检查 admin 是否已有待办
	row := utils.DB.QueryRow(`SELECT COUNT(*) FROM todos WHERE username = ?`, "admin")
	var count int
	if err := row.Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		log.Println("ℹ️ admin 已存在待办事项，跳过初始化")
		return nil
	}

	// 2️⃣ 插入一条默认待办事项
	_, err := utils.DB.Exec(`
		INSERT INTO todos (username, title, content, created_at, is_done, priority, category, deleted)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"admin",                                   // username
		"欢迎使用AtlHyper",                         // title
		"这是系统自动生成的第一条代办事项",           // content
		time.Now().Format("2006-01-02 15:04:05"),  // created_at
		0,                                         // is_done
		1,                                         // priority
		"系统初始化",                                // category
		0,                                         // deleted
	)
	if err != nil {
		log.Printf("⚠️ 默认代办事项初始化失败: %v", err)
		return err
	}

	log.Println("✅ 默认代办事项已创建 (用户名=admin)")
	return nil
}
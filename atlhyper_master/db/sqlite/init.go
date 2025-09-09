package sqlite

import (
	"AtlHyper/atlhyper_master/db/repository/config"
	"AtlHyper/atlhyper_master/db/repository/user"
	"AtlHyper/atlhyper_master/db/utils"
	"database/sql"
	"log"
	"os"
	"path/filepath"
)

// ============================================================
// ✅ InitDB：初始化 SQLite 数据库（目录 + 连接 + 表结构 + 管理员）
// ============================================================
// 注意：
// - 使用 utils.DB 作为全局数据库连接对象
// - 所有数据库操作均统一访问 utils.DB
func InitDB() {
	// 1️⃣ 创建数据库文件所在目录（如 db/data/）
	err := os.MkdirAll(filepath.Dir(utils.DBPath), 0755)
	if err != nil {
		log.Fatalf("创建数据库目录失败: %v", err)
	}

	// 2️⃣ 建立 SQLite 连接，并赋值给全局 utils.DB
	utils.DB, err = sql.Open("sqlite3", utils.DBPath)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 3️⃣ 创建所有表结构（如 users、event_logs 等）
	if err = CreateTables(); err != nil {
		log.Fatalf("表结构创建失败: %v", err)
	}

	// 4️⃣ 插入默认管理员（仅在用户表为空时创建）
	if err = user.EnsureAdminUser(); err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}
	// 5️⃣ 插入默认普通用户（仅在用户表为空时创建）
	err = user.EnsureDefaultUsers()
	err = user.EnsureAdminTodo()
	if err != nil {
		log.Fatalf("初始化默认用户失败: %v", err)
	}

	// 6️⃣ 初始化配置表（仅在 config 表无记录时创建）
	//    包括：Slack 警报的启用状态、Webhook URL、发送间隔等
	//    相关环境变量：ENABLE_SLACK_ALERT, SLACK_WEBHOOK_URL, SLACK_DISPATCH_INTERVAL_SEC
	if err = config.InitConfigTables(); err != nil {
		log.Fatalf("slack初始化配置表失败: %v", err)
	}

	log.Println("✅ SQLite 数据库初始化完成")
}

// ============================================================
// ✅ Open：提供非全局连接方式（供临时用途使用）
// ============================================================
// 示例：tx, _ := sqlite.Open() 后自行 Close()
func Open() (*sql.DB, error) {
	return sql.Open("sqlite3", utils.DBPath)
}

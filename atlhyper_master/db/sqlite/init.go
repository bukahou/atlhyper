package sqlite

import (
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
	// 5️⃣ 插入用户审计测试数据
	err = user.EnsureDefaultUsers()
	if err != nil {
		log.Fatalf("初始化默认用户失败: %v", err)
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

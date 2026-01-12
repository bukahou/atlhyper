// store/db.go
// 全局数据库连接管理
package store

import (
	"database/sql"
	"errors"
)

// DB 全局数据库连接
var DB *sql.DB

// SetDB 设置全局数据库连接
func SetDB(db *sql.DB) {
	DB = db
}

// GetDB 获取全局数据库连接
func GetDB() *sql.DB {
	return DB
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// Validate 检查数据库连接是否已初始化
func Validate() error {
	if DB == nil {
		return errors.New("database connection not initialized")
	}
	return nil
}

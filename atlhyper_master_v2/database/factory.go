// atlhyper_master_v2/database/factory.go
// 数据库工厂函数
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Config 数据库配置
type Config struct {
	Type string // sqlite / mysql / postgres
	Path string // SQLite 文件路径
}

// New 创建数据库实例
// 打开连接并执行迁移，Repository 通过 repo.Init() 注入
func New(cfg Config, dialect Dialect) (*DB, error) {
	var conn *sql.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		conn, err = openSQLite(cfg.Path)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 执行迁移
	if err := dialect.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Printf("[Database] 已连接数据库: type=%s", cfg.Type)
	return &DB{Conn: conn}, nil
}

// openSQLite 打开 SQLite 数据库连接
func openSQLite(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

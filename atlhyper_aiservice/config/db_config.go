// atlhyper_aiservice/config/db_config.go
package config

import (
	"errors"
	"os"
)

// =========================================
// 🧠 数据库（PostgreSQL / pgvector）配置
// =========================================
type DBConfig struct {
	URI string // PostgreSQL 连接字符串，例如：postgres://user:pass@host:5432/dbname
}

const (
	envDBURI = "PG_URI"
)

// loadDBConfig —— 加载数据库配置（必须从环境变量中获取）
func loadDBConfig() (DBConfig, error) {
	var c DBConfig

	if uri := os.Getenv(envDBURI); uri != "" {
		c.URI = uri
		return c, nil
	}
	return c, errors.New("PG_URI 未设置，请通过环境变量提供数据库连接字符串")
}

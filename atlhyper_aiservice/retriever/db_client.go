// retriever/db_client.go
package retriever

import (
	"AtlHyper/atlhyper_aiservice/config"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

// InitDB —— 初始化 PostgreSQL 连接池
// 连接信息来自环境变量 PG_URI（已在 config.DBConfig 中加载）
func InitDB(ctx context.Context) error {
	cfg := config.GetDBConfig()
	if cfg.URI == "" {
		return fmt.Errorf("PG_URI 未设置，请通过环境变量提供数据库连接字符串")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.URI)
	if err != nil {
		return fmt.Errorf("解析数据库配置失败: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	dbPool = pool
	return nil
}

// GetDB —— 获取连接池实例
func GetDB() *pgxpool.Pool {
	if dbPool == nil {
		panic("数据库未初始化，请先调用 InitDB()")
	}
	return dbPool
}

// CloseDB —— 关闭连接池（优雅退出时使用）
func CloseDB() {
	if dbPool != nil {
		dbPool.Close()
	}
}

// Package clickhouse 实现 ClickHouse 客户端
package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"AtlHyper/atlhyper_agent_v2/sdk"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// client ClickHouse 客户端实现
type client struct {
	db *sql.DB
}

// NewClient 创建 ClickHouse 客户端
//
// 参数:
//   - endpoint: ClickHouse 地址，如 "clickhouse://localhost:9000"
//   - database: 数据库名
//   - timeout: 连接超时
func NewClient(endpoint, database string, timeout time.Duration) (sdk.ClickHouseClient, error) {
	dsn := fmt.Sprintf("%s/%s?dial_timeout=%s&read_timeout=%s",
		endpoint, database, timeout.String(), timeout.String())

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute)

	return &client{db: db}, nil
}

func (c *client) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

func (c *client) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

func (c *client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

func (c *client) Close() error {
	return c.db.Close()
}

package ch

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// sqliteClient 用 SQLite 实现 ClickHouseClient：ClickHouse 方言的查询在这里一律解析失败，
// 正好用来检查「查询失败」有没有被当成「结果是 0」吞掉
type sqliteClient struct{ db *sql.DB }

func (c sqliteClient) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, q, args...)
}
func (c sqliteClient) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return c.db.QueryRowContext(ctx, q, args...)
}
func (c sqliteClient) Ping(ctx context.Context) error { return c.db.PingContext(ctx) }
func (c sqliteClient) Close() error                   { return c.db.Close() }

func failingClient(t *testing.T) sqliteClient {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "x.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return sqliteClient{db: db}
}

// 概览查询失败要返回错误：调用方据此沿用上一份数据；吞掉就会显示成「0 个节点、CPU 0%」
func TestSummaryQueries_ReturnErrors(t *testing.T) {
	repo := NewOTelSummaryRepository(failingClient(t))
	ctx := context.Background()
	if _, _, _, _, _, err := repo.GetAPMSummary(ctx); err == nil {
		t.Error("GetAPMSummary 查询失败应返回错误")
	}
	if _, _, err := repo.GetSLOSummary(ctx); err == nil {
		t.Error("GetSLOSummary 查询失败应返回错误")
	}
	if _, _, _, _, _, err := repo.GetMetricsSummary(ctx); err == nil {
		t.Error("GetMetricsSummary 查询失败应返回错误")
	}
}

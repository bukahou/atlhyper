// atlhyper_master_v2/database/repo/notify_query.go
// 通知各仓库共用的查询与执行辅助
package repo

import (
	"context"
	"database/sql"
)

// queryNotifyRows 执行查询并逐行扫描；结果恒为非 nil 切片
func queryNotifyRows[T any](ctx context.Context, db *sql.DB, scan func(*sql.Rows) (T, error), query string, args ...any) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []T{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// queryNotifyRow 取查询结果的第一行；没有行时 found 为假
func queryNotifyRow[T any](ctx context.Context, db *sql.DB, scan func(*sql.Rows) (T, error), query string, args ...any) (item T, found bool, err error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return item, false, err
	}
	defer rows.Close()

	if !rows.Next() {
		return item, false, rows.Err()
	}
	item, err = scan(rows)
	return item, err == nil, err
}

// queryNotifyCount 执行 COUNT 查询
func queryNotifyCount(ctx context.Context, db *sql.DB, query string, args ...any) (int, error) {
	var total int
	err := db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}

// execNotifyAffected 执行写语句并返回受影响的行数
func execNotifyAffected(ctx context.Context, db *sql.DB, query string, args ...any) (int64, error) {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

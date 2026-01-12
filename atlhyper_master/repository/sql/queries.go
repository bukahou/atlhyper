// repository/sql/queries.go
//
// SQL 语句全局变量
//
// 本文件定义了全局 SQL 语句实例 Q，所有业务逻辑代码通过 Q 访问 SQL 语句。
// 这种设计实现了 SQL 语句与业务逻辑的解耦，支持多数据库切换。
//
// 设计思路:
//   - Q 是一个指向 schema.Queries 的全局指针
//   - 初始化时根据数据库类型设置 Q 指向对应的 SQL 语句集
//   - 业务代码统一使用 Q.User.xxx、Q.Audit.xxx 等方式访问 SQL
//
// 使用示例:
//
//	// 在 init.go 中初始化
//	sql.SetQueries(sqlite.Queries)  // 使用 SQLite
//	sql.SetQueries(pg.Queries)      // 使用 PostgreSQL
//
//	// 在业务代码中使用
//	row := db.QueryRowContext(ctx, sql.Q.User.GetByID, userID)
package sql

import "AtlHyper/atlhyper_master/repository/sql/schema"

// Q 是全局 SQL 语句实例
// 在应用启动时通过 SetQueries 设置
// 所有 repository/sql 包下的业务逻辑代码都通过此变量访问 SQL 语句
//
// 注意: 必须在使用前调用 SetQueries 初始化，否则会导致空指针异常
var Q *schema.Queries

// SetQueries 设置当前使用的 SQL 语句集
// 根据数据库类型传入对应的 Queries 实例:
//   - sqlite.Queries: SQLite 数据库
//   - pg.Queries: PostgreSQL 数据库
//
// 参数:
//   - q: SQL 语句集合实例
//
// 调用时机:
//   - 应用启动时，在 Init() 函数中调用
//   - 必须在任何数据库操作之前完成
func SetQueries(q *schema.Queries) {
	Q = q
}

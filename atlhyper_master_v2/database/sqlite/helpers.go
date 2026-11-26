// atlhyper_master_v2/database/sqlite/helpers.go
// SQLite 方言层辅助函数
package sqlite

import (
	"errors"

	sqlitedriver "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// boolToInt 将 bool 转为 SQLite INTEGER (0/1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// violatesConstraint 写入失败是否因为某一类约束（SQLite 的扩展结果码，驱动默认开启扩展结果码）。
// 按结果码判断，不看错误文字
func violatesConstraint(err error, extendedCode int) bool {
	var driverErr *sqlitedriver.Error
	return errors.As(err, &driverErr) && driverErr.Code() == extendedCode
}

// violatesUnique 唯一约束冲突
func violatesUnique(err error) bool {
	return violatesConstraint(err, sqlite3.SQLITE_CONSTRAINT_UNIQUE)
}

// violatesForeignKey 外键约束冲突（指向的行不存在）
func violatesForeignKey(err error) bool {
	return violatesConstraint(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY)
}

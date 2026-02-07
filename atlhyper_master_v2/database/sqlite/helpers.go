// atlhyper_master_v2/database/sqlite/helpers.go
// SQLite 方言层辅助函数
package sqlite

// boolToInt 将 bool 转为 SQLite INTEGER (0/1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

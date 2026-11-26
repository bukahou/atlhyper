// atlhyper_master_v2/database/sqlite/service_account.go
// SQLite ServiceAccountDialect 实现：users 表里 kind = service 的行
//
// 每条语句都带 kind 条件：拿人的 id 调用等同于不存在。列与行扫描与 user.go 共用。
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type serviceAccountDialect struct {
	users userDialect
}

func (d *serviceAccountDialect) Insert(account *database.User, now time.Time) (string, []any) {
	stamp := utcStamp(now)
	return `INSERT INTO users (username, password_hash, display_name, email, role, status, kind, created_at, updated_at)
	VALUES (?, ?, ?, '', ?, ?, ?, ?, ?)`,
		[]any{account.Username, database.ServiceAccountPasswordHash, account.DisplayName, account.Role, account.Status,
			database.UserKindService, stamp, stamp}
}

func (d *serviceAccountDialect) SelectAll() (string, []any) {
	return "SELECT " + userColumns + " FROM users WHERE kind = ? ORDER BY username", []any{database.UserKindService}
}

func (d *serviceAccountDialect) SelectByID(id int64) (string, []any) {
	return "SELECT " + userColumns + " FROM users WHERE id = ? AND kind = ?", []any{id, database.UserKindService}
}

func (d *serviceAccountDialect) Update(account *database.User, now time.Time) (string, []any) {
	return `UPDATE users SET display_name = ?, role = ?, status = ?, updated_at = ? WHERE id = ? AND kind = ?`,
		[]any{account.DisplayName, account.Role, account.Status, utcStamp(now), account.ID, database.UserKindService}
}

func (d *serviceAccountDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM users WHERE id = ? AND kind = ?", []any{id, database.UserKindService}
}

func (d *serviceAccountDialect) ScanRow(rows *sql.Rows) (*database.User, error) {
	return d.users.ScanRow(rows)
}

// IsDuplicateUsername users 表上只有用户名一个唯一约束（id 自增），唯一约束冲突即用户名被占
func (d *serviceAccountDialect) IsDuplicateUsername(err error) bool {
	return violatesUnique(err)
}

var _ database.ServiceAccountDialect = (*serviceAccountDialect)(nil)

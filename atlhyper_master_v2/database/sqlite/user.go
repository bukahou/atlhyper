// atlhyper_master_v2/database/sqlite/user.go
// SQLite UserDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type userDialect struct{}

const userColumns = `id, username, password_hash, display_name, email, role, status, created_at, updated_at, last_login_at, last_login_ip`

func (d *userDialect) Insert(user *database.User) (string, []any) {
	now := time.Now().Format(time.RFC3339)
	query := `INSERT INTO users (username, password_hash, display_name, email, role, status, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{user.Username, user.PasswordHash, user.DisplayName, user.Email, user.Role, user.Status, now, now}
	return query, args
}

func (d *userDialect) Update(user *database.User) (string, []any) {
	query := `UPDATE users SET display_name = ?, email = ?, role = ?, status = ?, updated_at = ? WHERE id = ?`
	args := []any{user.DisplayName, user.Email, user.Role, user.Status, time.Now().Format(time.RFC3339), user.ID}
	return query, args
}

func (d *userDialect) Delete(id int64) (string, []any) {
	return "DELETE FROM users WHERE id = ?", []any{id}
}

func (d *userDialect) SelectByID(id int64) (string, []any) {
	return "SELECT " + userColumns + " FROM users WHERE id = ?", []any{id}
}

func (d *userDialect) SelectByUsername(username string) (string, []any) {
	return "SELECT " + userColumns + " FROM users WHERE username = ?", []any{username}
}

func (d *userDialect) SelectAll() (string, []any) {
	return "SELECT " + userColumns + " FROM users", nil
}

func (d *userDialect) UpdateLastLogin(id int64, ip string) (string, []any) {
	return "UPDATE users SET last_login_at = ?, last_login_ip = ? WHERE id = ?",
		[]any{time.Now().Format(time.RFC3339), ip, id}
}

func (d *userDialect) ScanRow(rows *sql.Rows) (*database.User, error) {
	user := &database.User{}
	var createdAt, updatedAt string
	var displayName, email sql.NullString
	var lastLoginAt, lastLoginIP sql.NullString
	err := rows.Scan(
		&user.ID, &user.Username, &user.PasswordHash, &displayName, &email,
		&user.Role, &user.Status, &createdAt, &updatedAt, &lastLoginAt, &lastLoginIP,
	)
	if err != nil {
		return nil, err
	}
	user.DisplayName = displayName.String
	user.Email = email.String
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if lastLoginAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastLoginAt.String)
		user.LastLoginAt = &t
	}
	user.LastLoginIP = lastLoginIP.String
	return user, nil
}

var _ database.UserDialect = (*userDialect)(nil)

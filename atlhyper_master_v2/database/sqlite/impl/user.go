// atlhyper_master_v2/database/sqlite/impl/user.go
// UserRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *repository.User) error {
	now := time.Now().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (username, password_hash, display_name, email, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username, user.PasswordHash, user.DisplayName, user.Email, user.Role, user.Status, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

func (r *UserRepository) Update(ctx context.Context, user *repository.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = ?, email = ?, role = ?, status = ?, updated_at = ? WHERE id = ?`,
		user.DisplayName, user.Email, user.Role, user.Status, time.Now().Format(time.RFC3339), user.ID,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	return err
}

// userColumns 用户表列名（避免 SELECT * 顺序依赖）
const userColumns = `id, username, password_hash, display_name, email, role, status, created_at, updated_at, last_login_at, last_login_ip`

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*repository.User, error) {
	return r.scanOne(ctx, "SELECT "+userColumns+" FROM users WHERE id = ?", id)
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*repository.User, error) {
	return r.scanOne(ctx, "SELECT "+userColumns+" FROM users WHERE username = ?", username)
}

func (r *UserRepository) List(ctx context.Context) ([]*repository.User, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+userColumns+" FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*repository.User
	for rows.Next() {
		user, err := r.scanRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, id int64, ip string) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE users SET last_login_at = ?, last_login_ip = ? WHERE id = ?",
		time.Now().Format(time.RFC3339), ip, id,
	)
	return err
}

func (r *UserRepository) scanOne(ctx context.Context, query string, args ...interface{}) (*repository.User, error) {
	row := r.db.QueryRowContext(ctx, query, args...)
	user := &repository.User{}
	var createdAt, updatedAt string
	var displayName, email sql.NullString
	var lastLoginAt, lastLoginIP sql.NullString
	err := row.Scan(
		&user.ID, &user.Username, &user.PasswordHash, &displayName, &email,
		&user.Role, &user.Status, &createdAt, &updatedAt, &lastLoginAt, &lastLoginIP,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
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

func (r *UserRepository) scanRow(rows *sql.Rows) (*repository.User, error) {
	user := &repository.User{}
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

var _ repository.UserRepository = (*UserRepository)(nil)

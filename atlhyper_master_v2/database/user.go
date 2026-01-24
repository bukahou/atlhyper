// atlhyper_master_v2/database/user.go
// UserRepository 实现
package database

import (
	"context"
	"database/sql"
)

type userRepo struct {
	db      *sql.DB
	dialect UserDialect
}

func newUserRepo(db *sql.DB, dialect UserDialect) *userRepo {
	return &userRepo{db: db, dialect: dialect}
}

func (r *userRepo) Create(ctx context.Context, user *User) error {
	query, args := r.dialect.Insert(user)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

func (r *userRepo) Update(ctx context.Context, user *User) error {
	query, args := r.dialect.Update(user)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*User, error) {
	query, args := r.dialect.SelectByID(id)
	return r.queryOne(ctx, query, args...)
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*User, error) {
	query, args := r.dialect.SelectByUsername(username)
	return r.queryOne(ctx, query, args...)
}

func (r *userRepo) List(ctx context.Context) ([]*User, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, id int64, ip string) error {
	query, args := r.dialect.UpdateLastLogin(id, ip)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *userRepo) queryOne(ctx context.Context, query string, args ...any) (*User, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	return r.dialect.ScanRow(rows)
}

// atlhyper_master_v2/database/repo/user.go
// UserRepository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type userRepo struct {
	db      *sql.DB
	dialect database.UserDialect
}

func newUserRepo(db *sql.DB, dialect database.UserDialect) *userRepo {
	return &userRepo{db: db, dialect: dialect}
}

func (r *userRepo) Create(ctx context.Context, user *database.User) error {
	query, args := r.dialect.Insert(user)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

func (r *userRepo) Update(ctx context.Context, user *database.User) error {
	query, args := r.dialect.Update(user)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *userRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*database.User, error) {
	query, args := r.dialect.SelectByID(id)
	return r.queryOne(ctx, query, args...)
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*database.User, error) {
	query, args := r.dialect.SelectByUsername(username)
	return r.queryOne(ctx, query, args...)
}

func (r *userRepo) List(ctx context.Context) ([]*database.User, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*database.User
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

func (r *userRepo) queryOne(ctx context.Context, query string, args ...any) (*database.User, error) {
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

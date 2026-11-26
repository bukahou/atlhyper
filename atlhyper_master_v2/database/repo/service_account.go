// atlhyper_master_v2/database/repo/service_account.go
// ServiceAccountRepository 实现（users 表里 kind = service 的行）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type serviceAccountRepo struct {
	db      *sql.DB
	dialect database.ServiceAccountDialect
}

func newServiceAccountRepo(db *sql.DB, dialect database.ServiceAccountDialect) *serviceAccountRepo {
	return &serviceAccountRepo{db: db, dialect: dialect}
}

func (r *serviceAccountRepo) Create(ctx context.Context, account *database.User) error {
	query, args := r.dialect.Insert(account, time.Now())
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if r.dialect.IsDuplicateUsername(err) {
			return database.ErrUsernameTaken
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	account.ID = id
	account.Kind = database.UserKindService
	account.PasswordHash = database.ServiceAccountPasswordHash
	return nil
}

func (r *serviceAccountRepo) List(ctx context.Context) ([]*database.User, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []*database.User{}
	for rows.Next() {
		account, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (r *serviceAccountRepo) GetByID(ctx context.Context, id int64) (*database.User, error) {
	query, args := r.dialect.SelectByID(id)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, rows.Err() // 没有行是 nil, nil；读取出错要报出来
	}
	return r.dialect.ScanRow(rows)
}

func (r *serviceAccountRepo) Update(ctx context.Context, account *database.User) error {
	query, args := r.dialect.Update(account, time.Now())
	return r.execOnServiceAccount(ctx, query, args...)
}

func (r *serviceAccountRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	return r.execOnServiceAccount(ctx, query, args...)
}

// execOnServiceAccount 执行一条针对单个服务账号的写语句；没有命中任何行说明不存在（或不是服务账号）
func (r *serviceAccountRepo) execOnServiceAccount(ctx context.Context, query string, args ...any) error {
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrServiceAccountNotFound
	}
	return nil
}

var _ database.ServiceAccountRepository = (*serviceAccountRepo)(nil)

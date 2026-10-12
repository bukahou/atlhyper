// atlhyper_master_v2/database/repo/user_identity.go
// UserIdentityRepository 实现
package repo

import (
	"context"
	"database/sql"
	"errors"

	"AtlHyper/atlhyper_master_v2/database"
)

type userIdentityRepo struct {
	db      *sql.DB
	users   database.UserDialect
	dialect database.UserIdentityDialect
}

func newUserIdentityRepo(db *sql.DB, users database.UserDialect, dialect database.UserIdentityDialect) *userIdentityRepo {
	return &userIdentityRepo{db: db, users: users, dialect: dialect}
}

func (r *userIdentityRepo) FindUserID(ctx context.Context, issuer, subject string) (int64, bool, error) {
	query, args := r.dialect.SelectUserID(issuer, subject)
	var id int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (r *userIdentityRepo) CreateUserWithIdentity(ctx context.Context, user *database.User, issuer, subject string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // 提交后为空操作

	query, args := r.users.Insert(user)
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return r.dialect.ClassifyInsertError(err)
	}
	userID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	query, args = r.dialect.Insert(userID, issuer, subject)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return r.dialect.ClassifyInsertError(err)
	}
	if err := tx.Commit(); err != nil {
		return r.dialect.ClassifyInsertError(err)
	}
	user.ID = userID
	return nil
}

func (r *userIdentityRepo) ListFederatedUserIDs(ctx context.Context) (map[int64]bool, error) {
	query, args := r.dialect.SelectFederatedUserIDs()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

func (r *userIdentityRepo) DeleteByUserID(ctx context.Context, userID int64) error {
	query, args := r.dialect.DeleteByUserID(userID)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

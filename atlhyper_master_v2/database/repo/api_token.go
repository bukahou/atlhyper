// atlhyper_master_v2/database/repo/api_token.go
// APITokenRepository 实现（访问令牌：只存哈希与显示用前缀）
package repo

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type apiTokenRepo struct {
	db      *sql.DB
	dialect database.APITokenDialect
}

func newAPITokenRepo(db *sql.DB, dialect database.APITokenDialect) *apiTokenRepo {
	return &apiTokenRepo{db: db, dialect: dialect}
}

func (r *apiTokenRepo) CreateWithinLimit(ctx context.Context, token *database.APIToken, limit int) error {
	query, args := r.dialect.InsertWithinLimit(token, limit)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if r.dialect.IsUnknownAccount(err) {
			return database.ErrServiceAccountNotFound
		}
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		return database.ErrAPITokenLimitReached
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	token.ID = id
	return nil
}

func (r *apiTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*database.APIToken, error) {
	// 空哈希不得匹配到任何令牌
	if tokenHash == "" {
		return nil, nil
	}
	query, args := r.dialect.SelectByHash(tokenHash)
	tokens, err := r.query(ctx, query, args...)
	if err != nil || len(tokens) == 0 {
		return nil, err
	}
	return tokens[0], nil
}

func (r *apiTokenRepo) List(ctx context.Context) ([]*database.APIToken, error) {
	query, args := r.dialect.SelectAll()
	return r.query(ctx, query, args...)
}

func (r *apiTokenRepo) ListByUser(ctx context.Context, userID int64) ([]*database.APIToken, error) {
	query, args := r.dialect.SelectByUser(userID)
	return r.query(ctx, query, args...)
}

func (r *apiTokenRepo) Revoke(ctx context.Context, userID, tokenID int64, at time.Time) error {
	query, args := r.dialect.Revoke(userID, tokenID, at)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return database.ErrAPITokenNotFound
	}
	return nil
}

func (r *apiTokenRepo) Touch(ctx context.Context, id int64, usedAt time.Time) error {
	query, args := r.dialect.Touch(id, usedAt)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// query 执行查询并扫描全部行；没有行时返回空切片
func (r *apiTokenRepo) query(ctx context.Context, query string, args ...any) ([]*database.APIToken, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []*database.APIToken{}
	for rows.Next() {
		token, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

var _ database.APITokenRepository = (*apiTokenRepo)(nil)

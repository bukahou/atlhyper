// atlhyper_master_v2/database/command.go
// CommandHistoryRepository 实现
package database

import (
	"context"
	"database/sql"
)

type commandRepo struct {
	db      *sql.DB
	dialect CommandDialect
}

func newCommandRepo(db *sql.DB, dialect CommandDialect) *commandRepo {
	return &commandRepo{db: db, dialect: dialect}
}

func (r *commandRepo) Create(ctx context.Context, cmd *CommandHistory) error {
	query, args := r.dialect.Insert(cmd)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cmd.ID = id
	return nil
}

func (r *commandRepo) Update(ctx context.Context, cmd *CommandHistory) error {
	query, args := r.dialect.Update(cmd)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *commandRepo) GetByCommandID(ctx context.Context, cmdID string) (*CommandHistory, error) {
	query, args := r.dialect.SelectByCommandID(cmdID)
	return r.queryOne(ctx, query, args...)
}

func (r *commandRepo) ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*CommandHistory, error) {
	query, args := r.dialect.SelectByCluster(clusterID, limit, offset)
	return r.queryAll(ctx, query, args...)
}

func (r *commandRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*CommandHistory, error) {
	query, args := r.dialect.SelectByUser(userID, limit, offset)
	return r.queryAll(ctx, query, args...)
}

func (r *commandRepo) queryOne(ctx context.Context, query string, args ...any) (*CommandHistory, error) {
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

func (r *commandRepo) queryAll(ctx context.Context, query string, args ...any) ([]*CommandHistory, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []*CommandHistory
	for rows.Next() {
		cmd, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

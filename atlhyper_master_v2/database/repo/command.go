// atlhyper_master_v2/database/repo/command.go
// CommandHistoryRepository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type commandRepo struct {
	db      *sql.DB
	dialect database.CommandDialect
}

func newCommandRepo(db *sql.DB, dialect database.CommandDialect) *commandRepo {
	return &commandRepo{db: db, dialect: dialect}
}

func (r *commandRepo) Create(ctx context.Context, cmd *database.CommandHistory) error {
	query, args := r.dialect.Insert(cmd)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cmd.ID = id
	return nil
}

func (r *commandRepo) Update(ctx context.Context, cmd *database.CommandHistory) error {
	query, args := r.dialect.Update(cmd)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *commandRepo) GetByCommandID(ctx context.Context, cmdID string) (*database.CommandHistory, error) {
	query, args := r.dialect.SelectByCommandID(cmdID)
	return r.queryOne(ctx, query, args...)
}

func (r *commandRepo) ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*database.CommandHistory, error) {
	query, args := r.dialect.SelectByCluster(clusterID, limit, offset)
	return r.queryAll(ctx, query, args...)
}

func (r *commandRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*database.CommandHistory, error) {
	query, args := r.dialect.SelectByUser(userID, limit, offset)
	return r.queryAll(ctx, query, args...)
}

func (r *commandRepo) List(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error) {
	query, args := r.dialect.SelectWithOpts(opts)
	return r.queryAll(ctx, query, args...)
}

func (r *commandRepo) Count(ctx context.Context, opts database.CommandQueryOpts) (int64, error) {
	query, args := r.dialect.CountWithOpts(opts)
	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *commandRepo) queryOne(ctx context.Context, query string, args ...any) (*database.CommandHistory, error) {
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

func (r *commandRepo) queryAll(ctx context.Context, query string, args ...any) ([]*database.CommandHistory, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []*database.CommandHistory
	for rows.Next() {
		cmd, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

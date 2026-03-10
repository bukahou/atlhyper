package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type deployHistoryRepo struct {
	db      *sql.DB
	dialect database.DeployHistoryDialect
}

func newDeployHistoryRepo(db *sql.DB, dialect database.DeployHistoryDialect) *deployHistoryRepo {
	return &deployHistoryRepo{db: db, dialect: dialect}
}

func (r *deployHistoryRepo) Create(ctx context.Context, record *database.DeployHistory) error {
	query, args := r.dialect.Insert(record)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	record.ID = id
	return nil
}

func (r *deployHistoryRepo) GetByID(ctx context.Context, id int64) (*database.DeployHistory, error) {
	query, args := r.dialect.SelectByID(id)
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

func (r *deployHistoryRepo) List(ctx context.Context, opts database.DeployHistoryQueryOpts) ([]*database.DeployHistory, error) {
	query, args := r.dialect.SelectWithOpts(opts)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []*database.DeployHistory
	for rows.Next() {
		rec, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *deployHistoryRepo) Count(ctx context.Context, opts database.DeployHistoryQueryOpts) (int, error) {
	query, args := r.dialect.CountWithOpts(opts)
	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (r *deployHistoryRepo) GetLatestByPath(ctx context.Context, clusterID, path string) (*database.DeployHistory, error) {
	query, args := r.dialect.SelectLatestByPath(clusterID, path)
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

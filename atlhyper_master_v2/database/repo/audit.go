// atlhyper_master_v2/database/repo/audit.go
// AuditRepository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type auditRepo struct {
	db      *sql.DB
	dialect database.AuditDialect
}

func newAuditRepo(db *sql.DB, dialect database.AuditDialect) *auditRepo {
	return &auditRepo{db: db, dialect: dialect}
}

func (r *auditRepo) Create(ctx context.Context, log *database.AuditLog) error {
	query, args := r.dialect.Insert(log)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

func (r *auditRepo) List(ctx context.Context, opts database.AuditQueryOpts) ([]*database.AuditLog, error) {
	query, args := r.dialect.List(opts)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*database.AuditLog
	for rows.Next() {
		log, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *auditRepo) Count(ctx context.Context, opts database.AuditQueryOpts) (int64, error) {
	query, args := r.dialect.Count(opts)
	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

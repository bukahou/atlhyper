// atlhyper_master_v2/database/sqlite/impl/audit.go
// AuditRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, log *repository.AuditLog) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs (timestamp, user_id, username, role, source, action, resource, method,
			request_body, status_code, success, error_message, ip, user_agent, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.Timestamp.Format(time.RFC3339), log.UserID, log.Username, log.Role, log.Source,
		log.Action, log.Resource, log.Method, log.RequestBody, log.StatusCode,
		log.Success, log.ErrorMessage, log.IP, log.UserAgent, log.DurationMs,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

func (r *AuditRepository) List(ctx context.Context, opts repository.AuditQueryOpts) ([]*repository.AuditLog, error) {
	query := "SELECT * FROM audit_logs WHERE 1=1"
	args := []interface{}{}

	if opts.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.Source != "" {
		query += " AND source = ?"
		args = append(args, opts.Source)
	}
	if opts.Action != "" {
		query += " AND action = ?"
		args = append(args, opts.Action)
	}
	if !opts.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, opts.Since.Format(time.RFC3339))
	}
	if !opts.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, opts.Until.Format(time.RFC3339))
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*repository.AuditLog
	for rows.Next() {
		log := &repository.AuditLog{}
		var ts string
		err := rows.Scan(
			&log.ID, &ts, &log.UserID, &log.Username, &log.Role, &log.Source,
			&log.Action, &log.Resource, &log.Method, &log.RequestBody, &log.StatusCode,
			&log.Success, &log.ErrorMessage, &log.IP, &log.UserAgent, &log.DurationMs,
		)
		if err != nil {
			return nil, err
		}
		log.Timestamp, _ = time.Parse(time.RFC3339, ts)
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (r *AuditRepository) Count(ctx context.Context, opts repository.AuditQueryOpts) (int64, error) {
	query := "SELECT COUNT(*) FROM audit_logs WHERE 1=1"
	args := []interface{}{}

	if opts.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.Source != "" {
		query += " AND source = ?"
		args = append(args, opts.Source)
	}

	var count int64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

var _ repository.AuditRepository = (*AuditRepository)(nil)

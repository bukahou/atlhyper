// atlhyper_master_v2/database/sqlite/audit.go
// SQLite AuditDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type auditDialect struct{}

func (d *auditDialect) Insert(log *database.AuditLog) (string, []any) {
	query := `INSERT INTO audit_logs (timestamp, user_id, username, role, source, action, resource, method,
		request_body, status_code, success, error_message, ip, user_agent, duration_ms)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		log.Timestamp.Format(time.RFC3339), log.UserID, log.Username, log.Role, log.Source,
		log.Action, log.Resource, log.Method, log.RequestBody, log.StatusCode,
		log.Success, log.ErrorMessage, log.IP, log.UserAgent, log.DurationMs,
	}
	return query, args
}

func (d *auditDialect) List(opts database.AuditQueryOpts) (string, []any) {
	query := "SELECT id, timestamp, user_id, username, role, source, action, resource, method, request_body, status_code, success, error_message, ip, user_agent, duration_ms FROM audit_logs WHERE 1=1"
	args := []any{}

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

	return query, args
}

func (d *auditDialect) Count(opts database.AuditQueryOpts) (string, []any) {
	query := "SELECT COUNT(*) FROM audit_logs WHERE 1=1"
	args := []any{}

	if opts.UserID > 0 {
		query += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.Source != "" {
		query += " AND source = ?"
		args = append(args, opts.Source)
	}

	return query, args
}

func (d *auditDialect) ScanRow(rows *sql.Rows) (*database.AuditLog, error) {
	log := &database.AuditLog{}
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
	return log, nil
}

var _ database.AuditDialect = (*auditDialect)(nil)

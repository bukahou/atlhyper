// atlhyper_master_v2/database/sqlite/command.go
// SQLite CommandDialect 实现
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

type commandDialect struct{}

func (d *commandDialect) Insert(cmd *database.CommandHistory) (string, []any) {
	query := `INSERT INTO command_history (command_id, cluster_id, source, user_id, action,
		target_kind, target_namespace, target_name, params, status, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []any{
		cmd.CommandID, cmd.ClusterID, cmd.Source, cmd.UserID, cmd.Action,
		cmd.TargetKind, cmd.TargetNamespace, cmd.TargetName, cmd.Params, cmd.Status,
		cmd.CreatedAt.Format(time.RFC3339),
	}
	return query, args
}

func (d *commandDialect) Update(cmd *database.CommandHistory) (string, []any) {
	var startedAt, finishedAt *string
	if cmd.StartedAt != nil {
		s := cmd.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	if cmd.FinishedAt != nil {
		f := cmd.FinishedAt.Format(time.RFC3339)
		finishedAt = &f
	}
	query := `UPDATE command_history SET status = ?, result = ?, error_message = ?,
		started_at = ?, finished_at = ?, duration_ms = ? WHERE command_id = ?`
	args := []any{cmd.Status, cmd.Result, cmd.ErrorMessage, startedAt, finishedAt, cmd.DurationMs, cmd.CommandID}
	return query, args
}

func (d *commandDialect) SelectByCommandID(cmdID string) (string, []any) {
	return "SELECT id, command_id, cluster_id, source, user_id, action, target_kind, target_namespace, target_name, params, status, result, error_message, created_at, started_at, finished_at, duration_ms FROM command_history WHERE command_id = ?", []any{cmdID}
}

func (d *commandDialect) SelectByCluster(clusterID string, limit, offset int) (string, []any) {
	return "SELECT id, command_id, cluster_id, source, user_id, action, target_kind, target_namespace, target_name, params, status, result, error_message, created_at, started_at, finished_at, duration_ms FROM command_history WHERE cluster_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		[]any{clusterID, limit, offset}
}

func (d *commandDialect) SelectByUser(userID int64, limit, offset int) (string, []any) {
	return "SELECT id, command_id, cluster_id, source, user_id, action, target_kind, target_namespace, target_name, params, status, result, error_message, created_at, started_at, finished_at, duration_ms FROM command_history WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		[]any{userID, limit, offset}
}

func (d *commandDialect) ScanRow(rows *sql.Rows) (*database.CommandHistory, error) {
	cmd := &database.CommandHistory{}
	var createdAt string
	var startedAt, finishedAt sql.NullString
	err := rows.Scan(
		&cmd.ID, &cmd.CommandID, &cmd.ClusterID, &cmd.Source, &cmd.UserID, &cmd.Action,
		&cmd.TargetKind, &cmd.TargetNamespace, &cmd.TargetName, &cmd.Params, &cmd.Status,
		&cmd.Result, &cmd.ErrorMessage, &createdAt, &startedAt, &finishedAt, &cmd.DurationMs,
	)
	if err != nil {
		return nil, err
	}
	cmd.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		cmd.StartedAt = &t
	}
	if finishedAt.Valid {
		t, _ := time.Parse(time.RFC3339, finishedAt.String)
		cmd.FinishedAt = &t
	}
	return cmd, nil
}

var _ database.CommandDialect = (*commandDialect)(nil)

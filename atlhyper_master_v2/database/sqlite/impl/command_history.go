// atlhyper_master_v2/database/sqlite/impl/command_history.go
// CommandHistoryRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

type CommandHistoryRepository struct {
	db *sql.DB
}

func NewCommandHistoryRepository(db *sql.DB) *CommandHistoryRepository {
	return &CommandHistoryRepository{db: db}
}

func (r *CommandHistoryRepository) Create(ctx context.Context, cmd *repository.CommandHistory) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO command_history (command_id, cluster_id, source, user_id, action,
			target_kind, target_namespace, target_name, params, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmd.CommandID, cmd.ClusterID, cmd.Source, cmd.UserID, cmd.Action,
		cmd.TargetKind, cmd.TargetNamespace, cmd.TargetName, cmd.Params, cmd.Status,
		cmd.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cmd.ID = id
	return nil
}

func (r *CommandHistoryRepository) Update(ctx context.Context, cmd *repository.CommandHistory) error {
	var startedAt, finishedAt *string
	if cmd.StartedAt != nil {
		s := cmd.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	if cmd.FinishedAt != nil {
		f := cmd.FinishedAt.Format(time.RFC3339)
		finishedAt = &f
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE command_history SET status = ?, result = ?, error_message = ?,
			started_at = ?, finished_at = ?, duration_ms = ? WHERE command_id = ?`,
		cmd.Status, cmd.Result, cmd.ErrorMessage, startedAt, finishedAt, cmd.DurationMs, cmd.CommandID,
	)
	return err
}

func (r *CommandHistoryRepository) GetByCommandID(ctx context.Context, cmdID string) (*repository.CommandHistory, error) {
	row := r.db.QueryRowContext(ctx, "SELECT * FROM command_history WHERE command_id = ?", cmdID)
	return r.scanRow(row)
}

func (r *CommandHistoryRepository) ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*repository.CommandHistory, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT * FROM command_history WHERE cluster_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		clusterID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *CommandHistoryRepository) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*repository.CommandHistory, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT * FROM command_history WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?",
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanRows(rows)
}

func (r *CommandHistoryRepository) scanRow(row *sql.Row) (*repository.CommandHistory, error) {
	cmd := &repository.CommandHistory{}
	var createdAt string
	var startedAt, finishedAt sql.NullString
	err := row.Scan(
		&cmd.ID, &cmd.CommandID, &cmd.ClusterID, &cmd.Source, &cmd.UserID, &cmd.Action,
		&cmd.TargetKind, &cmd.TargetNamespace, &cmd.TargetName, &cmd.Params, &cmd.Status,
		&cmd.Result, &cmd.ErrorMessage, &createdAt, &startedAt, &finishedAt, &cmd.DurationMs,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
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

func (r *CommandHistoryRepository) scanRows(rows *sql.Rows) ([]*repository.CommandHistory, error) {
	var cmds []*repository.CommandHistory
	for rows.Next() {
		cmd := &repository.CommandHistory{}
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
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

var _ repository.CommandHistoryRepository = (*CommandHistoryRepository)(nil)

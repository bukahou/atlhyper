// atlhyper_master_v2/database/sqlite/impl/notify_channel.go
// NotifyChannelRepository SQLite 实现
package impl

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/database/repository"
)

type NotifyChannelRepository struct {
	db *sql.DB
}

func NewNotifyChannelRepository(db *sql.DB) *NotifyChannelRepository {
	return &NotifyChannelRepository{db: db}
}

func (r *NotifyChannelRepository) Create(ctx context.Context, ch *repository.NotifyChannel) error {
	now := time.Now().Format(time.RFC3339)
	result, err := r.db.ExecContext(ctx,
		"INSERT INTO notify_channels (type, name, enabled, config, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		ch.Type, ch.Name, ch.Enabled, ch.Config, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	ch.ID = id
	return nil
}

func (r *NotifyChannelRepository) Update(ctx context.Context, ch *repository.NotifyChannel) error {
	_, err := r.db.ExecContext(ctx,
		"UPDATE notify_channels SET name = ?, enabled = ?, config = ?, updated_at = ? WHERE id = ?",
		ch.Name, ch.Enabled, ch.Config, time.Now().Format(time.RFC3339), ch.ID,
	)
	return err
}

func (r *NotifyChannelRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM notify_channels WHERE id = ?", id)
	return err
}

func (r *NotifyChannelRepository) GetByID(ctx context.Context, id int64) (*repository.NotifyChannel, error) {
	return r.scanOne(ctx, "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels WHERE id = ?", id)
}

func (r *NotifyChannelRepository) GetByType(ctx context.Context, channelType string) (*repository.NotifyChannel, error) {
	return r.scanOne(ctx, "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels WHERE type = ?", channelType)
}

func (r *NotifyChannelRepository) List(ctx context.Context) ([]*repository.NotifyChannel, error) {
	return r.scanAll(ctx, "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels")
}

func (r *NotifyChannelRepository) ListEnabled(ctx context.Context) ([]*repository.NotifyChannel, error) {
	return r.scanAll(ctx, "SELECT id, type, name, enabled, config, created_at, updated_at FROM notify_channels WHERE enabled = 1")
}

func (r *NotifyChannelRepository) scanOne(ctx context.Context, query string, args ...interface{}) (*repository.NotifyChannel, error) {
	var ch repository.NotifyChannel
	var enabled int
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&ch.ID, &ch.Type, &ch.Name, &enabled, &ch.Config, &createdAt, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	ch.Enabled = enabled == 1
	ch.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	ch.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &ch, nil
}

func (r *NotifyChannelRepository) scanAll(ctx context.Context, query string, args ...interface{}) ([]*repository.NotifyChannel, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*repository.NotifyChannel
	for rows.Next() {
		var ch repository.NotifyChannel
		var enabled int
		var createdAt, updatedAt string
		if err := rows.Scan(&ch.ID, &ch.Type, &ch.Name, &enabled, &ch.Config, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		ch.Enabled = enabled == 1
		ch.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		ch.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		channels = append(channels, &ch)
	}
	return channels, rows.Err()
}

var _ repository.NotifyChannelRepository = (*NotifyChannelRepository)(nil)

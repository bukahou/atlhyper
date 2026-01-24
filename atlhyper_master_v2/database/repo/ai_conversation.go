// atlhyper_master_v2/database/repo/ai_conversation.go
// AIConversationRepository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiConversationRepo struct {
	db      *sql.DB
	dialect database.AIConversationDialect
}

func newAIConversationRepo(db *sql.DB, dialect database.AIConversationDialect) *aiConversationRepo {
	return &aiConversationRepo{db: db, dialect: dialect}
}

func (r *aiConversationRepo) Create(ctx context.Context, conv *database.AIConversation) error {
	query, args := r.dialect.Insert(conv)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	conv.ID = id
	return nil
}

func (r *aiConversationRepo) Update(ctx context.Context, conv *database.AIConversation) error {
	query, args := r.dialect.Update(conv)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiConversationRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiConversationRepo) GetByID(ctx context.Context, id int64) (*database.AIConversation, error) {
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

func (r *aiConversationRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*database.AIConversation, error) {
	query, args := r.dialect.SelectByUser(userID, limit, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var convs []*database.AIConversation
	for rows.Next() {
		conv, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		convs = append(convs, conv)
	}
	return convs, rows.Err()
}

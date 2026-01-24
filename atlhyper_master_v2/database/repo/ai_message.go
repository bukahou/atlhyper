// atlhyper_master_v2/database/repo/ai_message.go
// AIMessageRepository 实现
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiMessageRepo struct {
	db      *sql.DB
	dialect database.AIMessageDialect
}

func newAIMessageRepo(db *sql.DB, dialect database.AIMessageDialect) *aiMessageRepo {
	return &aiMessageRepo{db: db, dialect: dialect}
}

func (r *aiMessageRepo) Create(ctx context.Context, msg *database.AIMessage) error {
	query, args := r.dialect.Insert(msg)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	msg.ID = id
	return nil
}

func (r *aiMessageRepo) ListByConversation(ctx context.Context, convID int64) ([]*database.AIMessage, error) {
	query, args := r.dialect.SelectByConversation(convID)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []*database.AIMessage
	for rows.Next() {
		msg, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, rows.Err()
}

func (r *aiMessageRepo) DeleteByConversation(ctx context.Context, convID int64) error {
	query, args := r.dialect.DeleteByConversation(convID)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

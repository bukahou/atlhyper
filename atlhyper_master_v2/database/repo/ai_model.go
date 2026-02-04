// atlhyper_master_v2/database/repo/ai_model.go
// AI Provider Model Repository 実装
package repo

import (
	"context"
	"database/sql"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiProviderModelRepo struct {
	db      *sql.DB
	dialect database.AIProviderModelDialect
}

func newAIProviderModelRepo(db *sql.DB, dialect database.AIProviderModelDialect) *aiProviderModelRepo {
	return &aiProviderModelRepo{db: db, dialect: dialect}
}

func (r *aiProviderModelRepo) Create(ctx context.Context, m *database.AIProviderModel) error {
	query, args := r.dialect.Insert(m)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = id
	return nil
}

func (r *aiProviderModelRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiProviderModelRepo) GetByID(ctx context.Context, id int64) (*database.AIProviderModel, error) {
	query, args := r.dialect.SelectByID(id)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanRow(rows)
	}
	return nil, nil
}

func (r *aiProviderModelRepo) ListByProvider(ctx context.Context, provider string) ([]*database.AIProviderModel, error) {
	query, args := r.dialect.SelectByProvider(provider)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*database.AIProviderModel
	for rows.Next() {
		m, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *aiProviderModelRepo) ListAll(ctx context.Context) ([]*database.AIProviderModel, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []*database.AIProviderModel
	for rows.Next() {
		m, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (r *aiProviderModelRepo) GetDefaultModel(ctx context.Context, provider string) (*database.AIProviderModel, error) {
	query, args := r.dialect.SelectDefault(provider)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return r.dialect.ScanRow(rows)
	}
	return nil, nil
}

var _ database.AIProviderModelRepository = (*aiProviderModelRepo)(nil)

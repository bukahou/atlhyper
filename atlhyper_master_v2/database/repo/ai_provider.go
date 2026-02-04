// atlhyper_master_v2/database/repo/ai_provider.go
// AI Provider Repository 实现
package repo

import (
	"context"
	"database/sql"
	"log"

	"AtlHyper/atlhyper_master_v2/database"
)

type aiProviderRepo struct {
	db      *sql.DB
	dialect database.AIProviderDialect
}

func newAIProviderRepo(db *sql.DB, dialect database.AIProviderDialect) *aiProviderRepo {
	return &aiProviderRepo{db: db, dialect: dialect}
}

func (r *aiProviderRepo) Create(ctx context.Context, p *database.AIProvider) error {
	// 加密 API Key
	if globalEncryptor != nil && p.APIKey != "" {
		encrypted, err := globalEncryptor.Encrypt(p.APIKey)
		if err != nil {
			log.Printf("[AIProvider] API Key 加密失败: %v", err)
			return err
		}
		p.APIKey = encrypted
	}

	query, args := r.dialect.Insert(p)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func (r *aiProviderRepo) Update(ctx context.Context, p *database.AIProvider) error {
	// 加密 API Key
	if globalEncryptor != nil && p.APIKey != "" {
		encrypted, err := globalEncryptor.Encrypt(p.APIKey)
		if err != nil {
			log.Printf("[AIProvider] API Key 加密失败: %v", err)
			return err
		}
		p.APIKey = encrypted
	}

	query, args := r.dialect.Update(p)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiProviderRepo) Delete(ctx context.Context, id int64) error {
	query, args := r.dialect.Delete(id)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiProviderRepo) GetByID(ctx context.Context, id int64) (*database.AIProvider, error) {
	query, args := r.dialect.SelectByID(id)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		p, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		// 解密 API Key
		if globalEncryptor != nil && p.APIKey != "" {
			decrypted, err := globalEncryptor.Decrypt(p.APIKey)
			if err != nil {
				log.Printf("[AIProvider] API Key 解密失败: %v", err)
				// 解密失败不阻断，可能是旧的未加密数据
			} else {
				p.APIKey = decrypted
			}
		}
		return p, nil
	}
	return nil, nil
}

func (r *aiProviderRepo) List(ctx context.Context) ([]*database.AIProvider, error) {
	query, args := r.dialect.SelectAll()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*database.AIProvider
	for rows.Next() {
		p, err := r.dialect.ScanRow(rows)
		if err != nil {
			return nil, err
		}
		// 解密 API Key
		if globalEncryptor != nil && p.APIKey != "" {
			decrypted, err := globalEncryptor.Decrypt(p.APIKey)
			if err != nil {
				log.Printf("[AIProvider] API Key 解密失败 (ID=%d): %v", p.ID, err)
				// 解密失败不阻断，可能是旧的未加密数据
			} else {
				p.APIKey = decrypted
			}
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (r *aiProviderRepo) IncrementUsage(ctx context.Context, id int64, requests, tokens int64, cost float64) error {
	query, args := r.dialect.IncrementUsage(id, requests, tokens, cost)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiProviderRepo) UpdateStatus(ctx context.Context, id int64, status, errorMsg string) error {
	query, args := r.dialect.UpdateStatus(id, status, errorMsg)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

var _ database.AIProviderRepository = (*aiProviderRepo)(nil)

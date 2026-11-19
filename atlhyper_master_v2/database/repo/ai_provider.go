// atlhyper_master_v2/database/repo/ai_provider.go
// AI Provider Repository 实现
package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/common/logger"
)

var log = logger.Module("AIProvider")

type aiProviderRepo struct {
	db      *sql.DB
	dialect database.AIProviderDialect
}

func newAIProviderRepo(db *sql.DB, dialect database.AIProviderDialect) *aiProviderRepo {
	return &aiProviderRepo{db: db, dialect: dialect}
}

func (r *aiProviderRepo) Create(ctx context.Context, p *database.AIProvider) error {
	if err := encryptAPIKey(p); err != nil {
		return err
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
	if err := encryptAPIKey(p); err != nil {
		return err
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
		decryptAPIKey(p)
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
		decryptAPIKey(p)
		providers = append(providers, p)
	}
	return providers, nil
}

func (r *aiProviderRepo) UpdateRoles(ctx context.Context, id int64, roles []string) error {
	rolesJSON := "[]"
	if len(roles) > 0 {
		b, _ := json.Marshal(roles)
		rolesJSON = string(b)
	}
	query, args := r.dialect.UpdateRoles(id, rolesJSON)
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *aiProviderRepo) FindByRole(ctx context.Context, role string) (*database.AIProvider, error) {
	// 查询所有 provider，在应用层过滤角色（SQLite JSON 查询能力有限）
	providers, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		for _, r := range p.Roles {
			if r == role {
				return p, nil
			}
		}
	}
	return nil, nil
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

// encryptAPIKey API Key 落库前加密；没有加密器时拒绝（宁可拒绝，也不存明文）
func encryptAPIKey(p *database.AIProvider) error {
	if p.APIKey == "" {
		return nil
	}
	if globalEncryptor == nil {
		return database.ErrEncryptionUnavailable
	}
	encrypted, err := globalEncryptor.Encrypt(p.APIKey)
	if err != nil {
		return fmt.Errorf("加密 API Key 失败: %w", err)
	}
	p.APIKey = encrypted
	return nil
}

// decryptAPIKey 读出后解密。解不开（加密密钥换过、数据损坏）时按没有填 API Key 处理：
// 调用会因缺少 API Key 失败并在设置页留下错误，管理员重新填写即可；⛔ 不把密文当 API Key 发给模型厂商
func decryptAPIKey(p *database.AIProvider) {
	if p.APIKey == "" {
		return
	}
	if globalEncryptor == nil {
		log.Error("没有加密器，无法读取 API Key", "provider", p.ID)
		p.APIKey = ""
		return
	}
	decrypted, err := globalEncryptor.Decrypt(p.APIKey)
	if err != nil {
		log.Error("API Key 无法解密，按未填写处理（请在 AI 设置中重新填写）", "provider", p.ID)
		p.APIKey = ""
		return
	}
	p.APIKey = decrypted
}

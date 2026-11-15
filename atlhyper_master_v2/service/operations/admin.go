// atlhyper_master_v2/service/operations/admin.go
// AdminService — 管理写入操作（AI 设置、AI Provider）
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// AdminService 管理写入服务
type AdminService struct {
	aiProviderRepo database.AIProviderRepository
	aiSettingsRepo database.AISettingsRepository
	aiBudgetRepo   database.AIRoleBudgetRepository
}

// NewAdminService 创建 AdminService
func NewAdminService(
	aiProviderRepo database.AIProviderRepository,
	aiSettingsRepo database.AISettingsRepository,
	aiBudgetRepo database.AIRoleBudgetRepository,
) *AdminService {
	return &AdminService{
		aiProviderRepo: aiProviderRepo,
		aiSettingsRepo: aiSettingsRepo,
		aiBudgetRepo:   aiBudgetRepo,
	}
}

// ==================== AI Provider ====================

func (s *AdminService) CreateAIProvider(ctx context.Context, p *database.AIProvider) error {
	return s.aiProviderRepo.Create(ctx, p)
}

func (s *AdminService) UpdateAIProvider(ctx context.Context, p *database.AIProvider) error {
	return s.aiProviderRepo.Update(ctx, p)
}

func (s *AdminService) DeleteAIProvider(ctx context.Context, id int64) error {
	return s.aiProviderRepo.Delete(ctx, id)
}

func (s *AdminService) UpdateAISettings(ctx context.Context, cfg *database.AISettings) error {
	return s.aiSettingsRepo.Update(ctx, cfg)
}

func (s *AdminService) UpdateAIProviderRoles(ctx context.Context, id int64, roles []string) error {
	return s.aiProviderRepo.UpdateRoles(ctx, id, roles)
}

// ==================== AI Role Budget ====================

func (s *AdminService) UpdateAIRoleBudget(ctx context.Context, budget *database.AIRoleBudget) error {
	return s.aiBudgetRepo.Upsert(ctx, budget)
}

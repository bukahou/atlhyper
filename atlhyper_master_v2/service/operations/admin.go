// atlhyper_master_v2/service/operations/admin.go
// AdminService — 管理写入操作（通知渠道、设置、AI Provider）
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// AdminService 管理写入服务
type AdminService struct {
	notifyRepo     database.NotifyChannelRepository
	settingsRepo   database.SettingsRepository
	aiProviderRepo database.AIProviderRepository
	aiActiveRepo   database.AIActiveConfigRepository
	aiBudgetRepo   database.AIRoleBudgetRepository
}

// NewAdminService 创建 AdminService
func NewAdminService(
	notifyRepo database.NotifyChannelRepository,
	settingsRepo database.SettingsRepository,
	aiProviderRepo database.AIProviderRepository,
	aiActiveRepo database.AIActiveConfigRepository,
	aiBudgetRepo database.AIRoleBudgetRepository,
) *AdminService {
	return &AdminService{
		notifyRepo:     notifyRepo,
		settingsRepo:   settingsRepo,
		aiProviderRepo: aiProviderRepo,
		aiActiveRepo:   aiActiveRepo,
		aiBudgetRepo:   aiBudgetRepo,
	}
}

// ==================== Notify ====================

func (s *AdminService) CreateNotifyChannel(ctx context.Context, ch *database.NotifyChannel) error {
	return s.notifyRepo.Create(ctx, ch)
}

func (s *AdminService) UpdateNotifyChannel(ctx context.Context, ch *database.NotifyChannel) error {
	return s.notifyRepo.Update(ctx, ch)
}

// ==================== Settings ====================

func (s *AdminService) SetSetting(ctx context.Context, setting *database.Setting) error {
	return s.settingsRepo.Set(ctx, setting)
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

func (s *AdminService) UpdateAIActiveConfig(ctx context.Context, cfg *database.AIActiveConfig) error {
	return s.aiActiveRepo.Update(ctx, cfg)
}

func (s *AdminService) UpdateAIProviderRoles(ctx context.Context, id int64, roles []string) error {
	return s.aiProviderRepo.UpdateRoles(ctx, id, roles)
}

// ==================== AI Role Budget ====================

func (s *AdminService) UpdateAIRoleBudget(ctx context.Context, budget *database.AIRoleBudget) error {
	return s.aiBudgetRepo.Upsert(ctx, budget)
}

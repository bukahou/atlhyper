// atlhyper_master_v2/service/operations/deploy.go
// OpsDeploy 实现 — 部署配置与 GitHub 集成的写入
//
// admin/deploy.go 与 settings/github.go 的写入都经这里，Gateway 不直接访问数据库。
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// DeployService 部署与 GitHub 集成的写入操作
type DeployService struct {
	deployConfigRepo  database.DeployConfigRepository
	githubInstallRepo database.GitHubInstallationRepository
}

// NewDeployService 创建 DeployService
func NewDeployService(
	deployConfigRepo database.DeployConfigRepository,
	githubInstallRepo database.GitHubInstallationRepository,
) *DeployService {
	return &DeployService{
		deployConfigRepo:  deployConfigRepo,
		githubInstallRepo: githubInstallRepo,
	}
}

func (s *DeployService) UpsertDeployConfig(ctx context.Context, cfg *database.DeployConfig) error {
	return s.deployConfigRepo.Upsert(ctx, cfg)
}

func (s *DeployService) UpsertGitHubInstallation(ctx context.Context, inst *database.GitHubInstallation) error {
	return s.githubInstallRepo.Upsert(ctx, inst)
}

func (s *DeployService) DeleteGitHubInstallation(ctx context.Context) error {
	return s.githubInstallRepo.Delete(ctx)
}

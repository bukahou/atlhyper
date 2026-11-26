// atlhyper_master_v2/service/operations/github.go
// OpsGitHub 实现 — GitHub 集成的写入
//
// settings/github.go 的写入都经这里，Gateway 不直接访问数据库。
package operations

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// GitHubService GitHub 集成的写入操作
type GitHubService struct {
	githubInstallRepo database.GitHubInstallationRepository
}

// NewGitHubService 创建 GitHubService
func NewGitHubService(githubInstallRepo database.GitHubInstallationRepository) *GitHubService {
	return &GitHubService{githubInstallRepo: githubInstallRepo}
}

func (s *GitHubService) UpsertGitHubInstallation(ctx context.Context, inst *database.GitHubInstallation) error {
	return s.githubInstallRepo.Upsert(ctx, inst)
}

func (s *GitHubService) DeleteGitHubInstallation(ctx context.Context) error {
	return s.githubInstallRepo.Delete(ctx)
}

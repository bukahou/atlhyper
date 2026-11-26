// atlhyper_master_v2/service/query/github.go
// QueryGitHub 实现 — GitHub 集成的读取
//
// settings/github.go 的读取都经这里，Gateway 不直接访问数据库。
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

func (q *GitHubQuery) GetGitHubInstallation(ctx context.Context) (*database.GitHubInstallation, error) {
	return q.githubInstallRepo.Get(ctx)
}

func (q *GitHubQuery) ListRepoConfigs(ctx context.Context) ([]*database.RepoConfig, error) {
	return q.repoConfigRepo.List(ctx)
}

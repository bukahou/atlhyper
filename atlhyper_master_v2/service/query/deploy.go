// atlhyper_master_v2/service/query/deploy.go
// QueryDeploy 实现 — 部署配置、部署历史、GitHub 集成的读取
//
// admin/deploy.go 与 settings/github.go 的读取都经这里，Gateway 不直接访问数据库。
package query

import (
	"context"

	"AtlHyper/atlhyper_master_v2/database"
)

// ==================== Deploy Config ====================

func (q *DeployQuery) GetDeployConfig(ctx context.Context, clusterID string) (*database.DeployConfig, error) {
	return q.deployConfigRepo.GetByCluster(ctx, clusterID)
}

// ==================== Deploy History ====================

func (q *DeployQuery) ListDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) ([]*database.DeployHistory, error) {
	return q.deployHistoryRepo.List(ctx, opts)
}

func (q *DeployQuery) CountDeployHistory(ctx context.Context, opts database.DeployHistoryQueryOpts) (int, error) {
	return q.deployHistoryRepo.Count(ctx, opts)
}

// ==================== GitHub 集成 ====================

func (q *DeployQuery) GetGitHubInstallation(ctx context.Context) (*database.GitHubInstallation, error) {
	return q.githubInstallRepo.Get(ctx)
}

func (q *DeployQuery) ListRepoConfigs(ctx context.Context) ([]*database.RepoConfig, error) {
	return q.repoConfigRepo.List(ctx)
}

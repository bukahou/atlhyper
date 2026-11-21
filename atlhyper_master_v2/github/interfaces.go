// atlhyper_master_v2/github/interfaces.go
// GitHub 集成模块 — 接口定义
package github

import "context"

// Client GitHub API 客户端接口
type Client interface {
	// App 认证
	GetInstallationToken(ctx context.Context) (string, error)

	// 仓库信息
	ListRepos(ctx context.Context, installationID int64) ([]Repository, error)
	IsPrivateRepo(ctx context.Context, repo string) (bool, error)
	ListTopDirs(ctx context.Context, repo, branch string) ([]string, error)
	ReadFile(ctx context.Context, repo, path, ref string) (string, error)

	// Code Intel
	ListCommits(ctx context.Context, repo, branch string, limit int) ([]Commit, error)
	SearchCode(ctx context.Context, repo, query string) ([]CodeSearchResult, error)
}

// InstallationManager GitHub App 的安装管理（以 App JWT 身份调用，与上面的仓库读取能力分开）
type InstallationManager interface {
	AuthURL(state string) string
	SetInstallationID(id int64)
	GetAppOwner(ctx context.Context) (string, error) // App 所有者账号：只有它名下的安装才允许接入
	GetInstallationAccount(ctx context.Context, installationID int64) (string, error)
	ListInstallations(ctx context.Context) ([]Installation, error)
}

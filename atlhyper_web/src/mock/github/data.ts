/**
 * GitHub 集成 Mock 数据
 */

export interface MockGitHubConnection {
  connected: boolean;
  accountLogin: string;
  avatarUrl: string;
  installationId: number;
}

export const MOCK_GITHUB_CONNECTION: MockGitHubConnection = {
  connected: true,
  accountLogin: "example-org",
  avatarUrl: "",
  installationId: 12345678,
};

export const MOCK_GITHUB_NOT_CONNECTED: MockGitHubConnection = {
  connected: false,
  accountLogin: "",
  avatarUrl: "",
  installationId: 0,
};

export interface MockAuthorizedRepo {
  fullName: string;
  defaultBranch: string;
  private: boolean;
}

export const MOCK_AUTHORIZED_REPOS: MockAuthorizedRepo[] = [
  { fullName: "example-org/cluster-config", defaultBranch: "main", private: true },
  { fullName: "example-org/demo-app", defaultBranch: "main", private: false },
  { fullName: "example-org/atlhyper", defaultBranch: "main", private: false },
];

// 仓库顶层目录（GitHub API ListTopDirs 返回）
export const MOCK_REPO_DIRS: Record<string, string[]> = {
  "example-org/demo-app": [
    "geass_auth/",
    "geass_gateway/",
    "geass_media/",
    "geass_user/",
    "geass_favorites/",
    "geass_history/",
    "geass_web/",
    "geass_common/",
  ],
  "example-org/atlhyper": [
    "atlhyper_master_v2/",
    "atlhyper_agent_v2/",
    "atlhyper_web/",
    "common/",
    "model_v3/",
  ],
};

export function mockGetRepoDirs(repo: string) {
  return MOCK_REPO_DIRS[repo] || [];
}

export function mockGetGitHubConnection() {
  return MOCK_GITHUB_CONNECTION;
}

export function mockGetAuthorizedRepos() {
  return MOCK_AUTHORIZED_REPOS;
}

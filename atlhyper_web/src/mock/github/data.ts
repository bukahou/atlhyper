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
  accountLogin: "wuxiafeng",
  avatarUrl: "https://github.com/wuxiafeng.png",
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
  { fullName: "wuxiafeng/Config", defaultBranch: "main", private: true },
  { fullName: "wuxiafeng/Geass", defaultBranch: "main", private: false },
  { fullName: "wuxiafeng/atlhyper", defaultBranch: "main", private: false },
];

// 仓库顶层目录（GitHub API ListTopDirs 返回）
export const MOCK_REPO_DIRS: Record<string, string[]> = {
  "wuxiafeng/Geass": [
    "geass_auth/",
    "geass_gateway/",
    "geass_media/",
    "geass_user/",
    "geass_favorites/",
    "geass_history/",
    "geass_web/",
    "geass_common/",
  ],
  "wuxiafeng/atlhyper": [
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

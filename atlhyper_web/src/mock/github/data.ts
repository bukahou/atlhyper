/**
 * GitHub 集成 Mock 数据
 */

export interface MockGitHubConnection {
  connected: boolean;
  accountLogin: string;
  avatarUrl: string;
  installationId: number;
}

export interface MockRepoMapping {
  id: number;
  clusterId: string;
  repo: string;
  namespace: string;
  deployment: string;
  container: string;
  imagePrefix: string;
  sourcePath: string;
  confirmed: boolean;
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

// 初始为空，用户手动添加映射行
export const MOCK_REPO_MAPPINGS: MockRepoMapping[] = [];

export interface MockAuthorizedRepo {
  fullName: string;
  defaultBranch: string;
  private: boolean;
  mappingEnabled: boolean;
}

export const MOCK_AUTHORIZED_REPOS: MockAuthorizedRepo[] = [
  { fullName: "wuxiafeng/Config", defaultBranch: "main", private: true, mappingEnabled: false },
  { fullName: "wuxiafeng/Geass", defaultBranch: "main", private: false, mappingEnabled: true },
  { fullName: "wuxiafeng/atlhyper", defaultBranch: "main", private: false, mappingEnabled: true },
];

// 可选项数据（来自 Snapshot + GitHub API）
export const MOCK_NAMESPACES = ["geass", "geass-web", "atlhyper", "default"];

export const MOCK_DEPLOYMENTS = [
  { name: "geass-auth", namespace: "geass", image: "bukahou/geass-auth:20260309-abc1234" },
  { name: "geass-gateway", namespace: "geass", image: "bukahou/geass-gateway:20260310-def5678" },
  { name: "geass-media", namespace: "geass", image: "bukahou/geass-media:20260310-def5678" },
  { name: "geass-user", namespace: "geass", image: "bukahou/geass-user:20260310-def5678" },
  { name: "geass-favorites", namespace: "geass", image: "bukahou/geass-favorites:20260310-def5678" },
  { name: "geass-history", namespace: "geass", image: "bukahou/geass-history:20260310-def5678" },
  { name: "geass-web", namespace: "geass-web", image: "bukahou/geass-web:20260308-ghi9012" },
  { name: "atlhyper-master", namespace: "atlhyper", image: "bukahou/atlhyper-master:20260310-mst4567" },
  { name: "atlhyper-agent", namespace: "atlhyper", image: "bukahou/atlhyper-agent:20260310-agt8901" },
  { name: "atlhyper-web", namespace: "atlhyper", image: "bukahou/atlhyper-web:20260309-web2345" },
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

// 初始为空，用户手动为每个仓库添加 Namespace
export const MOCK_REPO_NAMESPACES: Record<string, string[]> = {};

export function mockGetNamespaces() {
  return MOCK_NAMESPACES;
}

export function mockGetRepoNamespaces(repo: string) {
  return MOCK_REPO_NAMESPACES[repo] || [];
}

export function mockGetDeployments() {
  return MOCK_DEPLOYMENTS;
}

export function mockGetRepoDirs(repo: string) {
  return MOCK_REPO_DIRS[repo] || [];
}

export function mockGetGitHubConnection() {
  return MOCK_GITHUB_CONNECTION;
}

export function mockGetAuthorizedRepos() {
  return MOCK_AUTHORIZED_REPOS;
}

export function mockGetRepoMappings() {
  return MOCK_REPO_MAPPINGS;
}

/**
 * 部署管理 Mock 数据
 */

export interface MockDeployConfig {
  repoUrl: string;
  paths: string[];
  intervalSec: number;
  autoDeploy: boolean;
  clusterId: string;
}

export interface MockPathStatus {
  path: string;
  namespace: string;   // 后端从 kustomize build 结果中提取
  inSync: boolean;
  resourceCount: number;
  lastSyncAt: string;
}

export type DeployTrigger = "auto" | "manual" | "rollback";

export interface MockChangedFile {
  filename: string;
  status: string;   // added, modified, removed
  additions: number;
  deletions: number;
}

export interface MockDeployRecord {
  id: number;
  clusterId: string;
  path: string;
  namespace: string;   // 后端从 kustomize build 结果中提取
  commitSha: string;
  commitMessage: string;
  commitAuthor: string;
  commitAvatarUrl: string;
  prNumber: number;
  prTitle: string;
  prUrl: string;
  changedFiles: string;    // JSON array of MockChangedFile
  compareUrl: string;      // GitHub compare URL
  sourceRepo: string;      // 源码仓库 (e.g. "bukahou/Geass")
  sourceCommitSha: string; // 源码 commit SHA (e.g. "b572f17")
  deployedAt: string;      // 实际部署完成时间
  trigger: DeployTrigger;
  status: "pending" | "success" | "failed";
  // 详情字段（点击查看时展示）
  durationMs: number;          // 部署耗时（毫秒）
  resourceTotal: number;       // 总资源数
  resourceChanged: number;     // 变更资源数
  errorMessage?: string;       // 失败时的错误信息
}

export const MOCK_DEPLOY_CONFIG: MockDeployConfig = {
  repoUrl: "wuxiafeng/Config",
  paths: [
    "zgmf-x10a/k8s-configs/Geass/backend",
    "zgmf-x10a/k8s-configs/Geass/web",
    "zgmf-x10a/k8s-configs/atlhyper/master",
    "zgmf-x10a/k8s-configs/atlhyper/web",
  ],
  intervalSec: 60,
  autoDeploy: true,
  clusterId: "zgmf-x10a",
};

export const MOCK_PATH_STATUS: MockPathStatus[] = [
  {
    path: "zgmf-x10a/k8s-configs/Geass/backend",
    namespace: "geass",
    inSync: false,
    resourceCount: 12,
    lastSyncAt: "2026-03-10T10:32:00Z",
  },
  {
    path: "zgmf-x10a/k8s-configs/Geass/web",
    namespace: "geass-web",
    inSync: true,
    resourceCount: 4,
    lastSyncAt: "2026-03-10T10:32:00Z",
  },
  {
    path: "zgmf-x10a/k8s-configs/atlhyper/master",
    namespace: "atlhyper",
    inSync: true,
    resourceCount: 5,
    lastSyncAt: "2026-03-10T10:30:00Z",
  },
  {
    path: "zgmf-x10a/k8s-configs/atlhyper/web",
    namespace: "atlhyper",
    inSync: true,
    resourceCount: 4,
    lastSyncAt: "2026-03-10T10:30:00Z",
  },
];

export const MOCK_DEPLOY_HISTORY: MockDeployRecord[] = [
  {
    id: 1,
    clusterId: "zgmf-x10a",
    path: "Geass/backend",
    namespace: "geass",
    commitSha: "def5678",
    commitMessage: "refactor: routing matching logic",
    commitAuthor: "bukahou",
    commitAvatarUrl: "",
    prNumber: 5,
    prTitle: "refactor: routing matching logic",
    prUrl: "",
    changedFiles: JSON.stringify([
      { filename: "geass_gateway/src/main/java/Router.java", status: "modified", additions: 42, deletions: 15 },
      { filename: "geass_gateway/src/main/java/Config.java", status: "modified", additions: 8, deletions: 3 },
    ]),
    compareUrl: "",
    sourceRepo: "bukahou/Geass",
    sourceCommitSha: "def5678",
    deployedAt: "2026-03-10T10:32:15Z",
    trigger: "auto",
    status: "success",
    durationMs: 2340,
    resourceTotal: 12,
    resourceChanged: 3,
  },
  {
    id: 2,
    clusterId: "zgmf-x10a",
    path: "Geass/backend",
    namespace: "geass",
    commitSha: "abc1234",
    commitMessage: "fix: JWT token refresh bug — 修复 Token 过期后无法自动刷新导致 401 的问题",
    commitAuthor: "bukahou",
    commitAvatarUrl: "",
    prNumber: 4,
    prTitle: "fix: JWT token refresh bug",
    prUrl: "",
    changedFiles: JSON.stringify([
      { filename: "geass_auth/src/main/java/TokenService.java", status: "modified", additions: 25, deletions: 8 },
    ]),
    compareUrl: "",
    sourceRepo: "bukahou/Geass",
    sourceCommitSha: "abc1234",
    deployedAt: "2026-03-09T14:33:20Z",
    trigger: "auto",
    status: "failed",
    durationMs: 5120,
    resourceTotal: 12,
    resourceChanged: 1,
    errorMessage: "Apply failed: Deployment geass-auth — container image pull error: ImagePullBackOff for registry.example.com/geass-auth:v1.2.3",
  },
  {
    id: 3,
    clusterId: "zgmf-x10a",
    path: "Geass/backend",
    namespace: "geass",
    commitSha: "old7890",
    commitMessage: "fix: JWT token refresh bug — rollback to previous version",
    commitAuthor: "bukahou",
    commitAvatarUrl: "",
    prNumber: 0,
    prTitle: "",
    prUrl: "",
    changedFiles: "[]",
    compareUrl: "",
    sourceRepo: "bukahou/Geass",
    sourceCommitSha: "old7890",
    deployedAt: "2026-03-09T14:40:00Z",
    trigger: "rollback",
    status: "success",
    durationMs: 1890,
    resourceTotal: 12,
    resourceChanged: 1,
  },
  {
    id: 4,
    clusterId: "zgmf-x10a",
    path: "Geass/web",
    namespace: "geass-web",
    commitSha: "ghi9012",
    commitMessage: "feat: dark mode support",
    commitAuthor: "bukahou",
    commitAvatarUrl: "",
    prNumber: 3,
    prTitle: "feat: dark mode support",
    prUrl: "",
    changedFiles: JSON.stringify([
      { filename: "geass_web/src/theme/dark.css", status: "added", additions: 120, deletions: 0 },
      { filename: "geass_web/src/components/ThemeToggle.tsx", status: "added", additions: 45, deletions: 0 },
    ]),
    compareUrl: "",
    sourceRepo: "bukahou/Geass",
    sourceCommitSha: "ghi9012",
    deployedAt: "2026-03-08T09:15:30Z",
    trigger: "auto",
    status: "success",
    durationMs: 1560,
    resourceTotal: 4,
    resourceChanged: 2,
  },
  {
    id: 5,
    clusterId: "zgmf-x10a",
    path: "atlhyper/master",
    namespace: "atlhyper",
    commitSha: "mst4567",
    commitMessage: "fix: snapshot processing timeout — 增加 processor 超时时间至 30s",
    commitAuthor: "bukahou",
    commitAvatarUrl: "",
    prNumber: 0,
    prTitle: "",
    prUrl: "",
    changedFiles: JSON.stringify([
      { filename: "atlhyper_master_v2/processor/snapshot.go", status: "modified", additions: 5, deletions: 2 },
    ]),
    compareUrl: "",
    sourceRepo: "bukahou/AtlHyper",
    sourceCommitSha: "mst4567",
    deployedAt: "2026-03-07T16:22:10Z",
    trigger: "manual",
    status: "success",
    durationMs: 3200,
    resourceTotal: 5,
    resourceChanged: 1,
  },
];

// 从 GitHub 集成共享的已授权仓库列表
export const MOCK_AUTHORIZED_REPOS = [
  { fullName: "wuxiafeng/Config", defaultBranch: "main", private: true },
  { fullName: "wuxiafeng/Geass", defaultBranch: "main", private: false },
  { fullName: "wuxiafeng/atlhyper", defaultBranch: "main", private: false },
];

// 扫描 Config 仓库发现的所有 kustomization.yaml 路径
// 实际由后端通过 GitHub API 遍历仓库目录获取
export const MOCK_KUSTOMIZE_PATHS: Record<string, string[]> = {
  "wuxiafeng/Config": [
    "zgmf-x10a/k8s-configs/Geass/backend",
    "zgmf-x10a/k8s-configs/Geass/web",
    "zgmf-x10a/k8s-configs/atlhyper/master",
    "zgmf-x10a/k8s-configs/atlhyper/agent",
    "zgmf-x10a/k8s-configs/atlhyper/web",
    "zgmf-x10a/k8s-configs/core",
    "zgmf-x10a/k8s-configs/nginx",
    "zgmf-x10a/k8s-configs/redis",
  ],
};

export function mockGetDeployConfig() {
  return MOCK_DEPLOY_CONFIG;
}

export function mockGetPathStatus() {
  return MOCK_PATH_STATUS;
}

export function mockGetDeployHistory() {
  return MOCK_DEPLOY_HISTORY;
}

export function mockGetAuthorizedRepos() {
  return MOCK_AUTHORIZED_REPOS;
}

export function mockGetKustomizePaths(repo: string) {
  return MOCK_KUSTOMIZE_PATHS[repo] || [];
}

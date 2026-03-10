# GitHub 集成 + CD + 代码智能 设计

> 状态: active | 创建: 2026-03-08 | 更新: 2026-03-10

---

## 背景

| 问题 | 说明 |
|------|------|
| **AI 缺乏代码上下文** | AI 只能访问运行时数据，无法理解应用代码。Geass Gateway 遭受 WordPress 扫描时，AI 无法识别 Spring Boot 项目，给出错误建议。 |
| **部署与异常无法关联** | AIOps 检测到异常后，不知道是否刚发生过部署，需要人工去 GitHub 查 commit 比对时间线。 |
| **部署和回滚依赖手动操作** | 手动改 YAML → kubectl apply，回滚也需人工找到上一个版本再手动操作。 |

## 目标

| 能力 | 当前 | 目标 |
|------|------|------|
| **认证** | 本地账号 | 本地账号 + GitHub OAuth 登录 |
| **代码感知** | 无 | AI 可读取关联仓库的源码、commits、PRs |
| **部署** | 手动 kubectl | Full GitOps — Config 仓库任何变更自动部署 |
| **部署感知** | 只知道当前 image | 完整部署历史（按 kustomize 路径 + commit 关联） |
| **回滚** | 手动改 YAML | AtlHyper 一键回滚到指定 commit |
| **变更关联** | 无 | 「异常发生前 3 分钟部署了 commit def5678」 |

---

## 核心架构

### 三关注点分离

```
┌─────────────┐     ┌──────────────────────┐     ┌──────────────────────┐
│     CI      │     │        CD            │     │    Code Intel        │
│   (外部)    │     │   (AtlHyper Master)  │     │   (AtlHyper AIOps)  │
│             │     │                      │     │                      │
│  测试/编译   │     │  轮询 Config 仓库     │     │  Commits / Diffs     │
│  构建镜像   │     │  kustomize build     │     │  PRs / 代码读取       │
│  推送镜像   │     │  Command → Agent     │     │  变更与异常关联       │
│  更新 Config│     │  记录 deploy_history  │     │  AI 根因推理         │
│             │     │                      │     │                      │
│  GitHub     │     │  页面: /admin/deploy  │     │  页面: /settings/github │
│  Actions    │     │                      │     │                      │
└──────┬──────┘     └──────────┬───────────┘     └──────────┬───────────┘
       │                       │                            │
       └───────────────────────┴────────────────────────────┘
                     唯一交集: Config 仓库
```

三者完全独立，唯一交集是 Config 仓库中的变更。可独立实施、独立运行。

### GitHub App 统一认证

CD 和 Code Intel **共用同一个 GitHub App 安装**。

- 用户在 `/settings/github` 点击「连接 GitHub」→ 跳转安装 → 勾选仓库 → 回调完成
- Installation Token 自动刷新（1 小时），用于 GitHub REST API
- 无需 git clone，通过 API 直接读取仓库文件

### 模块关系：auth 定义接口，github 实现一切

```
auth/                          ← 通用认证框架
├── interfaces.go              ← OAuthProvider 接口（不知道 GitHub 的存在）
└── types.go                   ← OAuthUser 通用类型

github/                        ← 所有 GitHub 交互（App + API + OAuth 全部收敛）
├── interfaces.go              ← GitHubClient 接口
├── app.go                     ← App 身份管理（JWT + Installation Token 自动刷新）
├── api.go                     ← REST API 调用
├── oauth.go                   ← 实现 auth.OAuthProvider
└── types.go                   ← Repository, Commit, PullRequest 等
```

```go
// auth/interfaces.go
type OAuthProvider interface {
    AuthURL(state string) string
    Exchange(ctx context.Context, code string) (*OAuthUser, error)
}

type OAuthUser struct {
    Provider   string // "github"
    ExternalID string // GitHub user ID
    Login      string // GitHub username
    Email      string
    AvatarURL  string
}
```

```go
// github/interfaces.go
type GitHubClient interface {
    // App 认证
    GetInstallationToken(ctx context.Context) (string, error)

    // 仓库信息
    ListRepos(ctx context.Context, installationID int64) ([]Repository, error)
    ListTopDirs(ctx context.Context, repo, branch string) ([]string, error)
    ScanKustomizePaths(ctx context.Context, repo, branch string) ([]string, error)

    // CD 轮询
    GetLatestCommitSHA(ctx context.Context, repo, branch string) (string, error)
    CompareCommits(ctx context.Context, repo, base, head string) ([]ChangedFile, error)
    ReadFile(ctx context.Context, repo, path, ref string) (string, error)
    ReadDirectory(ctx context.Context, repo, path, ref string) ([]FileEntry, error)

    // Code Intel
    ListCommits(ctx context.Context, repo, branch string, limit int) ([]Commit, error)
    GetPRByCommit(ctx context.Context, repo, sha string) (*PullRequest, error)
    GetCommitFiles(ctx context.Context, repo, sha string) ([]string, error)
    SearchCode(ctx context.Context, repo, query string) ([]CodeSearchResult, error)
}

// github/oauth.go — 同一 Client 实现 auth.OAuthProvider
func (c *Client) AuthURL(state string) string { ... }
func (c *Client) Exchange(ctx context.Context, code string) (*auth.OAuthUser, error) { ... }
```

```go
// master.go — 注入
githubClient := github.NewClient(cfg)
authModule.RegisterProvider("github", githubClient)  // OAuth 登录
deployer.SetGitHubClient(githubClient)               // CD 轮询
```

### 为什么轮询而非 Webhook

Master 运行在家庭 K3s 集群内，**无公网 IP**，无法接收 Webhook。

采用**间隔轮询**（非长轮询）：
- 定时器每 N 秒（可配，默认 60s）发起 REST 请求检查 commit SHA
- SHA 未变 → 跳过（极低开销）；变了 → 触发部署
- 网络异常仅记录日志，不触发告警，等待下次自动恢复

### Full GitOps 部署原理

CD 范围是 **Config 仓库中的所有配置变更**（不仅限于镜像更新），与 ArgoCD / FluxCD 一致。

**部署方式：全量 apply（声明式）** — Master 执行 `kustomize build`，Agent 使用 Dynamic Client + Server-Side Apply 全量提交，K8s 内部判断变更。

```
变更检测流程:

1. GitHub API: GET /repos/{owner}/{repo}/commits?per_page=1 → 最新 SHA
2. 对比缓存 SHA → 相同跳过 / 不同继续
3. GitHub API: compare → 变更文件列表
4. 匹配已配置的部署路径 → 受影响路径
5. 读取路径下文件 → kustomize build → 渲染 YAML
6. Command → Agent → Dynamic Client + SSA
7. 记录 deploy_history
```

### Agent 端 — Server-Side Apply

Agent 不需要 `kubectl` 或 `kustomize` 二进制。接收 Master 渲染后的 YAML：

```go
objects, err := parseMultiDocYAML(manifests)
for _, obj := range objects {
    gvr := getGVR(obj)  // Discovery API 获取 GVR
    _, err := dynamicClient.Resource(gvr).
        Namespace(obj.GetNamespace()).
        Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
            FieldManager: "atlhyper-agent",
        })
}
```

---

## 功能一：GitHub 连接管理

**页面**：`/settings/github` — ConnectionCard

**用户故事**：用户连接/断开 GitHub App，查看连接状态。

### 1.1 API

#### GET /api/github/connection — 获取连接状态

```
响应 200:
{
  "message": "获取成功",
  "data": {
    "connected": true,
    "accountLogin": "wuxiafeng",
    "avatarUrl": "https://github.com/wuxiafeng.png",
    "installationId": 12345678
  }
}
```

#### POST /api/github/connect — 发起 OAuth 连接

```
请求: 无
响应 200:
{
  "data": {
    "authUrl": "https://github.com/login/oauth/authorize?client_id=..."
  }
}
```

前端跳转到 `authUrl`，用户授权后 GitHub 回调 → 前端拿到 `code` → 调用 callback API。

#### POST /api/github/callback — OAuth 回调

```
请求:
{
  "code": "github_oauth_code"
}

响应 200:
{
  "data": {
    "connected": true,
    "accountLogin": "wuxiafeng",
    "avatarUrl": "https://github.com/wuxiafeng.png",
    "installationId": 12345678
  }
}
```

#### DELETE /api/github/connection — 断开连接

```
响应 200:
{
  "message": "已断开 GitHub 连接"
}
```

### 1.2 数据对照

| 前端 Mock (TypeScript) | 后端模型 (Go) | DB 表 |
|------------------------|---------------|-------|
| `MockGitHubConnection.connected` | `github.ConnectionStatus.Connected` | `github_installations.id IS NOT NULL` |
| `MockGitHubConnection.accountLogin` | `github.ConnectionStatus.AccountLogin` | `github_installations.account_login` |
| `MockGitHubConnection.avatarUrl` | `github.ConnectionStatus.AvatarURL` | — (GitHub API 获取) |
| `MockGitHubConnection.installationId` | `github.ConnectionStatus.InstallationID` | `github_installations.installation_id` |

```typescript
// 前端 Mock — mock/github/data.ts
interface MockGitHubConnection {
  connected: boolean;
  accountLogin: string;
  avatarUrl: string;
  installationId: number;
}
```

```go
// 后端 — github/types.go
type ConnectionStatus struct {
    Connected      bool   `json:"connected"`
    AccountLogin   string `json:"accountLogin"`
    AvatarURL      string `json:"avatarUrl"`
    InstallationID int64  `json:"installationId"`
}
```

```sql
CREATE TABLE github_installations (
    id              INTEGER PRIMARY KEY,
    installation_id INTEGER NOT NULL UNIQUE,
    account_login   TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 1.3 后端实现

| 文件 | 职责 |
|------|------|
| `github/app.go` | Installation Token 管理、JWT 签名 |
| `github/oauth.go` | OAuth 流程（AuthURL / Exchange），实现 `auth.OAuthProvider` |
| `database/sqlite/github_install.go` | 安装记录 CRUD |
| `gateway/handler/settings/github.go` | HTTP Handler (connection/connect/callback/disconnect) |

### 1.4 前端实现

| 文件 | 职责 |
|------|------|
| `app/settings/github/components/ConnectionCard.tsx` | 连接状态展示 + 连接/断开按钮 [✅] |
| `mock/github/data.ts` | `MOCK_GITHUB_CONNECTION` [✅] |
| `api/github.ts` | `getConnection()` / `connect()` / `disconnect()` [待实现] |

---

## 功能二：已授权仓库管理

**页面**：`/settings/github` — ReposCard

**用户故事**：查看 GitHub App 授权的仓库列表，为每个仓库开启/关闭映射功能。

### 2.1 API

#### GET /api/github/repos — 获取已授权仓库列表

```
响应 200:
{
  "data": [
    { "fullName": "wuxiafeng/Config", "defaultBranch": "main", "private": true, "mappingEnabled": false },
    { "fullName": "wuxiafeng/Geass", "defaultBranch": "main", "private": false, "mappingEnabled": true },
    { "fullName": "wuxiafeng/atlhyper", "defaultBranch": "main", "private": false, "mappingEnabled": true }
  ]
}
```

#### PUT /api/github/repos/:repo/mapping — 切换映射开关

```
请求:
{
  "enabled": true
}

响应 200:
{
  "message": "映射已开启",
  "data": {
    "repoDirs": ["geass_auth/", "geass_gateway/", "geass_web/", ...]
  }
}
```

开启时后端调用 `ListTopDirs` 返回仓库顶层目录（用于映射行的源码目录下拉）。

#### GET /api/github/repos/:repo/dirs — 获取仓库顶层目录

```
响应 200:
{
  "data": ["geass_auth/", "geass_gateway/", "geass_media/", "geass_web/", ...]
}
```

### 2.2 数据对照

| 前端 Mock | 后端模型 | DB |
|-----------|----------|-----|
| `MockAuthorizedRepo.fullName` | `github.Repository.FullName` | GitHub API（不存 DB） |
| `MockAuthorizedRepo.private` | `github.Repository.Private` | GitHub API |
| `MockAuthorizedRepo.mappingEnabled` | `github.RepoConfig.MappingEnabled` | `repo_config.mapping_enabled` |
| `MOCK_REPO_DIRS[repo]` | `github.ListTopDirs()` 返回值 | GitHub API（不存 DB） |

```typescript
// 前端 Mock
interface MockAuthorizedRepo {
  fullName: string;
  defaultBranch: string;
  private: boolean;
  mappingEnabled: boolean;
}

const MOCK_REPO_DIRS: Record<string, string[]> = {
  "wuxiafeng/Geass": ["geass_auth/", "geass_gateway/", "geass_web/", ...],
  "wuxiafeng/atlhyper": ["atlhyper_master_v2/", "atlhyper_agent_v2/", "atlhyper_web/", ...],
};
```

```go
// 后端
type Repository struct {
    FullName      string `json:"fullName"`
    DefaultBranch string `json:"defaultBranch"`
    Private       bool   `json:"private"`
}

type AuthorizedRepo struct {
    Repository
    MappingEnabled bool `json:"mappingEnabled"`
}
```

### 2.3 后端实现

| 文件 | 职责 |
|------|------|
| `github/api.go` | `ListRepos()` / `ListTopDirs()` |
| `database/sqlite/repo_config.go` | 仓库映射开关持久化 |
| `gateway/handler/settings/github.go` | HTTP Handler (repos/mapping toggle/dirs) |

### 2.4 前端实现

| 文件 | 职责 |
|------|------|
| `app/settings/github/components/ReposCard.tsx` | 仓库列表 + 映射开关 [✅] |
| `mock/github/data.ts` | `MOCK_AUTHORIZED_REPOS` / `MOCK_REPO_DIRS` [✅] |
| `api/github.ts` | `getRepos()` / `toggleMapping()` / `getRepoDirs()` [待实现] |

---

## 功能三：仓库 ↔ Deployment 映射配置

**页面**：`/settings/github` — MappingCard

**用户故事**：为已启用映射的仓库手动配置 Namespace → Deployment → 源码目录的映射关系，供 Code Intel 使用。

### 3.1 配置流程

```
1. 在仓库卡片头部添加 NS（如 geass 添加 "geass" + "geass-web"）
2. 点击「添加映射」→ 新增一行
3. 选择 NS（仅显示已添加的 NS）→ Deployment（按 NS 过滤）→ 源码目录（仓库顶层目录）
4. 镜像 Tag 自动从 Deployment 当前镜像填充（只读）
5. 点击确认按钮
```

**为什么不用自动匹配**：`geass-web` 在 `geass-web` NS，源码目录叫 `geass_web/`，与 AtlHyper 的 `atlhyper_web/` 冲突；同一仓库跨多个 NS，自动匹配无法处理。手动配置准确率 100%，且只需配置一次。

### 3.2 API

#### GET /api/github/repos/:repo/namespaces — 获取仓库已配置的 NS

```
响应 200:
{
  "data": ["geass", "geass-web"]
}
```

#### POST /api/github/repos/:repo/namespaces — 添加 NS

```
请求:
{
  "namespace": "geass"
}

响应 200:
{
  "message": "Namespace 已添加",
  "data": ["geass", "geass-web"]
}
```

#### DELETE /api/github/repos/:repo/namespaces/:ns — 移除 NS

移除时同时删除该 NS 下所有未确认的映射行。

```
响应 200:
{
  "message": "Namespace 已移除"
}
```

#### GET /api/github/mappings — 获取所有映射

```
响应 200:
{
  "data": [
    {
      "id": 1,
      "clusterId": "zgmf-x10a",
      "repo": "wuxiafeng/Geass",
      "namespace": "geass",
      "deployment": "geass-auth",
      "container": "",
      "imagePrefix": "bukahou/geass-auth",
      "sourcePath": "geass_auth/",
      "confirmed": true
    }
  ]
}
```

#### POST /api/github/mappings — 添加映射行

```
请求:
{
  "repo": "wuxiafeng/Geass",
  "namespace": "geass",
  "deployment": "geass-auth",
  "sourcePath": "geass_auth/"
}

响应 201:
{
  "data": { "id": 1, ... }
}
```

#### PUT /api/github/mappings/:id — 更新映射行

```
请求:
{
  "namespace": "geass",
  "deployment": "geass-gateway",
  "sourcePath": "geass_gateway/"
}

响应 200:
{
  "data": { "id": 1, ... }
}
```

#### PUT /api/github/mappings/:id/confirm — 确认映射

```
响应 200:
{
  "message": "映射已确认"
}
```

#### DELETE /api/github/mappings/:id — 删除映射

```
响应 200:
{
  "message": "映射已删除"
}
```

#### GET /api/cluster/deployments — 获取集群 Deployment 列表（已有 API）

映射页面需要按 NS 过滤 Deployment 下拉菜单。此 API 已存在，返回所有 Deployment。

```
响应 200:
{
  "data": [
    { "name": "geass-auth", "namespace": "geass", "image": "bukahou/geass-auth:20260309-abc1234" },
    { "name": "geass-web", "namespace": "geass-web", "image": "bukahou/geass-web:20260308-ghi9012" },
    ...
  ]
}
```

#### GET /api/cluster/namespaces — 获取集群 NS 列表（已有 API）

用于添加 Namespace 时的下拉菜单。

### 3.3 数据对照

| 前端 Mock | 后端模型 | DB 表 |
|-----------|----------|-------|
| `MockRepoMapping.id` | `RepoDeployMapping.ID` | `repo_deploy_mapping.id` |
| `MockRepoMapping.repo` | `RepoDeployMapping.Repo` | `repo_deploy_mapping.repo` |
| `MockRepoMapping.namespace` | `RepoDeployMapping.Namespace` | `repo_deploy_mapping.namespace` |
| `MockRepoMapping.deployment` | `RepoDeployMapping.Deployment` | `repo_deploy_mapping.deployment` |
| `MockRepoMapping.sourcePath` | `RepoDeployMapping.SourcePath` | `repo_deploy_mapping.source_path` |
| `MockRepoMapping.confirmed` | `RepoDeployMapping.Confirmed` | `repo_deploy_mapping.confirmed` |
| `MOCK_REPO_NAMESPACES[repo]` | — | `repo_namespaces` |
| `MOCK_DEPLOYMENTS` | — | Snapshot（内存） |
| `MOCK_NAMESPACES` | — | Snapshot（内存） |

```typescript
// 前端 Mock
interface MockRepoMapping {
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

const MOCK_REPO_MAPPINGS: MockRepoMapping[] = [];  // 初始为空，用户手动添加

const MOCK_REPO_NAMESPACES: Record<string, string[]> = {};  // 初始为空

const MOCK_DEPLOYMENTS = [
  { name: "geass-auth", namespace: "geass", image: "bukahou/geass-auth:20260309-abc1234" },
  ...
];

const MOCK_NAMESPACES = ["geass", "geass-web", "atlhyper", "default"];
```

```go
// 后端
type RepoDeployMapping struct {
    ID          int    `json:"id"`
    ClusterID   string `json:"clusterId"`
    Repo        string `json:"repo"`
    Namespace   string `json:"namespace"`
    Deployment  string `json:"deployment"`
    Container   string `json:"container"`
    ImagePrefix string `json:"imagePrefix"`
    SourcePath  string `json:"sourcePath"`
    Confirmed   bool   `json:"confirmed"`
}
```

```sql
CREATE TABLE repo_deploy_mapping (
    id            INTEGER PRIMARY KEY,
    cluster_id    TEXT NOT NULL,
    repo          TEXT NOT NULL,
    namespace     TEXT NOT NULL,
    deployment    TEXT NOT NULL,
    container     TEXT DEFAULT "",
    image_prefix  TEXT NOT NULL,
    source_path   TEXT DEFAULT "",
    confirmed     BOOLEAN DEFAULT FALSE,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id, namespace, deployment)
);

CREATE TABLE repo_namespaces (
    id         INTEGER PRIMARY KEY,
    repo       TEXT NOT NULL,
    namespace  TEXT NOT NULL,
    UNIQUE(repo, namespace)
);
```

### 3.4 后端实现

| 文件 | 职责 |
|------|------|
| `database/sqlite/repo_mapping.go` | 映射 CRUD |
| `database/sqlite/repo_namespaces.go` | 仓库 NS 关联 CRUD |
| `gateway/handler/settings/github.go` | HTTP Handler (namespaces/mappings) |

### 3.5 前端实现

| 文件 | 职责 |
|------|------|
| `app/settings/github/components/MappingCard.tsx` | NS 标签管理 + 映射行 CRUD [✅] |
| `app/settings/github/components/useGitHubPage.ts` | 状态管理 [✅] |
| `mock/github/data.ts` | Mock 数据 [✅] |
| `api/github.ts` | 映射相关 API 调用 [待实现] |

---

## 功能四：部署配置管理

**页面**：`/admin/deploy` — ConfigCard

**用户故事**：配置 CD 的 Config 仓库、选择需要监控的 kustomize 路径、设置轮询间隔和自动部署开关。

### 4.1 配置流程

```
1. 选择 Config 仓库（从 GitHub 已授权仓库下拉）
2. 后端扫描仓库中所有 kustomization.yaml 路径
3. 下拉选择需要监控的路径（可多选，互斥）
4. 设置轮询间隔（10~600 秒）
5. 开启/关闭自动部署
6. 保存
```

ConfigCard 分为**只读模式**（展示 + 「修改配置」按钮）和**编辑模式**（下拉选择 + 保存/取消）。

### 4.2 API

#### GET /api/deploy/config — 获取部署配置

```
响应 200:
{
  "data": {
    "repoUrl": "wuxiafeng/Config",
    "paths": [
      "zgmf-x10a/k8s-configs/Geass/backend",
      "zgmf-x10a/k8s-configs/Geass/web",
      "zgmf-x10a/k8s-configs/atlhyper/master",
      "zgmf-x10a/k8s-configs/atlhyper/web"
    ],
    "intervalSec": 60,
    "autoDeploy": true,
    "clusterId": "zgmf-x10a"
  }
}

未配置时:
{
  "data": null
}
```

#### PUT /api/deploy/config — 保存部署配置

```
请求:
{
  "repoUrl": "wuxiafeng/Config",
  "paths": ["zgmf-x10a/k8s-configs/Geass/backend", "zgmf-x10a/k8s-configs/Geass/web"],
  "intervalSec": 60,
  "autoDeploy": true,
  "clusterId": "zgmf-x10a"
}

响应 200:
{
  "message": "配置已保存"
}
```

#### GET /api/deploy/kustomize-paths?repo=:repo — 扫描 kustomize 路径

后端通过 GitHub API 遍历仓库目录，找到所有包含 `kustomization.yaml` 的路径。

```
响应 200:
{
  "data": [
    "zgmf-x10a/k8s-configs/Geass/backend",
    "zgmf-x10a/k8s-configs/Geass/web",
    "zgmf-x10a/k8s-configs/atlhyper/master",
    "zgmf-x10a/k8s-configs/atlhyper/agent",
    "zgmf-x10a/k8s-configs/atlhyper/web",
    "zgmf-x10a/k8s-configs/core",
    "zgmf-x10a/k8s-configs/nginx",
    "zgmf-x10a/k8s-configs/redis"
  ]
}
```

#### POST /api/deploy/test-connection — 测试连接

验证 GitHub API 能否访问指定仓库。

```
请求:
{
  "repoUrl": "wuxiafeng/Config"
}

响应 200:
{
  "data": { "ok": true }
}
```

#### GET /api/deploy/repos — 获取可选仓库列表

复用 GitHub 已授权仓库数据。

```
响应 200:
{
  "data": [
    { "fullName": "wuxiafeng/Config", "defaultBranch": "main", "private": true },
    { "fullName": "wuxiafeng/Geass", "defaultBranch": "main", "private": false },
    { "fullName": "wuxiafeng/atlhyper", "defaultBranch": "main", "private": false }
  ]
}
```

### 4.3 数据对照

| 前端 Mock | 后端模型 | DB 表 |
|-----------|----------|-------|
| `MockDeployConfig.repoUrl` | `DeployConfig.RepoURL` | `deploy_config.repo_url` |
| `MockDeployConfig.paths` | `DeployConfig.Paths` | `deploy_config.paths`（JSON 数组） |
| `MockDeployConfig.intervalSec` | `DeployConfig.IntervalSec` | `deploy_config.interval_sec` |
| `MockDeployConfig.autoDeploy` | `DeployConfig.AutoDeploy` | `deploy_config.auto_deploy` |
| `MockDeployConfig.clusterId` | `DeployConfig.ClusterID` | `deploy_config.cluster_id` |
| `MOCK_KUSTOMIZE_PATHS[repo]` | `GitHubClient.ScanKustomizePaths()` | — (GitHub API) |

```typescript
// 前端 Mock
interface MockDeployConfig {
  repoUrl: string;
  paths: string[];
  intervalSec: number;
  autoDeploy: boolean;
  clusterId: string;
}

const MOCK_KUSTOMIZE_PATHS: Record<string, string[]> = {
  "wuxiafeng/Config": [
    "zgmf-x10a/k8s-configs/Geass/backend",
    "zgmf-x10a/k8s-configs/Geass/web",
    ...
  ],
};
```

```go
// 后端
type DeployConfig struct {
    RepoURL     string   `json:"repoUrl"`
    Paths       []string `json:"paths"`
    IntervalSec int      `json:"intervalSec"`
    AutoDeploy  bool     `json:"autoDeploy"`
    ClusterID   string   `json:"clusterId"`
}
```

```sql
CREATE TABLE deploy_config (
    id            INTEGER PRIMARY KEY,
    cluster_id    TEXT NOT NULL,
    repo_url      TEXT NOT NULL,
    paths         TEXT NOT NULL DEFAULT '[]',
    interval_sec  INTEGER DEFAULT 60,
    auto_deploy   BOOLEAN DEFAULT TRUE,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id)
);
```

### 4.4 后端实现

| 文件 | 职责 |
|------|------|
| `deployer/types.go` | DeployConfig 类型 |
| `github/api.go` | `ScanKustomizePaths()` |
| `database/sqlite/deploy_config.go` | 配置 CRUD |
| `gateway/handler/admin/deploy.go` | HTTP Handler (config/kustomize-paths/test-connection) |

### 4.5 前端实现

| 文件 | 职责 |
|------|------|
| `app/admin/deploy/components/ConfigCard.tsx` | 只读/编辑双模式 + 路径下拉 + 分页 [✅] |
| `app/admin/deploy/components/useDeployPage.ts` | 状态管理 [✅] |
| `mock/deploy/data.ts` | `MOCK_DEPLOY_CONFIG` / `MOCK_KUSTOMIZE_PATHS` [✅] |
| `api/deploy.ts` | `getConfig()` / `saveConfig()` / `getKustomizePaths()` / `testConnection()` [待实现] |

---

## 功能五：同步状态与手动同步

**页面**：`/admin/deploy` — StatusCard

**用户故事**：查看每条 kustomize 路径的当前同步状态，对待同步路径执行手动同步。

**待同步触发条件**：轮询检测到 Config 仓库有新 commit，且该 commit 影响了此路径，但尚未完成部署。

### 5.1 API

#### GET /api/deploy/status — 获取路径同步状态

```
响应 200:
{
  "data": [
    {
      "path": "zgmf-x10a/k8s-configs/Geass/backend",
      "namespace": "geass",
      "inSync": false,
      "resourceCount": 12,
      "lastSyncAt": "2026-03-10T10:32:00Z"
    },
    {
      "path": "zgmf-x10a/k8s-configs/Geass/web",
      "namespace": "geass-web",
      "inSync": true,
      "resourceCount": 4,
      "lastSyncAt": "2026-03-10T10:32:00Z"
    }
  ]
}
```

Namespace 由后端从 `kustomize build` 结果提取，前端不输入。
ResourceCount 由后端统计该路径渲染后的 K8s 资源数量。

#### POST /api/deploy/sync — 手动触发同步

```
请求:
{
  "path": "zgmf-x10a/k8s-configs/Geass/backend"
}

响应 200:
{
  "message": "同步已触发"
}
```

### 5.2 数据对照

| 前端 Mock | 后端模型 | 来源 |
|-----------|----------|------|
| `MockPathStatus.path` | `PathStatus.Path` | deploy_config.paths |
| `MockPathStatus.namespace` | `PathStatus.Namespace` | kustomize build 提取 |
| `MockPathStatus.inSync` | `PathStatus.InSync` | 对比 commit SHA |
| `MockPathStatus.resourceCount` | `PathStatus.ResourceCount` | kustomize build 统计 |
| `MockPathStatus.lastSyncAt` | `PathStatus.LastSyncAt` | deploy_history 最新记录 |

```typescript
// 前端 Mock
interface MockPathStatus {
  path: string;
  namespace: string;
  inSync: boolean;
  resourceCount: number;
  lastSyncAt: string;
}
```

```go
// 后端
type PathStatus struct {
    Path          string    `json:"path"`
    Namespace     string    `json:"namespace"`
    InSync        bool      `json:"inSync"`
    ResourceCount int       `json:"resourceCount"`
    LastSyncAt    time.Time `json:"lastSyncAt"`
}
```

### 5.3 后端实现

| 文件 | 职责 |
|------|------|
| `deployer/service.go` | `GetPathStatus()` — 对比缓存 SHA + kustomize build 统计 |
| `deployer/service.go` | `SyncNow()` — 立即触发指定路径的部署 |
| `gateway/handler/admin/deploy.go` | HTTP Handler (status/sync) |

### 5.4 前端实现

| 文件 | 职责 |
|------|------|
| `app/admin/deploy/components/StatusCard.tsx` | 状态列表 + 分页(5) + 立即同步按钮 [✅] |
| `mock/deploy/data.ts` | `MOCK_PATH_STATUS` [✅] |
| `api/deploy.ts` | `getStatus()` / `syncNow()` [待实现] |

---

## 功能六：部署历史与回滚

**页面**：`/admin/deploy` — HistoryCard

**用户故事**：查看部署历史（简略表格），点击查看详情（弹窗），对非回滚记录执行一键回滚。

### 6.1 API

#### GET /api/deploy/history — 获取部署历史

```
请求参数: ?page=0&pageSize=8
响应 200:
{
  "data": [
    {
      "id": 1,
      "clusterId": "zgmf-x10a",
      "path": "Geass/backend",
      "namespace": "geass",
      "commitSha": "def5678",
      "commitMessage": "refactor: routing matching logic",
      "deployedAt": "2026-03-10T10:32:15Z",
      "trigger": "auto",
      "status": "success",
      "durationMs": 2340,
      "resourceTotal": 12,
      "resourceChanged": 3,
      "errorMessage": null
    }
  ],
  "total": 20
}
```

#### GET /api/deploy/history/:id — 获取部署详情

与列表项相同字段，单独查询用于详情弹窗。

```
响应 200:
{
  "data": {
    "id": 2,
    "path": "Geass/backend",
    "namespace": "geass",
    "commitSha": "abc1234",
    "commitMessage": "fix: JWT token refresh bug — 修复 Token 过期后无法自动刷新导致 401 的问题",
    "deployedAt": "2026-03-09T14:33:20Z",
    "trigger": "auto",
    "status": "failed",
    "durationMs": 5120,
    "resourceTotal": 12,
    "resourceChanged": 1,
    "errorMessage": "Apply failed: Deployment geass-auth — container image pull error: ImagePullBackOff ..."
  }
}
```

#### POST /api/deploy/rollback — 回滚

```
请求:
{
  "path": "Geass/backend",
  "targetCommitSha": "old7890"
}

响应 200:
{
  "message": "回滚已触发"
}
```

回滚流程：
1. 从 deploy_history 获取目标 commit 的版本
2. GitHub API 读取该 commit 版本的文件
3. kustomize build → Command → Agent → SSA
4. 记录 deploy_history (trigger = "rollback")

> 回滚不修改 Config 仓库。回滚是临时措施，下次轮询检测到 Config 仓库变更后会重新部署。

### 6.2 数据对照

| 前端 Mock | 后端模型 | DB 表 |
|-----------|----------|-------|
| `MockDeployRecord.id` | `DeployRecord.ID` | `deploy_history.id` |
| `MockDeployRecord.path` | `DeployRecord.Path` | `deploy_history.path` |
| `MockDeployRecord.namespace` | `DeployRecord.Namespace` | `deploy_history.namespace` |
| `MockDeployRecord.commitSha` | `DeployRecord.CommitSHA` | `deploy_history.commit_sha` |
| `MockDeployRecord.commitMessage` | `DeployRecord.CommitMessage` | `deploy_history.commit_message` |
| `MockDeployRecord.deployedAt` | `DeployRecord.DeployedAt` | `deploy_history.deployed_at` |
| `MockDeployRecord.trigger` | `DeployRecord.Trigger` | `deploy_history.trigger` |
| `MockDeployRecord.status` | `DeployRecord.Status` | `deploy_history.status` |
| `MockDeployRecord.durationMs` | `DeployRecord.DurationMs` | `deploy_history.duration_ms` |
| `MockDeployRecord.resourceTotal` | `DeployRecord.ResourceTotal` | `deploy_history.resource_total` |
| `MockDeployRecord.resourceChanged` | `DeployRecord.ResourceChanged` | `deploy_history.resource_changed` |
| `MockDeployRecord.errorMessage` | `DeployRecord.ErrorMessage` | `deploy_history.error_message` |

```typescript
// 前端 Mock
type DeployTrigger = "auto" | "manual" | "rollback";

interface MockDeployRecord {
  id: number;
  clusterId: string;
  path: string;
  namespace: string;
  commitSha: string;
  commitMessage: string;
  deployedAt: string;
  trigger: DeployTrigger;
  status: "pending" | "success" | "failed";
  durationMs: number;
  resourceTotal: number;
  resourceChanged: number;
  errorMessage?: string;
}
```

```go
// 后端
type DeployRecord struct {
    ID              int       `json:"id"`
    ClusterID       string    `json:"clusterId"`
    Path            string    `json:"path"`
    Namespace       string    `json:"namespace"`
    CommitSHA       string    `json:"commitSha"`
    CommitMessage   string    `json:"commitMessage"`
    DeployedAt      time.Time `json:"deployedAt"`
    Trigger         string    `json:"trigger"`
    Status          string    `json:"status"`
    DurationMs      int       `json:"durationMs"`
    ResourceTotal   int       `json:"resourceTotal"`
    ResourceChanged int       `json:"resourceChanged"`
    ErrorMessage    string    `json:"errorMessage,omitempty"`
}
```

```sql
CREATE TABLE deploy_history (
    id                INTEGER PRIMARY KEY,
    cluster_id        TEXT NOT NULL,
    path              TEXT NOT NULL,
    namespace         TEXT NOT NULL,
    commit_sha        TEXT NOT NULL,
    commit_message    TEXT,
    deployed_at       DATETIME NOT NULL,
    trigger           TEXT NOT NULL DEFAULT 'auto',
    status            TEXT DEFAULT 'pending',
    duration_ms       INTEGER DEFAULT 0,
    resource_total    INTEGER DEFAULT 0,
    resource_changed  INTEGER DEFAULT 0,
    error_message     TEXT
);
CREATE INDEX idx_deploy_history_lookup ON deploy_history(cluster_id, path, deployed_at DESC);
```

### 6.3 后端实现

| 文件 | 职责 |
|------|------|
| `deployer/service.go` | `GetHistory()` / `Rollback()` |
| `deployer/types.go` | DeployRecord 类型 |
| `database/sqlite/deploy_history.go` | 历史 CRUD |
| `gateway/handler/admin/deploy.go` | HTTP Handler (history/rollback) |

### 6.4 前端实现

| 文件 | 职责 |
|------|------|
| `app/admin/deploy/components/HistoryCard.tsx` | 简略表格 + 详情弹窗 + 分页(8) + 回滚按钮 [✅] |
| `app/admin/deploy/components/Pagination.tsx` | 通用分页组件 [✅] |
| `mock/deploy/data.ts` | `MOCK_DEPLOY_HISTORY` [✅] |
| `api/deploy.ts` | `getHistory()` / `getHistoryDetail()` / `rollback()` [待实现] |

---

## 功能七：AI Tool 扩展

**用户故事**：AI 在分析事件时可查询部署历史、读取代码、搜索代码、回滚部署。

### 7.1 Tool 定义

```json
{
  "name": "get_deploy_history",
  "description": "查询指定路径的部署历史（commit + 状态 + 关联 PR）",
  "parameters": {
    "path": { "type": "string", "description": "kustomize 路径" },
    "limit": { "type": "integer", "default": 5 }
  }
}
```

```json
{
  "name": "rollback_deployment",
  "description": "将指定路径回滚到历史 commit 版本（需用户确认）",
  "parameters": {
    "path": { "type": "string" },
    "target_commit_sha": { "type": "string" }
  }
}
```

```json
{
  "name": "github_read_file",
  "description": "读取关联仓库中的文件内容",
  "parameters": {
    "repo": { "type": "string" },
    "path": { "type": "string" },
    "branch": { "type": "string", "default": "main" }
  }
}
```

```json
{
  "name": "github_search_code",
  "description": "在关联仓库中搜索代码",
  "parameters": {
    "repo": { "type": "string" },
    "query": { "type": "string" }
  }
}
```

```json
{
  "name": "github_recent_commits",
  "description": "查看仓库最近的 commits",
  "parameters": {
    "repo": { "type": "string" },
    "path": { "type": "string", "description": "限定路径（可选）" },
    "limit": { "type": "integer", "default": 10 }
  }
}
```

**安全约束**：`rollback_deployment` 是写操作，AI 只能提出建议，必须经用户确认后才执行。

### 7.2 Code Intel 数据流

deploy_history 是 CD 和 Code Intel 的衔接点：

```
CD 写入 deploy_history:
  commit_sha = "def5678"

Code Intel 异步补充:
  GitHub API → GetPRByCommit("wuxiafeng/Geass", "def5678")
  → PR#42 "重构路由匹配逻辑"
  → 更新 deploy_history: pr_number=42, pr_title="..."
```

### 7.3 后端实现

| 文件 | 职责 |
|------|------|
| `ai/prompts/tools.go` | Tool 定义 |
| `ai/tool.go` | Tool 调用分发（deploy_history → deployer / github_* → github client） |
| `github/api.go` | `ReadFile()` / `SearchCode()` / `ListCommits()` / `GetPRByCommit()` |

---

## 实施阶段

| 阶段 | 功能 | 依赖 |
|------|------|------|
| **Phase 1** | 功能一（GitHub 连接） + 功能二（仓库管理） | 无 |
| **Phase 2** | 功能三（映射配置） | Phase 1 |
| **Phase 3** | 功能四（部署配置） | Phase 1 |
| **Phase 4** | 功能五（同步状态） + 功能六（部署历史） | Phase 3 |
| **Phase 5** | Agent apply_manifests | Phase 4 |
| **Phase 6** | 功能七（AI Tool） | Phase 4 + Phase 2 |

前端页面均已完成 [✅]，后续按 Phase 顺序实现后端 API + Agent + 前端 API 层对接。

---

## 文件变更清单

`[新增]` = 新建文件 | `[修改]` = 修改已有文件 | `[✅]` = 已完成

### Master（atlhyper_master_v2/）

```
atlhyper_master_v2/
├── master.go                              [修改] 初始化 Deployer + GitHub 模块 + OAuth 路由
├── deployer/                              [新增] CD 部署模块
│   ├── interfaces.go                          Deployer 接口（Start/Stop/SyncNow/GetPathStatus/GetHistory/Rollback）
│   ├── service.go                             轮询循环、commit SHA 对比、变更检测、触发部署
│   ├── kustomize.go                           kustomize build 本地渲染
│   └── types.go                               DeployConfig, PathStatus, DeployRecord
├── auth/
│   ├── interfaces.go                      [修改] 新增通用 OAuthProvider 接口 + OAuthUser 类型
│   └── types.go                           [修改] OAuth 相关通用类型
├── github/                                [新增] GitHub 集成模块（App + API + OAuth 全部收敛）
│   ├── interfaces.go                          GitHubClient 接口
│   ├── app.go                                 GitHub App 身份管理（JWT + Installation Token 自动刷新）
│   ├── api.go                                 GitHub REST API 调用
│   ├── oauth.go                               实现 auth.OAuthProvider
│   └── types.go                               Repository, Commit, PullRequest, ChangedFile, FileEntry
├── database/
│   └── sqlite/
│       ├── migrations.go                  [修改] 新增 5 张表
│       ├── deploy_config.go               [新增] 部署配置 CRUD
│       ├── deploy_history.go              [新增] 部署历史 CRUD
│       ├── github_install.go              [新增] GitHub App 安装记录 CRUD
│       ├── repo_mapping.go                [新增] 仓库映射 CRUD
│       └── repo_namespaces.go             [新增] 仓库 NS 关联 CRUD
├── gateway/
│   ├── routes.go                          [修改] 新增路由
│   └── handler/
│       ├── admin/
│       │   └── deploy.go                  [新增] 部署管理 API
│       └── settings/
│           └── github.go                  [新增] GitHub 设置 API
├── service/
│   └── interfaces.go                      [修改] 新增 Deployer 相关查询方法
└── ai/
    └── prompts/
        └── tools.go                       [修改] 新增 AI Tool 定义
```

### Agent（atlhyper_agent_v2/）

```
atlhyper_agent_v2/
├── sdk/
│   ├── interfaces.go                      [修改] K8sClient 新增 ApplyManifests 方法
│   └── impl/
│       └── k8s/
│           └── apply.go                   [新增] Dynamic Client + SSA 实现
│               · parseMultiDocYAML()          多文档 YAML → []Unstructured
│               · getGVR()                     Discovery API 解析 GVR
│               · ApplyManifests()             逐资源 SSA + 结果统计
├── service/
│   ├── interfaces.go                      [修改] CommandService 新增 apply_manifests handler
│   └── command/
│       └── apply.go                       [新增] apply_manifests Command handler
└── model/
    └── command_types.go                   [修改] 新增 ApplyManifestsParams / ApplyManifestsResult
```

### Web 前端（atlhyper_web/）

```
atlhyper_web/src/
├── api/
│   ├── deploy.ts                          [待实现] 部署管理 API 调用
│   └── github.ts                          [待实现] GitHub 集成 API 调用
├── app/
│   ├── admin/deploy/                      [✅] 部署管理页面
│   │   ├── page.tsx
│   │   └── components/
│   │       ├── ConfigCard.tsx                 Config 仓库配置（只读/编辑 + 路径分页）
│   │       ├── StatusCard.tsx                 同步状态（per-path + 分页）
│   │       ├── HistoryCard.tsx                部署历史（简略表格 + 详情弹窗 + 分页）
│   │       ├── Pagination.tsx                 通用分页组件
│   │       ├── useDeployPage.ts               页面状态 Hook
│   │       └── index.ts
│   └── settings/github/                   [✅] GitHub 设置页面
│       ├── page.tsx
│       └── components/
│           ├── ConnectionCard.tsx             连接状态
│           ├── ReposCard.tsx                  已授权仓库 + 映射开关
│           ├── MappingCard.tsx                NS 标签 + 手动映射行
│           ├── useGitHubPage.ts               页面状态 Hook
│           └── index.ts
├── mock/
│   ├── deploy/data.ts                     [✅] Deploy Mock 数据
│   └── github/data.ts                     [✅] GitHub Mock 数据
├── types/i18n.ts                          [✅] 翻译类型
└── i18n/locales/{zh,ja}.ts               [✅] 翻译文件
```

---

## 附录

### A. CI（外部，不在 AtlHyper 范围内）

CI 完全在 GitHub Actions 中运行。最后一步更新 Config 仓库的 `kustomization.yaml`（image tag）。

**Image Tag 推荐格式**：`{YYYYMMDD}-{commitSHA7}`（如 `20260310-def5678`）

```go
// 解析规则：提取最后一个 "-" 后的部分
func ExtractCommitSHA(tag string) string {
    if i := strings.LastIndex(tag, "-"); i != -1 { return tag[i+1:] }
    return ""
}
```

### B. Config 仓库结构规范

所有部署统一使用 kustomize，按 NS / 部署单元拆分目录：

```
Config/zgmf-x10a/k8s-configs/
├── Geass/
│   ├── backend/              ← NS: geass（6 个后端微服务）
│   │   ├── kustomization.yaml
│   │   ├── geass-auth.yaml
│   │   └── ...
│   └── web/                  ← NS: geass-web
│       ├── kustomization.yaml
│       └── ...
├── atlhyper/
│   ├── master/               ← NS: atlhyper
│   ├── agent/
│   └── web/
├── core/                     ← 基础设施
├── nginx/
└── redis/
```

原则：每个 kustomize 目录 = 一个部署单元。禁止裸 YAML 直接部署。

### C. GitHub App 注册

| 权限 | 级别 | 用途 |
|------|------|------|
| Contents | Read-only | 读取代码、目录、文件 |
| Pull requests | Read-only | 查询 PR 详情 |
| Metadata | Read-only | 仓库基本信息 |

不订阅 Webhook（Master 无公网端点）。

### D. 约束与风险

| 项目 | 说明 |
|------|------|
| **认证统一** | CD 和 Code Intel 共用 GitHub App Installation Token |
| **Token 安全** | GitHub App 私钥 / Client Secret 加密存储（common/crypto） |
| **API 限流** | GitHub API 5000 req/h；轮询每次 1 req（SHA 对比），AI Tool 调用需缓存 |
| **代码量控制** | github_read_file 单次不超过 500 行，超长截断 |
| **回滚安全** | rollback_deployment 是写操作，必须经用户确认 |
| **kustomize 依赖** | Master 需要 kustomize 二进制，Docker 镜像需包含 |
| **SSA 版本** | 要求 K8s 1.22+（当前 K3s 满足） |
| **网络容错** | 轮询失败静默跳过，不触发告警风暴 |
| **无 Webhook** | Master 无公网 IP，纯间隔轮询 |

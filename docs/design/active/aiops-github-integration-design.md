# GitHub 集成 + CD + 代码智能 设计

> 状态: active | 创建: 2026-03-08 | 更新: 2026-03-10

## 背景

### 问题 1：AI 缺乏代码上下文

AI 分析事件时只能访问运行时数据（K8s 状态、Pod 日志、指标），无法理解应用代码。
实际案例：Geass Gateway 遭受 WordPress 扫描，AI 无法识别这是 Spring Boot 项目，给出了错误的修复建议。

### 问题 2：部署与异常无法关联

AIOps 检测到异常后，不知道是否刚发生过部署。运维人员需要自己去 GitHub 查 commit 记录、比对时间线。

### 问题 3：部署和回滚依赖手动操作

部署流程完全手动（改 YAML → kubectl apply），回滚也需要人工找到上一个版本再手动操作。

## 目标

| 能力 | 当前 | 目标 |
|------|------|------|
| **认证** | 本地账号 | 本地账号 + GitHub OAuth 登录 |
| **代码感知** | 无 | AI 可读取关联仓库的源码、commits、PRs |
| **部署** | 手动 kubectl | Master 自动调谐（Config 仓库驱动） |
| **部署感知** | 只知道当前 image | 完整部署历史 + commit/PR 关联 |
| **回滚** | 手动改 YAML | AtlHyper 一键回滚 |
| **变更关联** | 无 | 「异常发生前 3 分钟部署了 PR#42」 |

---

## 核心架构：三关注点分离

```
┌─────────────┐     ┌──────────────────────┐     ┌──────────────────────┐
│     CI      │     │        CD            │     │    Code Intel        │
│   (外部)    │     │   (AtlHyper Master)  │     │   (AtlHyper AIOps)  │
│             │     │                      │     │                      │
│  测试       │     │  轮询 Config 仓库     │     │  Commits / Diffs     │
│  编译       │     │  对比期望 vs 实际     │     │  PRs                 │
│  构建镜像   │     │  Command → Agent     │     │  变更与异常关联       │
│  推送镜像   │     │  记录 deploy_history  │     │  AI 根因推理         │
│  更新 Config│     │                      │     │                      │
│             │     │  页面: /admin/deploy  │     │  配置: /settings/github │
│  GitHub     │     │                      │     │  展示: AIOps 事件详情  │
│  Actions    │     │                      │     │                      │
└──────┬──────┘     └──────────┬───────────┘     └──────────┬───────────┘
       │                       │                            │
       │                 无依赖关系                    无依赖关系
       │                       │                            │
       └───────────────────────┴────────────────────────────┘
                      唯一交集: image tag
              CI 产出 tag → CD 用 tag 部署 → Code Intel 用 tag 反查 commit
```

**三者完全独立**，唯一交集是 image tag。可以独立实施、独立运行。

### GitHub App 统一认证

CD 和 Code Intel **共用同一个 GitHub App 安装**。用户只需一次操作：

```
用户点击 [连接 GitHub] → 跳转 GitHub → 安装 App 时勾选所有需要的仓库
  · Config 仓库（私有，CD 用于 git clone/pull）
  · Geass 等代码仓库（Code Intel 用于读取 commits/PR/代码）
→ 回调 → 完成
```

GitHub App Installation Token 可同时用于：
- **git clone/pull**（CD 模块拉取 Config 仓库）— `https://x-access-token:{token}@github.com/user/Config.git`
- **GitHub REST API**（Code Intel 读取代码变更）

> 为什么不用 SSH Key？SSH Key 需要用户手动复制公钥到 GitHub 仓库 Deploy keys，体验差。
> GitHub App 只需点击授权，体验统一，且 Token 自动刷新。

---

## 一、CI（外部，不在 AtlHyper 范围内）

CI 完全在 GitHub Actions 中运行，AtlHyper 不参与。

```
push → test → build image → push to DockerHub
                           → 更新 Config 仓库的 kustomization.yaml (git push)
```

CI 的最后一步是更新 Config 仓库（如 `zgmf-x10a/k8s-configs/Geass/kustomization.yaml`），
将新的 image tag 写入。这一步在 GitHub Actions 云端 runner 即可完成，不需要访问集群。

CI 的设计详见各应用仓库自身的 CI/CD 文档（如 Geass: `docs/design/active/cicd-design.md`）。

### Image Tag 规范

Image tag 是 CI（外部）与 CD（AtlHyper）之间的契约。AtlHyper CD 对 tag 格式有以下需求：

| 需求方 | 需要从 tag 中获取什么 |
|--------|----------------------|
| **CD 调谐** | 字符串对比（期望 ≠ 实际 → 部署） |
| **CD 回滚** | 时间顺序（识别上一个版本） |
| **Code Intel** | commit SHA（反查 GitHub commit/PR） |
| **人类可读** | 一眼判断时间和提交 |

**推荐格式**：`{YYYYMMDD}-{commitSHA7}`

```
示例: 20260310-def5678
      ~~~~~~~~ ~~~~~~~
      日期前缀   commit SHA 前 7 位
```

- **日期前缀**：按字典序排列即时间序，回滚时直接取上一条 deploy_history
- **commit SHA**：Code Intel 提取后调用 `GetPRByCommit()` 反查关联 PR
- **不使用语义版本号**：CD 场景下镜像由 CI 产出，tag 是标识符而非版本号

**解析规则**：AtlHyper 不强制格式，通过提取最后一个 `-` 后的部分获取 commit SHA：

```go
// deployer/parser.go
func ExtractCommitSHA(tag string) string {
    if i := strings.LastIndex(tag, "-"); i != -1 {
        return tag[i+1:]
    }
    return ""
}
```

兼容其他格式（如 `v1.0.0-abc1234`、`main-abc1234`），只要 commit SHA 在末尾即可。

CI 端在构建镜像时按此格式生成 tag：

```yaml
# GitHub Actions 示例
- name: Set image tag
  run: echo "TAG=$(date +%Y%m%d)-$(git rev-parse --short=7 HEAD)" >> $GITHUB_ENV
```

---

## 二、CD — Master Deployer 模块

### 2.1 设计原理

AtlHyper Master 不接收外部 Webhook（Master 是集群内部应用，无公网端点）。
CD 采用 **GitOps 调谐模式**：Master 定期 git pull Config 仓库，对比期望状态与实际状态，有差异则通过 Command 机制部署。

```
Config 仓库 (期望状态)          集群 (实际状态)
kustomization.yaml:              Snapshot 上报:
  geass-auth: 0310-def5678         geass-auth: 0309-abc1234
                    │                        │
                    └───── Master 对比 ───────┘
                              │
                          有差异！
                              │
                              ▼
                    Command → Agent → kubectl apply
```

### 2.2 Config 仓库认证方式

CD 使用 GitHub App Installation Token 通过 HTTPS 协议访问 Config 仓库：

```
git clone https://x-access-token:{installation_token}@github.com/user/Config.git
```

- Installation Token 由 GitHub App 模块统一管理，自动刷新（有效期 1 小时）
- CD 模块通过 `GitHubClient.GetInstallationToken()` 获取当前有效 Token
- 无需用户手动配置 SSH Key 或 PAT

### 2.3 工作流程

```
Master 启动
  │
  ▼
Deployer 初始化
  · 从 Database 读取 Config 仓库配置
  · 从 GitHub App 获取 Installation Token
  · git clone Config 仓库到本地缓存目录
  · 启动定时调谐循环
  │
  ▼
定时调谐（每 N 秒，可在 Settings 配置）
  │
  ├── 1. git pull Config 仓库（获取最新期望状态）
  │
  ├── 2. 解析 kustomization.yaml → 期望的 image tags
  │      { "geass-auth": "0310-def5678", "geass-gateway": "0310-def5678", ... }
  │
  ├── 3. 从 Store 获取当前 Snapshot → 实际的 image tags
  │      { "geass-auth": "0309-abc1234", "geass-gateway": "0310-def5678", ... }
  │
  ├── 4. 对比差异
  │      geass-auth: 期望 0310-def5678 ≠ 实际 0309-abc1234 → 需要部署
  │      geass-gateway: 期望 = 实际 → 跳过
  │
  ├── 5. 生成 Command 发送给 Agent
  │      Command{action: "apply_kustomize", path: "zgmf-x10a/k8s-configs/Geass/"}
  │      或逐个: Command{action: "set_image", deployment: "geass-auth", image: "..."}
  │
  └── 6. 记录 deploy_history（状态: pending → success/failed）
```

### 2.4 回滚流程

```
用户在 /admin/deploy 点击「回滚 geass-auth」
  │
  ▼
Master Deployer:
  1. 从 deploy_history 查上一个版本的 tag
  2. 修改本地 Config 仓库缓存的 kustomization.yaml
  3. git commit + git push（推回 Config 仓库）
  │
  ▼
立即触发一次调谐（不等周期）:
  · 检测到期望状态变了
  · Command → Agent → kubectl apply
  · 记录 deploy_history (is_rollback = true)
```

### 2.5 模块结构

```
atlhyper_master_v2/
├── deployer/
│   ├── interfaces.go        # Deployer 接口
│   ├── service.go           # 核心逻辑（调谐循环、差异对比、触发部署）
│   ├── gitops.go            # Git 操作（clone/pull/commit/push Config 仓库）
│   ├── parser.go            # 解析 kustomization.yaml 提取 image tags
│   └── types.go             # DeployConfig, DesiredState, ReconcileResult 等
```

### 2.6 接口定义

```go
// deployer/interfaces.go

type Deployer interface {
    // 启动调谐循环
    Start(ctx context.Context) error
    // 停止
    Stop()
    // 立即触发一次调谐（回滚时使用）
    ReconcileNow(ctx context.Context) error

    // 获取当前状态（期望 vs 实际）
    GetStatus(ctx context.Context, clusterID string) ([]DeployStatus, error)
    // 获取部署历史
    GetHistory(ctx context.Context, clusterID, namespace, deployment string, limit int) ([]DeployRecord, error)

    // 回滚到指定版本
    Rollback(ctx context.Context, clusterID, namespace, deployment, targetImage string) error
}
```

```go
// deployer/types.go

type DeployConfig struct {
    RepoURL     string // "wuxiafeng/Config"（从已授权仓库中选择）
    Branch      string // "main"
    DeployPath  string // "zgmf-x10a/k8s-configs/Geass"
    IntervalSec int    // 调谐间隔（秒）
    AutoDeploy  bool   // 自动部署开关
    ClusterID   string // 关联的集群
    // 认证通过 GitHub App Installation Token，由 GitHubClient 统一管理
}

type DeployStatus struct {
    Namespace    string
    Deployment   string
    ActualImage  string // 集群中实际运行的 image
    DesiredImage string // Config 仓库声明的 image
    InSync       bool   // 是否同步
}

type DeployRecord struct {
    ID          int
    ClusterID   string
    Namespace   string
    Deployment  string
    OldImage    string
    NewImage    string
    CommitSHA   string // 从 image tag 提取
    PRNumber    int    // GitHub API 反查（Code Intel 填充）
    PRTitle     string
    DetectedAt  time.Time
    IsRollback  bool
    Status      string // "pending" / "success" / "failed"
}
```

### 2.7 数据模型

```sql
-- 部署配置（/admin/deploy 页面管理）
CREATE TABLE deploy_config (
    id            INTEGER PRIMARY KEY,
    cluster_id    TEXT NOT NULL,
    repo_url      TEXT NOT NULL,           -- Config 仓库（从已授权仓库中选择）
    branch        TEXT DEFAULT 'main',
    deploy_path   TEXT NOT NULL,           -- kustomization.yaml 所在子目录
    interval_sec  INTEGER DEFAULT 60,     -- 调谐间隔
    auto_deploy   BOOLEAN DEFAULT TRUE,   -- 自动部署开关
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id)
);
-- 认证通过 GitHub App Installation Token（统一管理，无需单独配置）

-- 部署历史
CREATE TABLE deploy_history (
    id            INTEGER PRIMARY KEY,
    cluster_id    TEXT NOT NULL,
    namespace     TEXT NOT NULL,
    deployment    TEXT NOT NULL,
    old_image     TEXT,
    new_image     TEXT NOT NULL,
    commit_sha    TEXT,
    pr_number     INTEGER,
    pr_title      TEXT,
    detected_at   DATETIME NOT NULL,
    is_rollback   BOOLEAN DEFAULT FALSE,
    status        TEXT DEFAULT 'pending'   -- pending / success / failed
);
CREATE INDEX idx_deploy_lookup ON deploy_history(cluster_id, namespace, deployment, detected_at);
```

### 2.8 层级职责

| 层 | 职责 | 不做 |
|---|---|---|
| **Master Deployer** | git pull Config、解析期望状态、对比差异、决策部署、回滚逻辑、记录历史 | 不直接 kubectl |
| **Agent** | 执行 kubectl apply / set image、上报 Snapshot | 不做决策 |
| **Web /admin/deploy** | Config 仓库配置 + 调谐参数 + 同步状态 + 部署历史 + 回滚 | 不触发主动部署 |

> 注：CD 相关设置和管理统一在 `/admin/deploy` 一个页面，不拆分到 `/settings/`。

### 2.9 Agent 新增 Command

```go
// Agent 端新增两个 handler

// 1. apply_kustomize: 整体部署
case "apply_kustomize":
    path := params["path"].(string)
    // kubectl apply -k {path}
    err := k8sClient.ApplyKustomize(ctx, path)

// 2. set_image: 单个服务镜像更新（回滚场景）
case "set_image":
    namespace := params["namespace"].(string)
    deployment := params["deployment"].(string)
    container := params["container"].(string)
    image := params["image"].(string)
    // kubectl set image deployment/{deployment} {container}={image} -n {namespace}
    err := k8sClient.SetDeploymentImage(ctx, namespace, deployment, container, image)
```

### 2.10 Web 页面 — /admin/deploy

CD 配置和部署管理统一在一个页面，分为三个区块：

**前置条件**：GitHub App 已连接（在 `/settings/github` 页面完成）。未连接时显示提示引导用户先去连接。

```
┌──────────────────────────────────────────────────────────────┐
│  部署管理                                                      │
│                                                                │
│  ┌── Config 仓库配置 ──────────────────────────────────────┐  │
│  │                                                          │  │
│  │  Config 仓库:  [wuxiafeng/Config         ▼]  ← 已授权仓库下拉 │
│  │  分支:         [main                      ]              │  │
│  │  部署路径:     [zgmf-x10a/k8s-configs/Geass]             │  │
│  │                                                          │  │
│  │  轮询间隔:     [60 ▼] 秒                                 │  │
│  │  自动部署:     [开启 ▼]   ← 关闭后只检测差异不自动部署    │  │
│  │                                                          │  │
│  │                                   [测试连接]  [保存]     │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌── 当前状态 ──────────────────────────────────────────────┐  │
│  │ 服务           实际版本          期望版本      状态        │  │
│  │ ──────────────────────────────────────────────────────── │  │
│  │ geass-auth     0309-abc1234     0310-def5678  ⚠ 待部署   │  │
│  │ geass-gateway  0310-def5678     0310-def5678  ✅ 同步    │  │
│  │ geass-media    0310-def5678     0310-def5678  ✅ 同步    │  │
│  │ geass-web      0308-ghi9012     0308-ghi9012  ✅ 同步    │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                │
│  ┌── 部署历史 ──────────────────────────────────────────────┐  │
│  │ 时间          服务         变更              状态         │  │
│  │ ──────────────────────────────────────────────────────── │  │
│  │ 10:32        geass-gate   0309→0310-def5678  ✅ 成功     │  │
│  │ 10:32        geass-media  0309→0310-def5678  ✅ 成功     │  │
│  │ 昨天 14:32   geass-auth   0308→0309-abc1234  ✅ 成功     │  │
│  │                                                          │  │
│  │                              [回滚 geass-auth ▼]         │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

**页面逻辑**：
- Config 仓库下拉菜单从 GitHub App 已授权的仓库列表中选择（调用 `ListRepos`）
- 未配置 Config 仓库时，"当前状态"和"部署历史"区块显示空状态提示
- "测试连接"验证 git clone 是否成功

---

## 三、Code Intel — GitHub 集成（AIOps 子模块）

Code Intel 与 CD 完全独立。CD 负责部署，Code Intel 负责理解代码变更并关联到运行时事件。

### 3.1 为什么用 GitHub App 而不是 PAT

AtlHyper 未来要支持多用户 + OAuth 登录，GitHub App 是唯一合理选择。

| | PAT | GitHub App |
|---|---|---|
| **身份** | 代表某个用户 | 独立应用身份 |
| **OAuth 登录** | 不支持 | 支持 |
| **多租户** | 一个 token 一个用户 | 不同用户各自安装授权 |
| **权限** | 用户级别 | 安装时声明最小权限 |
| **Token 管理** | 手动创建、手动续期 | 自动刷新（1 小时短期 token） |

### 3.2 GitHub App 注册

在 GitHub Developer Settings 创建 GitHub App，声明权限：

| 权限 | 级别 | 用途 |
|------|------|------|
| Contents | Read-only | 读取代码、目录结构 |
| Pull requests | Read-only | 查询 PR 详情 |
| Metadata | Read-only | 仓库基本信息 |

> 注：不订阅 Webhook 事件。Master 没有公网端点，无法接收 Webhook。
> Code Intel 采用按需查询模式（deploy_history 触发 → GitHub API 反查）。

### 3.3 OAuth 登录流程

```
用户点击 [用 GitHub 登录]
    │
    ▼
跳转 GitHub OAuth 授权页
  https://github.com/login/oauth/authorize?client_id=...
    │
    ▼
用户授权 → 回调 Web 前端（Next.js）
  → Web 转发 code 给 Master API
    │
    ▼
Master 用 code 换取 access_token
  → 获取用户信息（login, email, avatar）
  → 创建/关联本地用户
  → 返回 AtlHyper session
```

> OAuth 回调走 Web 前端的 Ingress（已有公网端点），不需要 Master 暴露额外端口。

### 3.4 认证模块扩展

```go
// auth/oauth/interfaces.go
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

### 3.5 GitHub API Client

```go
// github/interfaces.go
type GitHubClient interface {
    // 认证（CD + Code Intel 共用）
    GetInstallationToken(ctx context.Context) (string, error) // CD 用于 git clone/pull

    // 仓库信息
    ListRepos(ctx context.Context, installationID int64) ([]Repository, error)
    ListTopDirs(ctx context.Context, repo, branch string) ([]string, error)

    // Commits & PRs（Code Intel 核心）
    ListCommits(ctx context.Context, repo, branch string, limit int) ([]Commit, error)
    GetPRByCommit(ctx context.Context, repo, sha string) (*PullRequest, error)
    GetCommitFiles(ctx context.Context, repo, sha string) ([]string, error)

    // 代码读取（供 AI Tool 使用）
    ReadFile(ctx context.Context, repo, path, branch string) (string, error)
    SearchCode(ctx context.Context, repo, query string) ([]CodeSearchResult, error)
}
```

```go
// github/types.go
type Repository struct {
    FullName      string // "wuxiafeng/Geass"
    DefaultBranch string // "main"
    Private       bool
}

type Commit struct {
    SHA       string
    Message   string
    Author    string
    Timestamp time.Time
    Files     []string
}

type PullRequest struct {
    Number    int
    Title     string
    Author    string
    MergedAt  *time.Time
    Files     []string
}
```

### 3.6 仓库 ↔ 部署映射

#### 数据模型

```sql
-- GitHub App 安装记录
CREATE TABLE github_installations (
    id              INTEGER PRIMARY KEY,
    installation_id INTEGER NOT NULL UNIQUE,
    account_login   TEXT NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 仓库 ↔ 部署映射
CREATE TABLE repo_deploy_mapping (
    id            INTEGER PRIMARY KEY,
    cluster_id    TEXT NOT NULL,
    repo          TEXT NOT NULL,           -- "wuxiafeng/Geass"
    namespace     TEXT NOT NULL,           -- "geass"
    deployment    TEXT NOT NULL,           -- "geass-auth"
    container     TEXT DEFAULT "",
    image_prefix  TEXT NOT NULL,           -- "bukahou/geass-auth"
    source_path   TEXT DEFAULT "",         -- "geass_auth/"（monorepo 子目录）
    confirmed     BOOLEAN DEFAULT FALSE,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id, namespace, deployment)
);
```

#### 自动推断逻辑

```go
// github/automatch.go
// 匹配规则: image 名称中的服务名 ↔ 仓库顶层目录名
//   bukahou/geass-auth → normalize → "geass-auth"
//   仓库目录 geass_auth → normalize → "geass-auth"
//   匹配！
func AutoMatch(
    repoDirs []string,
    deployments []DeploymentInfo,
) []RepoDeployMapping
```

#### Settings 页面 (/settings/github)

**状态 1：未连接**

```
┌─────────────────────────────────────────────────┐
│  GitHub 集成                                     │
│                                                 │
│  状态: 未连接                                    │
│                                                 │
│  [连接 GitHub]  ← 跳转 GitHub OAuth 授权         │
└─────────────────────────────────────────────────┘
```

**状态 2：已连接，自动匹配映射**

```
┌──────────────────────────────────────────────────────────────┐
│  GitHub 集成                                                  │
│                                                              │
│  状态: 已连接                                                 │
│  账号: wuxiafeng                              [断开连接]      │
│                                                              │
│  ─────────────────────────────────────────────────────────── │
│                                                              │
│  仓库映射                                                     │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  wuxiafeng/Geass                                       │  │
│  │                                                        │  │
│  │  集群: [zgmf-x10a ▼]                                   │  │
│  │                                                        │  │
│  │  Deployment       Namespace       源码目录              │  │
│  │  ─────────────────────────────────────────────────────  │  │
│  │  geass-auth       [geass     ▼]   [geass_auth/     ▼]  │  │
│  │  geass-gateway    [geass     ▼]   [geass_gateway/  ▼]  │  │
│  │  geass-media      [geass     ▼]   [geass_media/    ▼]  │  │
│  │  geass-user       [geass     ▼]   [geass_user/     ▼]  │  │
│  │  geass-favorites  [geass     ▼]   [geass_favorites/▼]  │  │
│  │  geass-history    [geass     ▼]   [geass_history/  ▼]  │  │
│  │  geass-web        [geass-web ▼]   [geass_web/      ▼]  │  │
│  │                                                        │  │
│  │  自动匹配 7/7              [确认映射]  [+ 手动添加]      │  │
│  └────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

**交互规则：**

- 自动匹配结果作为下拉菜单的默认选中值
- 匹配正确 → 用户直接点 [确认映射]
- 匹配不对 → 用户通过下拉菜单手动修改后确认
- 确认后 `confirmed = true`，不再自动覆盖
- 新增未映射 Deployment 时页面提示配置

### 3.7 Code Intel 数据流

deploy_history 是 CD 模块和 Code Intel 模块的衔接点。
CD 写入 deploy_history（含 image tag），Code Intel 读取 deploy_history 并用 GitHub API 补充关联信息。

```
CD 模块写入 deploy_history:
  new_image = "bukahou/geass-auth:20260310-def5678"
  commit_sha = "def5678"  （从 tag 提取）
  pr_number = NULL
  pr_title = NULL

Code Intel 异步补充:
  GitHub API → GetPRByCommit("wuxiafeng/Geass", "def5678")
  → PR#42 "重构路由匹配逻辑"
  → 更新 deploy_history: pr_number=42, pr_title="重构路由匹配逻辑"
```

### 3.8 AI Tool 扩展

```json
{
  "name": "get_deploy_history",
  "description": "查询指定服务的部署历史（镜像变更记录 + 关联的 commit/PR）",
  "parameters": {
    "namespace": { "type": "string" },
    "deployment": { "type": "string" },
    "limit": { "type": "integer", "default": 5 }
  }
}

{
  "name": "rollback_deployment",
  "description": "将 Deployment 回滚到指定的历史镜像版本（需用户确认）",
  "parameters": {
    "namespace": { "type": "string" },
    "deployment": { "type": "string" },
    "target_image": { "type": "string" }
  }
}

{
  "name": "github_read_file",
  "description": "读取关联仓库中的文件内容",
  "parameters": {
    "repo": { "type": "string" },
    "path": { "type": "string" },
    "branch": { "type": "string", "default": "main" }
  }
}

{
  "name": "github_search_code",
  "description": "在关联仓库中搜索代码",
  "parameters": {
    "repo": { "type": "string" },
    "query": { "type": "string" }
  }
}

{
  "name": "github_recent_commits",
  "description": "查看仓库最近的 commits",
  "parameters": {
    "repo": { "type": "string" },
    "path": { "type": "string", "description": "限定路径（可选，monorepo 子目录）" },
    "limit": { "type": "integer", "default": 10 }
  }
}
```

**安全约束**：`rollback_deployment` 是写操作，AI 只能提出建议，必须经用户确认后才执行。

---

## 实施阶段

三个关注点可独立实施，无强依赖关系。

### CD 模块

| 阶段 | 内容 |
|------|------|
| **CD-1** | Deployer 模块（gitops.go + parser.go + 调谐循环） |
| **CD-2** | deploy_config / deploy_history 表 + Service/Gateway API |
| **CD-3** | Agent 新增 apply_kustomize + set_image handler |
| **CD-4** | 前端 /admin/deploy 页面（配置 + 状态 + 历史 + 回滚） |
| **CD-5** | 回滚功能（Deployer.Rollback + 前端回滚按钮） |

### Code Intel 模块

| 阶段 | 内容 |
|------|------|
| **GH-1** | GitHub App 注册 + OAuth 登录 + 认证模块扩展 |
| **GH-2** | GitHub API Client（commits, PRs, 代码读取） |
| **GH-3** | 仓库映射配置（Settings 页面 + 自动匹配） |
| **GH-4** | deploy_history PR 关联（异步补充 commit/PR 信息） |
| **GH-5** | AI Tool 扩展（github_read_file, github_search_code 等） |

### 交叉依赖

```
CD-2 (deploy_history 表) ← GH-4 (PR 关联补充)
CD-5 (回滚) ← GH-5 (AI rollback_deployment tool)
```

其他阶段互不依赖，可并行。

---

## 文件变更清单

### CD 模块

**新增：**

| 文件 | 内容 |
|------|------|
| `deployer/interfaces.go` | Deployer 接口 |
| `deployer/service.go` | 调谐循环、差异对比、触发部署 |
| `deployer/gitops.go` | Git 操作（clone/pull/commit/push） |
| `deployer/parser.go` | 解析 kustomization.yaml |
| `deployer/types.go` | DeployConfig, DeployStatus, DeployRecord |
| `database/sqlite/deploy_config.go` | 部署配置 CRUD |
| `database/sqlite/deploy_history.go` | 部署历史 CRUD |
| `gateway/handler/admin/deploy.go` | 部署管理 API（配置 + 状态 + 历史） |
| `atlhyper_web/src/app/admin/deploy/` | 部署管理页面（配置 + 状态 + 历史 + 回滚） |
| `atlhyper_web/src/api/deploy.ts` | 部署 API 调用 |

**修改：**

| 文件 | 变更 |
|------|------|
| `database/sqlite/migrations.go` | 新增 deploy_config + deploy_history 表 |
| `master.go` | 初始化 Deployer 模块 |
| `gateway/routes.go` | 新增部署相关路由 |
| `atlhyper_agent_v2/service/command/` | 新增 apply_kustomize + set_image handler |

### Code Intel 模块

**新增：**

| 文件 | 内容 |
|------|------|
| `auth/oauth/interfaces.go` | OAuthProvider 接口 |
| `auth/oauth/github.go` | GitHub OAuth 实现 |
| `github/interfaces.go` | GitHubClient 接口 |
| `github/app.go` | GitHub App 身份管理（JWT + Installation Token） |
| `github/api.go` | GitHub REST API 调用 |
| `github/automatch.go` | 仓库目录 ↔ Deployment 自动匹配 |
| `github/types.go` | 数据类型 |
| `database/sqlite/repo_mapping.go` | 仓库映射 CRUD |
| `database/sqlite/github_install.go` | GitHub 安装记录 CRUD |
| `gateway/handler/settings/github.go` | GitHub 设置 API |
| `atlhyper_web/src/app/settings/github/` | GitHub 设置页面 |
| `atlhyper_web/src/api/github.ts` | GitHub API 调用 |

**修改：**

| 文件 | 变更 |
|------|------|
| `database/sqlite/migrations.go` | 新增 github_installations + repo_deploy_mapping 表 |
| `master.go` | 注册 GitHub 模块 + OAuth 路由 |
| `gateway/routes.go` | 新增 GitHub 相关路由 |
| `ai/prompts/tools.go` | 新增 AI Tool 定义 |
| `auth/` | 扩展支持 OAuth Provider |

---

## 约束与风险

| 项目 | 说明 |
|------|------|
| **认证统一** | CD 和 Code Intel 共用 GitHub App Installation Token，无需单独配置 SSH Key |
| **Token 安全** | GitHub App 私钥 / Client Secret 加密存储（common/crypto） |
| **API 限流** | GitHub API 5000 req/h，AI Tool 调用需缓存 |
| **代码量控制** | github_read_file 单次不超过 500 行，超长文件截断 |
| **回滚安全** | rollback_deployment 是写操作，必须经用户确认 |
| **调谐冲突** | 同一 Deployment 短时间内多次变更，需要防抖（debounce） |
| **Git 操作** | Master 需要 git 二进制，Docker 镜像需包含 git |
| **Monorepo** | source_path 字段支持 monorepo 子目录关联 |
| **私有仓库** | GitHub App 安装时授权，自动获得访问权 |

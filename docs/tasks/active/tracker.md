# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## GitHub 集成 + CD + 代码智能 — 🔄 进行中

> 设计文档: [aiops-github-integration-design.md](../../design/active/aiops-github-integration-design.md)

### Phase 1: 基础设施 — ✅ 完成

- GH-1: GitHub App 认证模块（JWT + Installation Token + OAuth） ✅
- GH-2: GitHub API Client（repos, dirs, kustomize scan） ✅
- CD-2: deploy_config / deploy_history / repo_config / github_installations 表 ✅
- CD-2: Gateway Handler（GitHub 连接 + 仓库管理 + 部署配置 + 部署历史） ✅
- CD-4: 前端 API + Datasource 代理层（github.ts + deploy.ts） ✅

### Phase 2: 仓库映射配置 — 待办

- GH-3: repo_mapping / repo_namespaces 表 + Namespace/Mapping CRUD
- GH-3: MappingCard 后端支持

### Phase 3: Deployer 模块 — 待办

- CD-1: deployer/ 模块核心（轮询器 + kustomize build）

### Phase 4: 同步状态 + 回滚 — 待办

- CD-5: 回滚功能（Deployer.Rollback + 前端回滚按钮）

### Phase 5: Agent 端 — 待办

- CD-3: Agent 新增 apply_manifests handler（Dynamic Client + SSA）

### Phase 6: AI Tool 扩展 — 待办

- GH-4: deploy_history PR 关联
- GH-5: AI Tool（github_read_file, github_search_code, get_deploy_history, rollback_deployment）

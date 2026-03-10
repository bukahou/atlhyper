# 任务追踪

> **本文件是任务状态的唯一权威源。**
> 只保留「待办」和「进行中」的任务。完成后归档到 `docs/tasks/archive/`。
>
> 状态标记：`✅` 完成 / `🔄` 进行中 / 无标记 = 待办

---

## GitHub 集成 + CD + 代码智能 — 待办

> 设计文档: [aiops-github-integration-design.md](../../design/active/aiops-github-integration-design.md)

### CD 模块（Master Deployer）

- CD-1: Deployer 模块核心（gitops.go + parser.go + 调谐循环）
- CD-2: deploy_config / deploy_history 表 + Service/Gateway API
- CD-3: Agent 新增 apply_kustomize + set_image handler
- CD-4: 前端 /settings/deploy + /admin/deploy 页面
- CD-5: 回滚功能（Deployer.Rollback + 前端回滚按钮）

### Code Intel 模块（GitHub 集成）

- GH-1: GitHub App 注册 + OAuth 登录 + 认证模块扩展
- GH-2: GitHub API Client（commits, PRs, 代码读取）
- GH-3: 仓库映射配置（Settings 页面 + 自动匹配）
- GH-4: deploy_history PR 关联（异步补充 commit/PR 信息）
- GH-5: AI Tool 扩展（github_read_file, github_search_code 等）

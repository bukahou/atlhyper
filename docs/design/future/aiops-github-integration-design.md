# AIOps GitHub 集成设计

> 状态: future | 创建: 2026-03-08

## 背景

当前 AIOps AI 分析（background/analysis 角色）只能访问运行时数据（K8s 状态、Pod 日志、指标），缺乏对应用代码的理解。

**实际案例**：Geass Gateway 遭受 WordPress 漏洞扫描，`GlobalExceptionHandler` 对 `NoResourceFoundException` 返回 500 而非 404。AI 分析正确定位到日志中的 500 错误，但错误地建议"检查 wp-content 静态资源是否存在"，因为它不知道：
- Geass 是 Spring Boot 项目，不是 WordPress
- 这些请求是恶意扫描，不是合法业务请求
- 真正的修复点是 `GlobalExceptionHandler` 的异常处理逻辑

**目标**：让 AI 在分析事件时能访问关联服务的源代码仓库，从而给出代码级别的根因分析和修复建议。

## 能力预期

| 层级 | 当前（无 GitHub） | 目标（有 GitHub） |
|------|-------------------|-------------------|
| **现象识别** | "日志中出现 500 错误" | 同左 |
| **根因定位** | "静态资源缺失" (错误) | "GlobalExceptionHandler 未处理 NoResourceFoundException" |
| **修复建议** | "检查文件是否存在" | "在 XxxHandler.java:42 添加 @ExceptionHandler(NoResourceFoundException.class) 返回 404" |
| **上下文判断** | 无法区分合法/恶意请求 | "Geass 是 Spring Boot 应用，wp-content 路径是 WordPress 扫描特征" |

## 设计方案

### 1. 数据模型

Service 与 GitHub 仓库的关联（新增配置表或扩展现有 Service 元数据）：

```sql
CREATE TABLE IF NOT EXISTS service_repo_mapping (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cluster_id TEXT NOT NULL,
    service_name TEXT NOT NULL,       -- K8s Service 名称
    namespace TEXT NOT NULL,
    repo_url TEXT NOT NULL,           -- GitHub 仓库 URL
    default_branch TEXT DEFAULT 'main',
    source_path TEXT DEFAULT '',      -- 仓库内服务根目录（monorepo 场景）
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(cluster_id, namespace, service_name)
);
```

### 2. AIOps Tool 扩展

新增 GitHub 相关 Tool，供 analysis 角色在多轮 Tool Calling 中使用：

| Tool | 功能 | 参数 |
|------|------|------|
| `github_search_code` | 在仓库中搜索代码 | repo, query, path(可选) |
| `github_read_file` | 读取文件内容 | repo, path, branch(可选) |
| `github_list_files` | 列出目录结构 | repo, path, branch(可选) |
| `github_recent_commits` | 查看最近提交 | repo, branch, since(可选) |
| `github_search_issues` | 搜索相关 Issue/PR | repo, query |

### 3. 调用流程

```
事件触发 → AI 获取事件上下文
         → 查询 service_repo_mapping 获取关联仓库
         → Tool Calling 循环中按需调用 GitHub Tool
         → 结合运行时数据 + 代码上下文给出分析
```

### 4. 认证方案

- GitHub Personal Access Token (PAT) 存储在 AI Provider 配置或独立的集成配置中
- Token 需要 `repo:read` 权限（只读）
- 使用 `common/crypto` AES-256-GCM 加密存储

### 5. 实现层级

| 阶段 | 内容 | 优先级 |
|------|------|--------|
| **Phase 1** | service_repo_mapping 配置 + github_read_file / github_search_code Tool | 高 |
| **Phase 2** | github_recent_commits + github_search_issues Tool | 中 |
| **Phase 3** | 前端设置页面（Service ↔ Repo 关联管理） | 中 |
| **Phase 4** | 自动推断（从 K8s label/annotation 自动关联仓库） | 低 |

## 约束与风险

| 项目 | 说明 |
|------|------|
| **Token 安全** | PAT 不能明文存储/日志输出，遵循安全规范 |
| **API 限流** | GitHub API 有速率限制（5000 req/h），Tool 调用需要考虑缓存 |
| **代码量控制** | 单次读取文件不宜过大，需截断或摘要，避免超出 LLM 上下文窗口 |
| **私有仓库** | 需要适当权限的 PAT，公开仓库可匿名访问 |
| **成本** | 代码上下文会增加 Token 消耗，analysis 角色预算需相应调整 |

## 文件变更清单（预估）

### 新增
- `atlhyper_master_v2/database/types.go` — ServiceRepoMapping 结构体
- `atlhyper_master_v2/database/interfaces.go` — ServiceRepoMappingRepository 接口
- `atlhyper_master_v2/database/sqlite/service_repo_mapping.go` — SQLite 实现
- `atlhyper_master_v2/aiops/ai/tools/github.go` — GitHub Tool 实现
- `atlhyper_master_v2/gateway/handler/admin/service_repo.go` — 管理 API

### 修改
- `atlhyper_master_v2/database/sqlite/migrations.go` — 新增表
- `atlhyper_master_v2/master.go` — 注册新 Tool
- `atlhyper_master_v2/aiops/ai/context_builder.go` — 构建上下文时附加仓库信息

# 权限模型 设计文档

> 状态：第 1 期代码完成（分支 `feat/permission-model`，本地 commit），待推 dev
> 后续：OIDC 接入（akasha 为首个身份源）建立在本设计之上，另起文档

## 背景 + 目标

现状有三个结构性问题：

1. **「公开 / 需登录」两套路由表**：公开组内混有内容级数据（日志正文、链路属性、指令结果），
   「什么算敏感」散落在路由注册里，没有一处能看清全局。
2. **已登录 ≠ 有权限的语义缺失**：最低角色 Viewer(1) 比访客多出 AI 对话等能力。
   接入外部身份源（任何 Google 账号都能自动注册）后，新用户若落在 Viewer，即对全网开放 LLM 额度。
3. **角色固化在 JWT 里**：提升 / 降级 / 禁用要等 token 过期（最长 24h）才生效。

**目标**：

- 访客（未登录）与新注册成员**权限完全相同**（0 级），只有管理员能提升。
- 每个接口声明所需**权限**，权限表集中定义一处。
- 后端是唯一权限源：按权限裁剪响应字段，前端只按后端下发的权限列表展示。
- 角色变更即时生效。

## 用户裁决（2026-10-12）

| # | 决定 | 裁决 |
|---|---|---|
| 1 | 权限检查形态 | **B**：固定角色 0–3 → 权限映射（代码内一张表），接口声明权限 |
| 2 | 分档边界 | 按本文「权限表」；AI 对话拆为独立权限 `ai.chat`，暂给 1 级，后续重新设计 |
| 3 | 无权限时的展示 | **C 混合**：真实数据页内无权限部分 → 锁定占位；纯管理功能页 → 整页演示数据 + 横幅；真实与演示数据不同屏 |
| 4 | 角色生效时机 | **B**：中间件每次请求按主键查库取当前角色与状态（用户明确允许） |
| 5 | 指令历史输出 | 先行单独修复（分支 `fix/command-history-output-leak`） |

## 核心架构

```
请求 ─> Authenticate（可选认证）
          ├─ 无 token        → Principal{level 0, 访客}
          ├─ token 无效/过期 → 401（前端重新登录）
          └─ token 有效      → 按 user_id 查库：不存在或已禁用 → 401；否则 Principal{level = 库中当前角色}
      ─> RequirePermission(p)：level 的权限集不含 p → 访客 401 / 已登录 403
      ─> Handler ─> Service ─> Convert ─> Redact(principal)：按权限裁剪字段 ─> 响应
```

| 主体 | level | 来源 |
|---|---|---|
| 访客（未登录） | 0 | 无 token |
| 成员 | 0 | 外部身份源首次登录自动建号（OIDC 文档） |
| Viewer | 1 | 管理员提升 |
| Operator | 2 | 管理员提升 |
| Admin | 3 | 管理员提升 / 启动种子 |

**无需任何主体的路由**（不进权限表）：`/health`、`/api/v2/user/login`、`/api/github/callback`（第三方回跳）、
以及后续 OIDC 的 start / callback。

## 功能一：主体解析与即时生效

**用户故事**：管理员提升 / 降级 / 禁用一个用户，对方的下一次请求立刻按新权限执行。

- 新中间件 `Authenticate` 取代「publicMux 优先 + `AuthRequired`」的双表分派：所有请求统一经过它。
- JWT 只承载身份（`user_id`），**角色不再从 claims 读取**；claims 中的 `role` 保留到下次登录改造，仅供前端显示。
- 每次请求 `SELECT role, status FROM users WHERE id = ?`（主键，微秒级）。
- 不改登录流程本身（005 冻结范围）。

**测试**：无 token → level 0；有效 token → 库中角色（库中降级后立即生效）；禁用用户 → 401；用户已删 → 401；坏签名 → 401。

## 功能二：权限表与路由声明

**用户故事**：维护者看一张表就知道每一级能做什么；新增接口必须声明权限才能注册。

`gateway/middleware/permission.go`（新增）：权限常量 + `levelPermissions` 累进表（高级继承低级）。

### 权限表

| level | 权限 | 覆盖接口 |
|---|---|---|
| **0** | `cluster.read` | overview、clusters、全部资源列表 / 详情（env 值裁剪，见功能三）、configmaps 列表、namespaces |
| | `events.read` | events、events/by-resource |
| | `metrics.read` | node-metrics、observe/freshness、observe/metrics/*（含 hardware） |
| | `slo.read` | slo/*、observe/slo/* |
| | `apm.summary.read` | observe/traces/services[/]、stats、topology、operations |
| | `logs.stats.read` | observe/logs/summary、histogram（只有计数） |
| | `aiops.read` | aiops/graph[/trace]、baseline、risk/*、incidents* |
| **1** | `workload.env.read` | 资源详情中的 env 字面值 |
| | `logs.read` | observe/logs/query |
| | `traces.read` | observe/traces（列表）、traces/{id}、traces/correlations（含属性值） |
| | `commands.read` | commands/history、commands/{id} |
| | `deploy.history.read` | api/deploy/history |
| | `github.status.read` | api/github/connection |
| | `ai.chat` | ai/conversations*、ai/chat（删除会话另需 level 2，沿用现状） |
| **2** | `secret.read` / `configmap.data.read` | secrets 列表、configmaps 详情、ops/secrets/data、ops/configmaps/data |
| | `pod.logs.read` | ops/pods/logs |
| | `cluster.operate` | commands（创建）、ops/pods/restart、ops/deployments/*、ops/nodes/* |
| | `audit.read` | audit/logs |
| | `notify.manage` | notify/channels（读 + 写） |
| | `ai.analyze` | aiops/ai/*（summarize / recommend / reports / analyze） |
| | `ai.settings.read` | settings/ai、ai/providers、ai/settings、ai/roles、ai/budgets、ai/reports（读） |
| **3** | `user.manage` | user/list、register、update-role、update-status、delete |
| | `ai.settings.write` | settings/ai/、ai/providers/、ai/settings/、ai/budgets/（写） |
| | `github.manage` | github/connect、disconnect、repos* |
| | `deploy.manage` | deploy/config、kustomize-paths、test-connection、status、sync、rollback |

**分档判据**：0 级只含**集群状态与形状**；内容级数据（可能带业务数据 / 个人信息）≥ 1；凭据与写操作 ≥ 2；系统配置 = 3。

路由注册改为 `r.route(pattern, perm, handler)` / `r.routeAudited(...)`；删除 `public / operator / admin` 系列辅助函数。

**测试**：表驱动遍历权限表——每个 level 恰好拥有声明的权限；level 0 不含任何 ≥1 的权限；
路由层：level 0 访问 `logs.read` 接口得 401，Viewer 访问 `cluster.operate` 得 403。

## 功能三：响应字段裁剪

**用户故事**：访客能看到 Pod 有哪些环境变量、引用了哪个 Secret，但看不到字面值。

- 资源详情（Pod / Deployment / StatefulSet / DaemonSet / Job / CronJob）的 `envs[].value`：
  无 `workload.env.read` 时置空并标记 `valueHidden: true`；`valueFrom`（如 `secret:xxx`）保留。
- **实施调整**：各资源详情的容器结构不统一（Pod 强类型，工作负载 template 为 `interface{}`），
  改为网关层装饰器 `middleware.RedactEnvValues` 在 **JSON 层**处理任意层级的 `envs[]`，
  套在全部 `cluster.read` 路由上（新增资源类型自动覆盖）；解析失败返回 500（失败即关）。
  因此 Go 模型**不新增** `ValueHidden` 字段，`valueHidden` 只出现在响应 JSON 与前端类型中。

| 前端 TS（`ContainerEnv` / `EnvVarSpec`） | 响应 JSON |
|---|---|
| `value?: string` | 有权限时原样；无权限时删除 |
| `valueHidden?: boolean` [新增] | 无权限且原有字面值时为 `true` |

**测试**：level 0 详情无字面值且 `valueHidden=true`、`valueFrom` 保留；level 1 原样。

## 功能四：`/auth/me` 与前端统一判断

**API**：`GET /api/v2/auth/me`（无需主体）

```json
{
  "message": "获取成功",
  "data": {
    "user": null,
    "level": 0,
    "permissions": ["cluster.read", "events.read", "metrics.read", "slo.read", "apm.summary.read", "logs.stats.read", "aiops.read"]
  }
}
```

已登录时 `user = {id, username, displayName, email, role}`。

- 前端 `authStore` 增 `permissions`；应用启动、登录成功、收到 401/403 时刷新。
- 新 `hooks/useCan.ts`：组件内 `can("logs.read")`；datasource 层用 `canNow(...)`，无权限时**不发请求**。
  「能看 / 能做」类判断全部改走它；`isAuthenticated` 只保留在「显示登录按钮还是用户菜单」这类身份展示上。
- `useRequireAuth(action, perm = "cluster.operate")`：有权限执行；访客弹登录框；已登录不足提示联系管理员。
- 侧栏菜单、操作按钮、页面演示态均由权限驱动。

## 功能五：无权限时的展示（裁决 3 = C）

| 场景 | 展示 |
|---|---|
| 真实数据页内无权限的部分（env 值、日志正文、链路列表 / 详情、指令历史） | 锁定占位：「需要 Viewer 权限 —— 登录或联系管理员」 |
| 纯管理功能页（AI 设置、审计、通知、用户管理、AI 对话） | 整页演示数据 + 醒目「演示数据」横幅 |
| 规则 | ⛔ 真实数据与演示数据不出现在同一视图 |

`components/common/PermissionLock.tsx`（新增）统一锁定占位；`components/common/DemoBanner.tsx`（新增）统一演示横幅；文案走 i18n。
锁定占位**不显示所需等级**：那需要前端复制一份权限表，违反「后端唯一权限源」。

已覆盖：日志列表、链路详情 / 慢链路 / 错误链路 / 相关性 / 状态码筛选链路、指令历史页、命名空间 Secret 标签页、用户管理（演示）。

## 实施阶段

| 期 | 内容 | 验收（用户可见） |
|---|---|---|
| 1 | 功能一 + 二 + 三 + 四（后端）+ 前端 `useCan` 与功能五 | 访客打开日志页见锁定占位；Pod 详情 env 值打码；提升用户后刷新即生效 |
| 2 | 成员 0 级 + OIDC 接入 | 另起文档 |

后端与前端放同一期：只改后端会让前端对访客出现大量 401 弹窗，是半成品。

## 文件变更清单

```
atlhyper_master_v2/
├── gateway/middleware/
│   ├── auth.go             [修改] Authenticate（可选认证 + 查库）取代 AuthRequired / RequireMinRole
│   ├── principal.go        [✅ 新增] Authenticate / RequirePermission / Principal
│   ├── permission.go       [✅ 新增] 权限常量 + level → 权限表
│   ├── redact_env.go       [✅ 新增] env 字面值裁剪装饰器
│   └── *_test.go           [✅ 新增]
├── gateway/routes.go       [修改] 全部路由改为声明权限；删除 public/operator/admin 辅助函数
├── gateway/handler/
│   ├── auth_me.go          [新增] GET /api/v2/auth/me
│   （env 字面值裁剪改在 middleware/redact_env.go，见功能三）
├── service/ + database/    [修改] 按 id 取 role/status 的只读查询
└── database/sqlite/migrations.go [修改] users.role 默认值 3 → 0
atlhyper_web/src/
├── store/authStore.ts      [修改] permissions
├── api/auth.ts             [修改] getMe()
├── hooks/useCan.ts         [新增]
├── components/common/PermissionLock.tsx [新增]
├── i18n/                   [修改] 锁定 / 演示文案
├── components/common/DemoBanner.tsx [新增]
└── （各页面判断点）        [修改] 改走 useCan
CLAUDE.md §4.3              [✅ 修改] 规范重写
```

## 附录

- **后续项（第 1 期未做）**：角色说明页（`app/admin/roles`）的权限矩阵是手写的，与新权限表不一致且缺「成员」档。
  建议改为由后端下发权限表生成，避免第二份手写真相。

- `users.role` 列默认值现为 3（Admin）：任何漏传 role 的插入都会得到管理员。改为 0（按 §1.9 直接改建表语句）。
- 已有用户角色值 1–3 不变；0 为新增档。

# OIDC 登录接入 设计文档

> 状态：设计已裁定（用户 2026-10-12：附录 A 三项全按推荐），A 期实施中
> 前置：[permission-model-design.md](permission-model-design.md)（第 1 期已上 dev）—— 本文是其第 2 期
> 首个身份源：akasha（自研 OIDC IdP）。代码按**通用 OIDC** 写，akasha 只是第一个被配置进来的 issuer

## 背景 + 目标

atlhyper 目前只有本地账号（管理员手动建号）。目标：

- 访客可用外部身份源（OIDC）**自助登录**，首次登录自动建号。
- 自动建出的账号一律为 **成员（level 0）**，与访客权限完全相同；只有管理员能提升。
- 不改本地密码登录流程（005 冻结范围），且不为其引入新的风险面。

**用户已裁定（2026-10-12）**：

| # | 决定 | 裁决 |
|---|---|---|
| 1 | 写死 akasha 还是通用 OIDC | 先接入 OIDC，插件式多源以后再定 → 按**通用单 issuer** 实现 |
| 2 | 普通用户的权限 | 新档 0 = 成员 = 访客（第 1 期已实现） |
| 3 | 自动建号准入 | 全放行（0 级无额外能力）；准入接口保留以便日后收紧 |
| 4 | 登录后 token 交付 | 沿用现有 JWT，经 URL 片段交给前端（cookie 化随 005） |
| 5 | 客户端库 | akasha 仓的 `pkg/oidcrp`（内部为 `coreos/go-oidc` + `x/oauth2`，与 provider 无关） |

## 核心架构

```
登录框「使用 <显示名> 登录」
  → GET  /api/v2/auth/oidc/start?next=/当前页      oidcrp.Start：种状态 cookie（state / nonce / PKCE），302 到 issuer
  → issuer 登录（akasha → Google）
  → GET  /api/v2/auth/oidc/callback                 oidcrp.Callback：验 state → 换票 → 验 id_token
       └ OnAuthenticated(identity)
           ├ 查 user_identities(issuer, subject)
           │   ├ 有 → 取 users 行；已禁用 → 失败
           │   └ 无 → 准入策略 → 事务内建 users（role=0, password_hash=""）+ user_identities
           ├ 同步展示资料（display_name / email）+ 记 last_login
           └ 签发现有格式 JWT
  → 302 /auth/callback#token=<jwt>&next=/当前页      片段不会发往任何服务器、不进访问日志
  → 前端回调页：存 token → refreshMe() → 跳回 next
```

- 浏览器侧只经过 Web 的 `/api/v2` 代理（Next.js middleware rewrite，302 与 Set-Cookie 原样透传），
  对 issuer 而言回调地址是 **Web 域名**下的 `/api/v2/auth/oidc/callback`。
- Master 是**唯一的 OIDC 客户端**（confidential client），secret 不经过浏览器。
- 未配置 issuer 时整个功能不注册路由、登录框不显示按钮，对现有部署零影响。

## 功能一：OIDC 登录与自动建号

**用户故事**：任何人用外部身份登录 atlhyper，第一次登录自动得到一个成员账号；之后再登录回到同一个账号。

### API

| Method + Path | 主体 | 说明 |
|---|---|---|
| `GET /api/v2/auth/providers` | 无需 | 登录框据此决定是否显示按钮 |
| `GET /api/v2/auth/oidc/start?next=` | 无需 | 302 到 issuer |
| `GET /api/v2/auth/oidc/callback` | 无需（审计） | issuer 回跳；302 到 `/auth/callback#...` |

`GET /api/v2/auth/providers` 响应：

```json
{ "message": "获取成功", "data": [ { "id": "oidc", "displayName": "Akasha" } ] }
```

未配置时 `data: []`。

回调成功：`302 /auth/callback#token=<jwt>&next=<已校验的应用内路径>`
回调失败：`302 /auth/callback#error=<原因码>`，原因码白名单：

| 原因码 | 来源 |
|---|---|
| `cancelled` / `state` / `upstream` | oidcrp 的 Reason* |
| `disabled` | 账号已被管理员禁用 |
| `rejected` | 准入策略拒绝 |
| `application` | 其余内部错误（不向用户泄漏细节） |

### 数据对照

**新表**（`CREATE TABLE IF NOT EXISTS`，对已有数据库是纯增量）：

```sql
CREATE TABLE IF NOT EXISTS user_identities (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL,
    issuer     TEXT    NOT NULL,
    subject    TEXT    NOT NULL,
    created_at TEXT    NOT NULL,
    UNIQUE (issuer, subject)
);
CREATE INDEX IF NOT EXISTS idx_user_identities_user ON user_identities(user_id);
```

| DB 列 | Go（`database.UserIdentity`） | 说明 |
|---|---|---|
| `user_id` | `UserID int64` | → `users.id` |
| `issuer` | `Issuer string` | 取自配置的 issuer（oidcrp 的 Identity 不带 issuer） |
| `subject` | `Subject string` | id_token `sub`；akasha 为 pairwise，**只对本 client 有效** |
| `created_at` | `CreatedAt time.Time` | |

**为什么不给 `users` 加列**：dev / prod 的 SQLite 里有真实数据，§1.9「改建表语句 + 删库重建」对已有库不生效
（`CREATE TABLE IF NOT EXISTS` 不会补列）。身份关联放新表，零迁移。

**建出的 users 行**：`role=0`、`status=1`、`password_hash=""`、
`username` 由 `preferred_username` → 邮箱本地部分 → `user` 依次取，规整为 `[a-z0-9._-]`，冲突加 `-2`、`-3`…；
`display_name` = `name`，`email` = 上游邮箱（**仅展示**，不参与任何匹配或找回）。

### 后端实现

```
atlhyper_master_v2/
├── auth/oidc/                     [新增] OIDC 登录模块（Facade：Handlers() 返回三个 handler）
│   ├── interfaces.go              FederatedAccounts（身份 → 本地用户）/ AdmissionPolicy
│   ├── module.go                  懒初始化 oidcrp.Flow（issuer 不可达不阻塞启动，首次使用时重试）
│   ├── handler.go                 providers / start / callback；OnAuthenticated / OnError
│   ├── accounts.go                查或建（事务；唯一键冲突时重查 —— 并发首登）
│   ├── username.go                用户名生成与去重
│   └── *_test.go
├── database/
│   ├── interfaces.go              [修改] UserIdentityRepository；UserRepository 增 GetByUsernameForPasswordLogin
│   └── sqlite/
│       ├── migrations.go          [修改] user_identities 建表
│       └── user_identity.go       [新增]
├── service/                       [修改] 查或建、本地登录查询走 service 层
├── gateway/routes.go              [修改] 三条 open 路由（callback 审计）
└── config/                        [修改] OIDCConfig
```

**本地密码登录不受影响的做法**：登录处理器改用 `GetByUsernameForPasswordLogin`，其 SQL 排除
`user_identities` 中存在的用户。OIDC 账号对密码登录而言与「用户不存在」走**同一条既有路径**，
不新增任何分支或时序差异；登录处理器本身的逻辑一行不改（005 冻结）。

**测试**：首登建号（role=0）/ 再登回到同一账号 / 禁用账号拒绝 / 并发首登只建一个 /
用户名冲突加后缀 / 用户名规整 / 密码登录查不到 OIDC 账号 / 未配置时不注册路由。
oidcrp 本身有自己的测试，这里用假的 `OnAuthenticated` 入参驱动，不起真 issuer。

### 前端实现

| 文件 | 变更 |
|---|---|
| `api/auth.ts` | [新增] `getAuthProviders()` |
| `components/auth/LoginDialog.tsx` | [修改] 有 provider 时在表单下方显示「使用 <displayName> 登录」，跳 `start?next=当前路径` |
| `app/auth/callback/page.tsx` | [新增] 读片段：成功 → 存 token → `refreshMe()` → `history.replaceState` 清片段 → 跳 `next`；失败 → 按原因码显示 i18n 文案 |
| `i18n` | [修改] 按钮与各原因码文案 |

## 功能二：管理员提升成员

第 1 期已具备：用户管理页可把 0 改为 1–3（`update-role` 允许 0）；角色每次请求查库，**刷新即生效**。
本期只补：用户列表显示「外部身份」标记（`users/list` 响应增 `federated: bool`），方便管理员区分。

## 配置

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `MASTER_OIDC_ISSUER` | `""` | 为空 = 功能关闭 |
| `MASTER_OIDC_CLIENT_ID` | `""` | issuer 非空时必填 |
| `MASTER_OIDC_CLIENT_SECRET` | `""` | 凭证：无默认值，issuer 非空时必填 |
| `MASTER_OIDC_REDIRECT_URL` | `""` | `https://<web-host>/api/v2/auth/oidc/callback`，须与 issuer 处登记逐字一致 |
| `MASTER_OIDC_COOKIE_SECRET` | `""` | 凭证：状态 cookie 签名，≥ 32 字节随机串，issuer 非空时必填 |
| `MASTER_OIDC_DISPLAY_NAME` | `"SSO"` | 按钮文字 |
| `MASTER_OIDC_COOKIE_SECURE` | `true` | 仅本机 http 调试时设 false |

真实值只进 config 私有仓与 K8s Secret（⛔ 不进本仓）。

## 实施阶段

| 期 | 内容 | 谁做 | 验收 |
|---|---|---|---|
| A | 功能一 + 二 的代码与测试（含 Go 版本升级，见裁定项 1） | Claude，本地 commit | 单元测试；未配置 issuer 时行为与现在完全一致 |
| B | akasha 登记 dev client（`clients.yaml` + bcrypt 哈希 + `validate-clients`）→ akasha SYNC | ⚠️ **用户亲自做或单独批准**：akasha 只有一个实例，同时服务 prod 应用，SYNC 会重启它 | issuer discovery 中可见、登录页显示 atlhyper |
| C | atlhyper dev 的 Secret / 环境变量 → 推 dev | 每步单独批准 | 浏览器：未登录 → akasha 登录 → 回到 atlhyper，身份为 Member、看到的与访客一致；admin 提升后刷新即变 |
| D | prod | ⛔ 用户自行处理 | —— |

## 文件变更清单

```
go.mod / deploy/docker/Dockerfile.{controller,agent}   [修改] Go 版本（裁定项 1）
atlhyper_master_v2/auth/oidc/                           [新增] 见功能一
atlhyper_master_v2/database/…                           [修改/新增] user_identities
atlhyper_master_v2/service/…                            [修改]
atlhyper_master_v2/gateway/routes.go                    [修改]
atlhyper_master_v2/gateway/handler/admin/user.go        [修改] 改用 GetByUsernameForPasswordLogin；list 增 federated
atlhyper_master_v2/config/{types,loader,defaults}.go    [修改] OIDCConfig
atlhyper_web/src/api/auth.ts                            [修改]
atlhyper_web/src/components/auth/LoginDialog.tsx        [修改]
atlhyper_web/src/app/auth/callback/page.tsx             [新增]
atlhyper_web/src/i18n/                                  [修改]
```

## 附录 A：裁定项（用户 2026-10-12：全按推荐 —— 1=A 升 1.26 / 2=A 只清本地 / 3=本期不处理）

### 裁定项 1：Go 版本升级

- **背景**：oidcrp 的 go.mod 要求 Go **1.26.6**；atlhyper 的 go.mod 是 1.25，Docker 构建镜像写死 `GO_VERSION=1.25`。
  引入依赖后 go.mod 会被抬到 1.26.6；构建镜像若仍是 1.25，会在构建时自动下载新工具链（依赖网络，且版本不受控）。
  geass（1.26.6）、melete（1.26.7）已在 1.26。CI 测试按 go.mod 取版本，不受影响。
- **影响**：
  - **A. 升到 1.26**：go.mod 与两个 Dockerfile 一起改。与另两个应用对齐；Agent 也会随之用新版编译（行为不变，但镜像会重建）。
  - **B. 不用 oidcrp，直接用 go-oidc**：不升版本，但 PKCE / state / nonce 这段要第三次手写，违反「复制三次必须抽象」。
- **推荐 A**。

### 裁定项 2：登出是否通知 issuer

- **背景**：之前方案写过「登出时跳 issuer 的 `end_session`」。核实后发现 akasha **没有 IdP 会话**
  （每次 authorize 都重新走上游登录），`end_session` 只做白名单跳转，不清除任何东西。
- **影响**：
  - **A. 只清本地 token**：实现最少；对 akasha 而言效果完全相同。换成有会话的 issuer（如 Keycloak）时，用户在 issuer 那边仍是登录态。
  - **B. 同时跳 `end_session`**：对将来的有会话 issuer 正确；需要多登记一个 `post_logout_redirect_uri`，多一次跳转。
- **推荐 A**，本期不做；换 issuer 时再加。⚠️ 这是对之前方案的改动，请确认。

### 裁定项 3：开放注册的行数增长

- **背景**：全放行意味着任何 Google 账号登录一次就多一行 users。
- **影响**：0 级没有任何额外能力，安全上无影响；代价是用户列表可能出现陌生人。
- **推荐**：本期不处理；用户列表显示「外部身份」标记即可。若将来变多，再加「N 天未登录的 0 级外部账号自动清理」。

## 附录 B：约束与风险

- **client_id 上线后不可改**：akasha 的 sub 按 client_id 派生（pairwise），改 client_id = 所有外部账号失联。
  dev 与 prod 是两个 client，同一个人在两边是两个账号（与 geass / melete 一致）。
- **issuer 不可达**：不阻塞 Master 启动；按钮照常显示，点击时若仍不可达则走 `upstream` 失败页。
- **开放重定向**：`next` 经 oidcrp `SafeNext`（只放行本站绝对路径），回调页跳转前再校验一次。
- **token 在 URL 片段**：片段不发往服务器、不进访问日志；回调页落盘后立即 `replaceState` 清掉。
- **邮箱只作展示**：⛔ 不按邮箱匹配已有账号、⛔ 不用于找回（上游的说法，不是本系统验证过的事实）。

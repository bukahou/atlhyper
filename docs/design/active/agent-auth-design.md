# 集群注册表与 Agent 身份认证

> 状态：执行中。「裁决记录」一节的各项已于 2026-11-14 由用户裁定。

## 背景

Master 设计上管理多个集群，Agent 部署在各集群里主动连接 Master。要支持「Agent 与 Master 不在同一个集群」，
Master 必须能确认每个连接来自哪个集群，并且只把属于它的指令交给它。本设计补上两块能力：

- **集群注册表**：集群是管理员登记的对象，而不是首次上报时自动出现的字符串
- **Agent 身份认证**：每个集群一个令牌，Master 由令牌确定请求属于哪个集群

同时把 Agent 在集群里的 K8s 权限从「集群管理员」收窄到它实际使用的范围。

## 目标

- 每个集群一个令牌；Agent 的每个请求都带令牌，Master 据此确定集群身份
- 快照、心跳、指令、执行结果四条通道都绑定到已认证的集群
- 三档开关，支持先观察后强制的滚动升级
- 管理页可以登记集群、签发与轮换令牌
- Agent 使用专用账号与最小权限；其余组件不挂载集群凭据

## 核心架构

```
Agent ── Authorization: Bearer <令牌> ──► AgentSDK
                                            │ authenticateAgent（中间件）
                                            │   令牌 → 哈希 → 注册表 → 集群身份
                                            ▼
                         snapshot / heartbeat / commands / result
                         一律使用「已认证的集群」，不再采信请求里自报的集群 ID

Web（3 级） ──► /api/v2/admin/clusters ──► service ──► database.Cluster（注册表）
```

### 令牌

- 形如 `atlk_` + 32 字节随机数的 base64url；由 Master 生成，**只在签发时返回一次**
- 库里只存 SHA-256 哈希与前 8 个字符（界面上用于辨认）。令牌是高熵随机数，不需要慢哈希
- 校验用常数时间比较
- 轮换 = 重新签发：旧令牌立即失效

### 认证模式

配置项 `MASTER_AGENT_AUTH_MODE`：

| 模式 | 带有效令牌 | 没带令牌 / 令牌无效 | 令牌所属集群 ≠ 自报集群 |
|------|-----------|--------------------|------------------------|
| `off` | 不检查 | 不检查 | 不检查 |
| `warn`（默认） | 以令牌的集群为准 | 放行，按自报的集群 ID 处理，记 WARN | 拒绝（403） |
| `enforce` | 以令牌的集群为准 | 拒绝（401） | 拒绝（403） |

`warn` 的日志按「集群 + 原因」限频（每分钟一条），避免每 10 秒一份快照刷屏。

### 四条通道的绑定

| 通道 | 集群身份的来源 |
|------|----------------|
| `POST /agent/snapshot` | 已认证的集群；请求头与请求体里自报的 ID 只用于比对 |
| `POST /agent/heartbeat` | 同上 |
| `GET /agent/commands` | 同上；只取该集群队列里的指令 |
| `POST /agent/result` | 同上；指令不属于该集群时拒绝（403），不写入结果 |

`mq.Consumer.AckCommand` 增加集群参数，由队列自己核对指令归属：

```go
// clusterID 非空时，指令必须属于该集群，否则返回 command.ErrNotOwnedByCluster
AckCommand(clusterID, cmdID string, result *command.Result) error
```

### 快照大小上限

`MASTER_AGENTSDK_MAX_SNAPSHOT_MB`（默认 256）：压缩前后都不得超过。超过返回 413。

### Agent 侧

- 新增 `AGENT_MASTER_TOKEN`；非空时每个请求带 `Authorization: Bearer`
- 执行指令前核对 `cmd.ClusterID` 与自身集群一致，不一致拒绝执行并回报失败
- 请求参数里的集群 ID 做 URL 转义

## 功能一：集群注册表与令牌管理

**用户故事**：管理员在「集群管理」页看到所有集群（已登记的、以及正在上报但尚未登记的），
可以登记集群、改显示名、签发或轮换令牌、删除登记。令牌只在签发时显示一次。

**权限**：新增 `clusters.manage`（3 级）。

**API**

```
GET /api/v2/admin/clusters
→ { "message": "获取成功", "data": [
     { "clusterId": "c1", "name": "生产", "description": "",
       "registered": true, "tokenIssued": true, "tokenPrefix": "atlk_Ab3", "tokenIssuedAt": "2026-11-14T10:00:00Z",
       "status": "online", "lastSeen": "2026-11-14T10:05:00Z", "authenticated": true }
   ], "total": 1 }

POST /api/v2/admin/clusters            { "clusterId": "c1", "name": "生产", "description": "" }
→ 201 { "message": "已登记", "data": { ...同上一项... } }      # 集群 ID 已存在 → 409

PUT /api/v2/admin/clusters/{clusterId}  { "name": "生产", "description": "东京" }
→ 200

POST /api/v2/admin/clusters/{clusterId}/token
→ 200 { "message": "已签发", "data": { "token": "atlk_...", "tokenPrefix": "atlk_Ab3", "tokenIssuedAt": "..." } }

DELETE /api/v2/admin/clusters/{clusterId}
→ 200                                   # 删除登记，令牌随之失效；不删除该集群的历史数据
```

- `status`：`online` / `offline` / `never`（登记了但从未上报）
- `authenticated`：该集群最近一次请求是否带了有效令牌
- `registered: false` 的行来自正在上报但未登记的集群（`off` / `warn` 模式下才会出现），可以直接登记收编
- 集群 ID 的格式：1–63 个字符，小写字母、数字、`-`、`.`，首尾必须是字母或数字
- 显示名必填、不超过 64 个字；描述不超过 256 个字
- 登记与修改成功时 `data` 带回该集群最新的一行；签发令牌的响应带 `Cache-Control: no-store`

**数据对照**

| API JSON | Go（`model.ClusterRegistryItem`） | 表 `clusters` |
|---|---|---|
| `clusterId` | `ClusterID` | `cluster_id TEXT PRIMARY KEY` |
| `name` | `Name` | `name TEXT NOT NULL` |
| `description` | `Description` | `description TEXT NOT NULL` |
| `tokenIssued` | `TokenIssued` | 由 `token_hash != ''` 得出 |
| `tokenPrefix` | `TokenPrefix` | `token_prefix TEXT NOT NULL` |
| `tokenIssuedAt` | `TokenIssuedAt` | `token_issued_at TEXT` |
| —（不出库） | — | `token_hash TEXT NOT NULL` |
| `registered` / `status` / `lastSeen` / `authenticated` | 同名 | 不落库，来自 DataHub 的在线状态 |

`clusters` 表原有定义（自增主键 + `cluster_uid` + `environment`）没有被任何代码使用，直接改为上表。

**后端实现**

- `database/`：`ClusterRepository` 改为 `Create / Update / Delete / GetByID / GetByTokenHash / List / SetToken`
- `agentsdk/clustertoken/`：令牌生成、哈希、常数时间比较（纯函数）
- `service/operations/cluster_registry.go`：登记、改名、签发、删除；`service/query/cluster_registry.go`：列表（注册表 + 在线状态 + 认证状态合并）
- `model/cluster_registry.go`（请求 / 响应模型与输入校验）、`model/convert/cluster_registry.go`
- `gateway/handler/admin/cluster_registry.go` + `routes.go` + 权限表

- 审计：写操作全部审计。审计记录新增 `path` 列（请求路径，不含查询串），
  因为改名、签发令牌、删除登记的集群 ID 只出现在路径里

**前端实现**

- `app/admin/clusters/page.tsx` [新增]：表格 + 登记弹窗 + 签发令牌弹窗（一次性显示、可复制）
- `queries/cluster-registry.ts`：`clusterRegistryQuery` 与四个 mutation（令牌明文不作为 mutation 的返回值，直接交给页面的本地状态）
- 导航增加「集群管理」；zh / ja 文案

## 功能二：AgentSDK 认证

**用户故事**：运维把令牌配进 Agent，Master 以 `warn` 运行时能在日志里看到哪些集群还没带令牌；
全部带上后切到 `enforce`，没有有效令牌的请求一律被拒绝。

**后端实现**

- `agentsdk/auth.go` [新增]：`authenticateAgent` 中间件、`agentIdentity`、限频日志
- `agentsdk/{snapshot,heartbeat,command,result}.go`：改用已认证的集群
- `mq/`：`AckCommand` 增加集群参数（memory / redis 两个实现）
- `agentsdk.AuthStatusTracker`：记录各集群最近一次上报是否带了有效令牌，供管理页显示。
  只在进程内存里：Master 重启后先显示「未认证」，下一次上报（10 秒内）恢复；多副本之间不共享
- `config/`：`MASTER_AGENT_AUTH_MODE`、`MASTER_AGENTSDK_MAX_SNAPSHOT_MB`
- Agent：`config/`（`AGENT_MASTER_TOKEN`）、`gateway/master_gateway.go`、`service/command/`（核对指令的集群）、`scheduler/`（日志区分身份被拒与网络故障）
  - 集群核对放在指令服务的 `Execute` 入口，与「目标是否合法资源名」的核对在同一处：调度层只管编排，是否执行一条指令由指令服务判断

**边界**

- `enforce` 下没带令牌 → 401；令牌无效 → 401；令牌有效但自报集群不同 → 403
- 回报不属于自己集群的指令结果 → 403，原指令状态不变
- 令牌轮换后旧令牌立即 401
- 删除登记后令牌失效；`enforce` 下该集群的请求被拒绝

## 功能三：部署清单（config 仓，不在本仓）

- Agent 专用 ServiceAccount 与 ClusterRole：采集所需的只读规则 + 重启、扩缩容、删 Pod、封锁节点、读日志
- 内置部署功能（`apply_manifests`）默认不授权；需要时按命名空间单独绑定
- controller / web / ClickHouse：`automountServiceAccountToken: false`
- NetworkPolicy：Agent 入口只允许 Agent 访问，接口入口只允许 web 访问
- Agent 的令牌放进 Secret，经 `AGENT_MASTER_TOKEN` 注入

## 上线顺序

1. Master 与 Agent 升级，Master 以 `warn` 运行（默认值）——此时行为与升级前一致
2. 在管理页登记集群、签发令牌，把令牌配进 Agent
3. 日志里不再出现「未带令牌」的告警后，把模式改为 `enforce`

表结构有变更（`clusters` 表），上线需清库。

## 裁决记录（2026-11-14，用户：「其他按照推荐吧」）

| 事项 | 裁定 |
|------|------|
| 令牌存放 | 集群注册表 + 管理页签发，库里只存哈希 |
| 过渡方式 | 三档开关；dev 验证后直接 `enforce`，prod 先 `warn` 后 `enforce` |
| 网络策略 | 加 |
| Agent 读取 Secret 的权限 | 先保留（Secret 页面与查看内容是现有功能） |
| 内置部署功能的权限 | 默认不给，按命名空间单独授权 |

## 文件变更清单

```
atlhyper_master_v2/
├── agentsdk/
│   ├── auth.go                      [新增]
│   ├── clustertoken/token.go        [新增]
│   ├── server.go / snapshot.go / heartbeat.go / command.go / result.go   [修改]
├── mq/interfaces.go, memory/bus.go, redis/bus.go                          [修改]
├── database/                        [修改] clusters 表、ClusterRepository
├── service/operations/cluster_registry.go, service/query/cluster_registry.go [新增]
├── service/interfaces.go, factory.go, query/impl.go                       [修改]
├── gateway/handler/admin/cluster_registry.go                              [新增]
├── gateway/routes.go, middleware/permission.go                            [修改]
├── model/cluster_registry.go, cluster_registry_validation.go, convert/cluster_registry.go [新增]
├── config/                          [修改] MASTER_AGENT_AUTH_MODE、MASTER_AGENTSDK_MAX_SNAPSHOT_MB
├── agent_entry_settings.go          [新增] 启动时校验 Agent 入口配置
└── master.go                        [修改] 装配
atlhyper_agent_v2/
├── config/                          [修改] AGENT_MASTER_TOKEN
├── gateway/master_gateway.go, errors.go [修改/新增]
├── service/command/command.go       [修改] 执行前核对指令所属集群
└── scheduler/scheduler.go           [修改] 日志区分身份被拒与网络故障
atlhyper_web/
├── src/app/admin/clusters/          [新增]
├── src/queries/cluster-registry.ts  [新增]
└── i18n、导航                        [修改]
```

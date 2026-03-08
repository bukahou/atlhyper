# AtlHyper

**面向 AI 时代的次世代 Kubernetes SRE 平台**

[English](../../README.md) | 中文 | [日本語](README_ja.md)

---

AtlHyper 是面向 AI 时代的次世代 SRE 平台，采用 Master-Agent 架构管理多集群 Kubernetes 环境。提供四信号域全栈可观测（Metrics / APM / Logs / SLO）、算法驱动的 AIOps 引擎和 AI 辅助运维，目标是构建「系统运行认知模型」，让系统理解自己。

---

## 功能特性

- **多集群管理** — 单一控制台管理多个 Kubernetes 集群，Agent 自动注册
- **实时监控** — Pod、Node、Deployment 等 21 种 K8s 资源实时状态与指标可视化
- **四信号域可观测** — 基于 ClickHouse + OTel Collector 的 Metrics / APM / Logs / SLO 全栈可观测
- **分布式追踪 (APM)** — Trace 瀑布图、Span 详情、服务拓扑、延迟分布、数据库调用分析
- **日志查询** — 多维度过滤（服务/级别/来源类）、柱状图时间线、结构化日志详情、Trace 关联
- **SLO 监控** — Ingress（Traefik）+ 服务网格（Linkerd）双层 SLO 追踪，延迟分布、错误预算、状态码分布
- **AIOps 引擎** — 依赖图构建、EMA 动态基线、三阶段风险评分、状态机、事件生命周期管理
- **因果拓扑图** — 四层有向无环图（Ingress→Service→Pod→Node），风险传播可视化
- **AI 助手** — 多模型驱动的自然语言运维（Chat + Tool Use），支持 Gemini / OpenAI / Claude / Ollama（本地），事件摘要与根因分析
- **AI 多角色路由** — 三种 AI 角色（background / chat / analysis），按角色独立路由 Provider，每角色每日 Token/调用预算控制
- **AI 事件分析** — 事件创建时自动后台分析，深度调查支持多轮 Tool Calling（最多 8 轮），结构化报告含置信度评分
- **告警通知** — 支持邮件 (SMTP) 和 Slack (Webhook) 通知
- **远程运维** — 远程执行 kubectl 命令、重启 Pod、调整副本数、隔离节点
- **审计日志** — 完整的操作历史记录与用户追踪
- **多语言支持** — 中文、日语

---

## 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| **Master** | Go 1.24 + net/http + SQLite | 中央控制、数据聚合、API 服务、AIOps 引擎 |
| **Agent** | Go 1.24 + client-go + ClickHouse | 集群数据采集、OTel 数据查询、命令执行 |
| **Web** | Next.js 16 + React 19 + Tailwind CSS 4 + ECharts + G6 | 可视化管理界面 |
| **可观测** | ClickHouse + OTel Collector + Linkerd | 时序存储、遥测采集、服务网格 |
| **AI** | Gemini / OpenAI / Claude / Ollama (Chat + Tool Use) | AI 对话运维、多角色路由、事件分析 |

---

## 系统架构

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                              AtlHyper 平台                                        │
├──────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────┐    ┌──────────────────────────────────────────────────────────┐    │
│  │  Web UI  │───▶│                        Master                           │    │
│  │(Next.js) │◀───│                                                          │    │
│  └──────────┘    │  ┌─────────┐ ┌────────┐ ┌─────────┐ ┌────────────────┐  │    │
│                  │  │ Gateway │ │DataHub │ │ Service │ │   Database     │  │    │
│                  │  │  (API)  │ │ (内存) │ │(业务层) │ │   (SQLite)     │  │    │
│                  │  └─────────┘ └────────┘ └─────────┘ └────────────────┘  │    │
│                  │  ┌──────────────────┐   ┌──────────────────────────┐     │    │
│                  │  │  AIOps Engine    │   │  AI (Multi-LLM+角色路由) │     │    │
│                  │  │ 依赖图│基线│风险  │   │ Gemini│OpenAI│Claude   │     │    │
│                  │  │ 状态机│事件存储   │   │ Ollama│角色预算       │     │    │
│                  │  │       │          │   └──────────────────────────┘     │    │
│                  │  └──────────────────┘                                    │    │
│                  └──────────────────────────────────────────────────────────┘    │
│                                          │                                       │
│         ┌────────────────────────────────┼────────────────────────────────┐      │
│         │                                │                                │      │
│         ▼                                ▼                                ▼      │
│  ┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐  │
│  │  Agent (集群 A)  │         │  Agent (集群 B)  │         │  Agent (集群 N)  │  │
│  │                  │         │                  │         │                  │  │
│  │ SDK (K8s+CH)     │         │ SDK (K8s+CH)     │         │ SDK (K8s+CH)     │  │
│  │ Repository       │         │ Repository       │         │ Repository       │  │
│  │ Concentrator     │         │ Concentrator     │         │ Concentrator     │  │
│  │ Service          │         │ Service          │         │ Service          │  │
│  │ Scheduler        │         │ Scheduler        │         │ Scheduler        │  │
│  └────────┬─────────┘         └────────┬─────────┘         └────────┬─────────┘  │
│           │                            │                            │            │
│           ▼                            ▼                            ▼            │
│  ┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐  │
│  │  Kubernetes 集群  │         │  Kubernetes 集群  │         │  Kubernetes 集群  │  │
│  │                  │         │                  │         │                  │  │
│  │ ┌──────────────┐ │         │ ┌──────────────┐ │         │ ┌──────────────┐ │  │
│  │ │OTel Collector│ │         │ │OTel Collector│ │         │ │OTel Collector│ │  │
│  │ │node_exporter │ │         │ │node_exporter │ │         │ │node_exporter │ │  │
│  │ │   Linkerd    │ │         │ │   Linkerd    │ │         │ │   Linkerd    │ │  │
│  │ │  ClickHouse  │ │         │ │  ClickHouse  │ │         │ │  ClickHouse  │ │  │
│  │ └──────────────┘ │         │ └──────────────┘ │         │ └──────────────┘ │  │
│  └──────────────────┘         └──────────────────┘         └──────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## 界面截图

### 集群总览
集群健康状态、工作负载概要、SLO 概要、节点资源使用量、最近告警。

![集群总览](../img/overview.png)

### Pod 管理
跨命名空间 Pod 列表，支持筛选过滤，右侧抽屉展示 Pod 详情（基本信息、容器、卷挂载、网络、调度）。

![Pod 管理](../img/cluster-pod.png)

### Node 管理
节点列表与详情抽屉，展示系统信息、角色、Pod CIDR、容器运行时版本。支持节点隔离/解除隔离操作。

![Node 管理](../img/cluster-node.png)

### 节点指标
全集群节点级硬件指标：CPU、内存、磁盘、温度，支持多时间范围（1h/6h/1d/7d）和多粒度（1m/5m/15m）。

![节点指标](../img/observe-metrics.png)

### APM 链路追踪
分布式追踪分析：延迟分布直方图、Trace 瀑布图、Span 详情（含数据库属性和 K8s 上下文），支持 Trace ↔ Log 关联。

![APM 链路追踪](../img/observe-apm.png)

### 日志查询
多维度日志过滤（服务/级别/来源类），时间线柱状图，结构化日志详情（含 Trace ID、K8s 资源信息），支持全文搜索。

![日志查询](../img/observe-logs.png)

### SLO 监控
域级 SLO 总览（可用性、P95 延迟、错误率、错误预算），延迟分布直方图、请求方法分布、状态码分布。

![SLO 监控](../img/observe-slo.png)

### AIOps 风险仪表盘
集群风险评分（0-100）、高风险实体列表，展示局部风险/最终风险/风险等级和首次异常时间。

![AIOps 风险](../img/aiops-risk.png)

### AIOps 因果拓扑
四层依赖图（Node→Pod→Service→Ingress），风险传播可视化。节点详情面板展示基线指标和因果链路。

![AIOps 拓扑](../img/aiops-topology.png)

### AI 助手
多模型驱动的自然语言运维对话（支持 Gemini / OpenAI / Claude / Ollama），Tool Use（事件查询、分析）。三种 AI 角色（background / chat / analysis）按角色独立路由 Provider 并控制预算。自动输出结构化事件摘要和根因分析。

![AI 助手](../img/aiops-chat.png)

---

## 数据流

### Agent → Master（快照上报 + 命令执行）

```
[快照流]
K8s SDK ──▶ Repository ──▶ SnapshotService ──▶ Scheduler ──▶ Master
• K8s 资源: Pod、Node、Deployment、Service、Ingress 等 21 种资源
• OTel 数据: ClickHouse 查询 Metrics / APM / Logs / SLO 四信号域
• 时序聚合: Concentrator 环形缓冲（1 小时 × 1 分钟粒度）

[命令流]
Master ──▶ Agent Poll ──▶ CommandService ──▶ K8s SDK ──▶ 结果 → Master

[心跳流]
Agent ──▶ 定时心跳 ──▶ Master（连接状态维护）
```

### 可观测管道（OTel → ClickHouse → Agent）

```
[节点指标]  node_exporter ──▶ OTel Collector ──▶ ClickHouse
[Ingress]   Traefik ──▶ OTel Collector ──▶ ClickHouse
[Mesh]      Linkerd Proxy ──▶ OTel Collector ──▶ ClickHouse
[Traces]    应用 SDK ──▶ OTel Collector ──▶ ClickHouse
[Logs]      应用日志 ──▶ Filebeat ──▶ OTel Collector ──▶ ClickHouse

                    ClickHouse ◀── Agent 定时查询
```

---

## 部署

### 环境要求

- Go 1.24+
- Node.js 20+
- Kubernetes 集群（用于 Agent 部署）
- ClickHouse（可观测数据存储）

### 快速开始（开发环境）

**1. 启动 Master**
```bash
export MASTER_ADMIN_USERNAME=admin
export MASTER_ADMIN_PASSWORD=$(openssl rand -base64 16)
export MASTER_JWT_SECRET=$(openssl rand -base64 32)

cd cmd/atlhyper_master_v2
go run main.go
# Gateway: :8080, AgentSDK: :8081
```

**2. 启动 Agent（在 K8s 集群中）**
```bash
# 集群 ID 自动检测（kube-system UID），也可通过环境变量指定
export AGENT_MASTER_URL=http://<MASTER_IP>:8081
# export AGENT_CLUSTER_ID=my-cluster  # 可选，默认自动检测

cd cmd/atlhyper_agent_v2
go run main.go
```

**3. 启动 Web**
```bash
cd atlhyper_web
npm install && npm run dev
# 访问: http://localhost:3000
```

### 配置参考

#### Master 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `MASTER_ADMIN_USERNAME` | 是 | - | 管理员用户名 |
| `MASTER_ADMIN_PASSWORD` | 是 | - | 管理员密码 |
| `MASTER_JWT_SECRET` | 是 | - | JWT 签名密钥 |
| `MASTER_GATEWAY_PORT` | 否 | `8080` | Web/API 端口 |
| `MASTER_AGENTSDK_PORT` | 否 | `8081` | Agent 数据端口 |
| `MASTER_LOG_LEVEL` | 否 | `info` | 日志级别 |

#### Agent 环境变量

| 变量 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `AGENT_MASTER_URL` | 是 | - | Master AgentSDK 地址 |
| `AGENT_CLUSTER_ID` | 否 | 自动检测 | 集群唯一标识（默认使用 kube-system UID） |
| `AGENT_CLICKHOUSE_DSN` | 否 | - | ClickHouse 连接地址（启用 OTel 查询） |

---

## 项目结构

```
atlhyper/
├── atlhyper_master_v2/     # Master（中央控制）— 41k 行
│   ├── gateway/            #   HTTP API 网关
│   │   └── handler/        #     Handler（k8s/observe/aiops/admin/slo 子目录）
│   ├── service/            #   业务逻辑（query + operations）
│   ├── datahub/            #   内存数据存储
│   ├── database/           #   持久化 (SQLite)
│   ├── processor/          #   数据处理
│   ├── agentsdk/           #   Agent 通信层
│   ├── mq/                 #   消息队列
│   ├── aiops/              #   AIOps 引擎
│   ├── ai/                 #   AI 助手 (Gemini/OpenAI/Claude/Ollama)
│   ├── slo/                #   SLO 路由更新
│   ├── notifier/           #   告警通知
│   └── config/             #   配置
│
├── atlhyper_agent_v2/      # Agent（集群代理）— 20k 行
│   ├── sdk/                #   K8s + ClickHouse SDK
│   ├── repository/         #   数据仓库 (K8s + CH 查询)
│   ├── service/            #   快照/命令服务
│   ├── concentrator/       #   OTel 时序聚合（环形缓冲）
│   ├── scheduler/          #   调度器
│   └── gateway/            #   Agent↔Master 通信
│
├── atlhyper_web/           # Web 前端 — 55k 行
│   ├── src/app/            #   Next.js 页面
│   ├── src/components/     #   React 组件
│   ├── src/api/            #   API 客户端
│   ├── src/datasource/     #   数据源层（API + mock 降级）
│   └── src/i18n/           #   国际化 (中文/日语)
│
├── model_v3/               # 共享模型 (cluster/agent/metrics/slo/command/apm/log)
├── common/                 # 工具包 (logger/crypto/gzip)
├── cmd/                    # 入口程序
└── docs/                   # 文档
```

---

## AIOps 引擎

算法驱动的 AIOps 引擎，实现自动化异常检测、根因定位和事件生命周期管理。核心设计原则：**算法可解释** — 每个风险评分都能追溯到具体公式和输入指标，不是 ML 黑盒。

### M1 — 依赖图（Correlator）

从 `ClusterSnapshot` 自动构建四层有向无环图（DAG）：

```
Ingress ──routes_to──▶ Service ──selects──▶ Pod ──runs_on──▶ Node
                         │
                         └──calls──▶ Service（Linkerd 服务间流量）
```

- **数据源**: K8s API（资源关系）+ Linkerd outbound（服务间调用）+ OTel Traces（追踪链路）
- **图结构**: 正向/反向邻接表，支持 BFS 链路追踪
- **持久化**: 每次快照后异步写入 SQLite

### M2 — 基线引擎（Baseline）

**EMA（指数移动平均）+ 3σ 动态基线**，双通道异常检测：

**通道 A — 统计型检测：**

```
EMA_t = α × x_t + (1-α) × EMA_{t-1}     (α = 0.033, 等效 60 采样点窗口)
异常分 = sigmoid(|x - EMA| / σ - 3)       (偏离度 > 3σ 即为异常)
```

| 实体 | 监控指标 |
|------|---------|
| Node | cpu_usage, memory_usage, disk_usage, psi_cpu/memory/io |
| Pod | restart_count, is_running, not_ready_containers |
| Service (Linkerd) | error_rate, avg_latency, request_rate |
| Ingress (Traefik) | error_rate, avg_latency |

**通道 B — 确定性检测（绕过冷启动）：**

| 检测项 | 分数 |
|--------|------|
| OOMKilled | 0.95 |
| CrashLoopBackOff | 0.90 |
| 配置错误 | 0.80 |
| K8s Critical Event（5 分钟内） | 0.85 |
| Deployment 不可用 ≥75% | 0.95 |

### M3 — 风险评分（Risk Scorer）

三阶段流水线，从局部指标到全局拓扑：

```
Stage 1 — 局部风险:  R_local = max(R_stat, R_det)
Stage 2 — 时序衰减:  W_time = 0.7 + 0.3 × (1 - exp(-Δt / τ))
Stage 3 — 图传播:    R_final = f(R_weighted, avg(R_final(deps)), SLO_context)
```

| R_final | 等级 |
|---------|------|
| ≥ 0.8 | Critical |
| ≥ 0.6 | High |
| ≥ 0.4 | Medium |
| ≥ 0.2 | Low |
| < 0.2 | Healthy |

### M4 — 状态机（State Machine）

```
                    R>0.2 持续≥2min           R>0.5 持续≥5min
  Healthy ──────────────────▶ Warning ──────────────────▶ Incident
     ▲  R<0.15 持续≥5min       │                            │
     └──────────────────────────┘          R<0.15 持续≥10min │
                                                             ▼
                               R>0.2 立即复发             Recovery
                    Warning ◀─────────────────────────────── │
                                                             │
                                           定时检查（10min） │
                                              Stable ◀───────┘
```

### M5 — 事件存储（Incident Store）

SQLite 持久化，结构化事件记录：

| 数据 | 内容 |
|------|------|
| **Incident** | ID、集群、状态、严重性、根因实体、峰值风险、持续时间 |
| **Entity** | 受影响实体列表（含 R_local / R_final / 角色） |
| **Timeline** | 状态变更时间线 |
| **Statistics** | MTTR、复发率、严重性分布、Top 根因 |

### M6 — AI 增强（AI Enhancer）

LLM 驱动的事件分析，三种 AI 角色：

| 角色 | 触发方式 | 说明 |
|------|---------|------|
| **background** | 自动（事件创建/升级时） | 快速摘要、处置建议、相似事件。频率限制（60s/事件），结果缓存 24h |
| **chat** | 用户发起 | 交互式自然语言运维，SSE 流式推送 |
| **analysis** | 用户发起 | 深度多轮调查（最多 8 轮 × 5 次 Tool Call），结构化报告含置信度评分 |

- **多 Provider**: Gemini / OpenAI / Claude / Ollama（本地），每个角色可路由到不同 Provider
- **角色预算**: 每角色每日 Token 限额和调用次数限额，预算用尽可降级到备用 Provider
- **报告持久化**: 所有 AI 报告（摘要、根因分析、调查步骤）持久化到 SQLite

---

## 安全

- **禁止硬编码** API 密钥、密码或密钥到代码中
- 所有凭证使用环境变量
- AI API 密钥加密存储在数据库中（通过 Web UI 配置）
- K8s Secret 内容脱敏展示

---

## 许可证

MIT

---

## 链接

- [GitHub 仓库](https://github.com/bukahou/atlhyper)

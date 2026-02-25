# AtlHyper

**Kubernetes 多集群监控与可观测性平台**

[English](../README.md) | 中文 | [日本語](README_ja.md)

---

AtlHyper 是面向 Kubernetes 环境的监控与可观测性平台，采用 Master-Agent 架构，提供多集群统一管理、全栈可观测（Metrics / Traces / Mesh）、算法驱动的 AIOps 引擎和 AI 辅助运维。

---

## 功能特性

- **多集群管理** — 单一控制台管理多个 Kubernetes 集群
- **实时监控** — Pod、Node、Deployment 等 21 种 K8s 资源实时状态与指标可视化
- **异常检测** — 自动检测 CrashLoopBackOff、OOMKilled、ImagePullBackOff 等异常
- **全栈可观测** — 基于 ClickHouse + OTel Collector + Linkerd 的 Metrics / Traces / Mesh 可观测
- **SLO 监控** — Ingress（Traefik）+ 服务网格（Linkerd）双层 SLO 追踪，含延迟分布、状态码分布
- **服务网格拓扑** — Linkerd 服务依赖可视化（mTLS 流量、延迟分布、请求成功率）
- **AIOps 引擎** — 依赖图构建、EMA 动态基线、三阶段风险评分、状态机、事件生命周期管理
- **AI 助手** — Gemini 驱动的自然语言运维（Chat + Tool Use），含事件摘要与根因分析
- **告警通知** — 支持邮件 (SMTP) 和 Slack (Webhook) 通知
- **远程运维** — 远程执行 kubectl 命令、重启 Pod、调整副本数
- **审计日志** — 完整的操作历史记录与用户追踪
- **多语言支持** — 中文、日语

---

## 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| **Master** | Go 1.24 + net/http + SQLite | 中央控制、数据聚合、API 服务、AIOps 引擎 |
| **Agent** | Go 1.24 + client-go + ClickHouse | 集群数据采集、OTel 数据查询、命令执行 |
| **Web** | Next.js 16 + React 19 + Tailwind CSS 4 + ECharts + G6 | 可视化管理界面 |
| **可观测** | ClickHouse + OTel Collector + Linkerd | 指标存储、遥测采集、服务网格 |
| **AI** | Gemini API (Chat + Tool Use) | AI 对话运维、事件分析 |

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
│                  │  │  AIOps Engine    │   │      AI (Gemini)        │     │    │
│                  │  │ 依赖图│基线│风险  │   │  Chat│Tool Use│事件分析  │     │    │
│                  │  │ 状态机│事件存储   │   └──────────────────────────┘     │    │
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

## 数据流

### 1. Agent → Master（快照上报 + 命令执行）

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Agent ↔ Master 数据流                              │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [快照流]                                                                │
│  K8s SDK ──▶ Repository ──▶ SnapshotService ──▶ Scheduler ──▶ Master   │
│  • K8s 资源: Pod、Node、Deployment、Service、Ingress 等 21 种资源       │
│  • SLO 指标: ClickHouse 查询 Traefik + Linkerd 数据                    │
│  • 节点指标: ClickHouse 查询 node_exporter 数据                         │
│  • APM 数据: ClickHouse 查询分布式追踪数据                               │
│  • 时序聚合: Concentrator 环形缓冲（1 小时 × 1 分钟粒度）               │
│                                                                          │
│  [命令流]                                                                │
│  Master ──▶ Agent Poll ──▶ CommandService ──▶ K8s SDK ──▶ 结果 → Master│
│  • 操作: 重启 Pod、调整副本、隔离节点等                                    │
│                                                                          │
│  [心跳流]                                                                │
│  Agent ──▶ 定时心跳 ──▶ Master（连接状态维护）                            │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2. 可观测管道（OTel → ClickHouse → Agent）

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        可观测数据管道                                      │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [节点指标]                                                              │
│  node_exporter ──▶ OTel Collector ──▶ ClickHouse                       │
│  • CPU、内存、磁盘、网络、PSI 压力、TCP、温度                              │
│                                                                          │
│  [Ingress SLO]                                                           │
│  Traefik ──▶ OTel Collector ──▶ ClickHouse                             │
│  • RPS、成功率、延迟分布（histogram 桶）、状态码分布                       │
│                                                                          │
│  [服务网格 SLO]                                                          │
│  Linkerd Proxy ──▶ OTel Collector ──▶ ClickHouse                       │
│  • 服务间流量拓扑、P50/P95/P99 延迟、请求成功率、mTLS                    │
│                                                                          │
│  [分布式追踪]                                                            │
│  应用 SDK ──▶ OTel Collector ──▶ ClickHouse                             │
│  • TraceID/SpanID、服务拓扑、操作统计                                     │
│                                                                          │
│                    ClickHouse ◀── Agent 定时查询                         │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 界面截图

### 集群总览
实时展示集群健康状态、资源使用情况和最近告警。

![集群总览](../img/overview.png)

### Pod 管理
跨命名空间列表、过滤和管理 Pod，展示详细状态。

![Pod 管理](../img/cluster_pod.png)

### 告警面板
查看和分析集群告警，支持过滤和 AI 智能分析。

![告警面板](../img/cluster_alert.png)

### 节点指标
详细的节点级指标与历史趋势图表。

![节点指标](../img/system_metrics.png)

### SLO 监控
基于 Ingress + 服务网格的双层 SLO 追踪。

![SLO 总览](../img/workbench_slo_overview.png)

![SLO 详情](../img/workbench_slo.png)

---

## 部署

### 环境要求

- Go 1.24+
- Node.js 20+
- Kubernetes 集群（用于 Agent 部署）
- ClickHouse（可观测数据存储）
- Docker（容器化部署）

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
cd cmd/atlhyper_agent_v2
go run main.go \
  --cluster-id=my-cluster \
  --master=http://<MASTER_IP>:8081
```

**3. 启动 Web**
```bash
cd atlhyper_web
npm install && npm run dev
# 访问: http://localhost:3000
```

### Kubernetes 部署（YAML）

部署顺序: **ClickHouse + OTel Collector + Linkerd → Master → Agent → Web**

```bash
cd deploy/k8s

# 1. 创建命名空间和配置
kubectl apply -f atlhyper-config.yaml

# 2. 部署 Master
kubectl apply -f atlhyper-master.yaml

# 3. 部署 Agent
kubectl apply -f atlhyper-agent.yaml

# 4. 部署 Web
kubectl apply -f atlhyper-web.yaml

# 5. (可选) Traefik IngressRoute
kubectl apply -f atlhyper-traefik.yaml
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

#### Agent 配置

| 参数 | 必填 | 说明 |
|------|------|------|
| `--cluster-id` | 是 | 集群唯一标识 |
| `--master` | 是 | Master AgentSDK 地址 |

---

## 项目结构

```
atlhyper/
├── atlhyper_master_v2/     # Master（中央控制）
│   ├── gateway/            #   HTTP API 网关
│   ├── datahub/            #   内存数据存储
│   ├── database/           #   持久化 (SQLite)
│   ├── service/            #   业务逻辑
│   ├── aiops/              #   AIOps 引擎
│   ├── ai/                 #   AI 助手 (Gemini)
│   ├── slo/                #   SLO 计算
│   ├── notifier/           #   告警通知
│   ├── processor/          #   数据处理
│   ├── agentsdk/           #   Agent 通信层
│   ├── mq/                 #   消息队列
│   └── config/             #   配置
│
├── atlhyper_agent_v2/      # Agent（集群代理）
│   ├── sdk/                #   K8s + ClickHouse SDK
│   ├── repository/         #   数据仓库 (K8s + CH 查询)
│   ├── service/            #   快照/命令服务
│   ├── concentrator/       #   OTel 时序聚合（环形缓冲）
│   ├── scheduler/          #   调度器
│   └── gateway/            #   Agent↔Master 通信
│
├── atlhyper_web/           # Web 前端
│   ├── src/app/            #   Next.js 页面
│   ├── src/components/     #   React 组件
│   ├── src/api/            #   API 客户端
│   └── src/i18n/           #   国际化 (中文/日语)
│
├── model_v2/               # 共享模型 (K8s 资源)
├── model_v3/               # 共享模型 (SLO/APM/Metrics/Log)
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

- 冷启动：前 100 个数据点只学习不告警（零值指标可快速通道 10 点结束）

**通道 B — 确定性检测（绕过冷启动）：**

| 检测项 | 分数 |
|--------|------|
| OOMKilled | 0.95 |
| CrashLoopBackOff | 0.90 |
| 配置错误 | 0.80 |
| K8s Critical Event（5 分钟内） | 0.85 |
| Deployment 不可用 ≥75% | 0.95 |

两路结果合并：同一指标取 Score 更高者。

### M3 — 风险评分（Risk Scorer）

三阶段流水线，从局部指标到全局拓扑：

**Stage 1 — 局部风险 (R_local):**
```
通道1（统计）: R_stat = Σ(w_i × score_i)
通道2（确定性）: R_det = max(score_i) × breadthBoost(n)
R_local = max(R_stat, R_det)
```

**Stage 2 — 时序衰减 (W_time):**
```
W_time = 0.7 + 0.3 × (1 - exp(-Δt / τ))
```
首次检测 W=0.7，持续 5 分钟 ≈0.82，持续 10 分钟 ≈0.93

**Stage 3 — 图传播 (R_final):**
```
拓扑排序: Node(0) → Pod(1) → Service(2) → Ingress(3)
R_final(v) = max(R_weighted(v), α × R_weighted(v) + (1-α) × avg(R_final(deps)))
```
叠加 SLO 上下文：根据最大错误预算燃烧速率修正

**风险等级映射:**

| R_final | 等级 |
|---------|------|
| ≥ 0.8 | Critical |
| ≥ 0.6 | High |
| ≥ 0.4 | Medium |
| ≥ 0.2 | Low |
| < 0.2 | Healthy |

### M4 — 状态机（State Machine）

五状态生命周期管理：

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

- 过期清理：30 分钟未评估的条目自动关闭（Pod 被删除后）
- 复发追踪：recovery → warning 时 recurrence 计数+1

### M5 — 事件存储（Incident Store）

SQLite 持久化，结构化事件记录：

| 数据 | 内容 |
|------|------|
| **Incident** | ID、集群、状态、严重性、根因实体、峰值风险、持续时间 |
| **Entity** | 受影响实体列表（含 R_local / R_final / 角色） |
| **Timeline** | 状态变更时间线（anomaly_detected → state_change → recovery_started） |
| **Statistics** | MTTR、复发率、严重性分布、Top 根因 |

AI 增强（可选）：Gemini LLM 生成事件摘要、根因分析和处置建议。

---

## 安全

### 敏感信息

- **禁止硬编码** API 密钥、密码或密钥到代码中
- 所有凭证使用环境变量
- AI API 密钥加密存储在数据库中（通过 Web UI 配置）

### 提交前检查

```bash
# 扫描可能的 API 密钥泄漏
grep -rE "sk-[a-zA-Z0-9]{20,}|AIza[a-zA-Z0-9]{30,}" \
  --include="*.go" --include="*.ts" --include="*.tsx" .
```

### .gitignore 忽略的文件

- `atlhyper_master_v2/database/sqlite/data/` — 数据库文件
- `atlhyper_web/.env.local` — 本地环境
- `*.db` — 所有 SQLite 数据库

---

## 许可证

MIT

---

## 链接

- [GitHub 仓库](https://github.com/bukahou/atlhyper)

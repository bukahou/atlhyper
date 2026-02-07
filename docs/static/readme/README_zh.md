# AtlHyper

**轻量级 Kubernetes 多集群监控与运维平台**

[English](../README.md) | 中文 | [日本語](README_ja.md)

---

AtlHyper 是一个面向轻量级 Kubernetes 环境的监控与管理平台，采用 Master-Agent 架构，支持多集群统一管理、实时资源监控、异常检测、SLO 追踪和远程运维操作。

---

## 功能特性

- **多集群管理** — 单一控制台管理多个 Kubernetes 集群
- **实时监控** — Pod、Node、Deployment 状态实时展示与指标可视化
- **异常检测** — 自动检测 CrashLoopBackOff、OOMKilled、ImagePullBackOff 等异常
- **SLO 监控** — 基于 Ingress 指标追踪服务可用性、延迟和错误率
- **告警通知** — 支持邮件 (SMTP) 和 Slack (Webhook) 通知
- **远程运维** — 远程执行 kubectl 命令、重启 Pod、调整副本数
- **AI 助手** — 自然语言交互进行集群运维（可选）
- **审计日志** — 完整的操作历史记录与用户追踪
- **多语言支持** — 中文、英文、日语

---

## 技术栈

| 组件 | 技术 | 说明 |
|------|------|------|
| **Master** | Go + Gin + SQLite/MySQL | 中央管控、数据聚合、API 服务 |
| **Agent** | Go + controller-runtime | 集群数据采集、命令执行 |
| **Metrics** | Go (DaemonSet) | 节点级指标采集 (CPU/内存/磁盘/网络) |
| **Web** | Next.js 15 + TypeScript + Tailwind CSS | 现代响应式管理界面 |

---

## 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AtlHyper 平台                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────┐     ┌─────────────────────────────────────────────────┐   │
│   │   Web UI    │────▶│                    Master                       │   │
│   │  (Next.js)  │◀────│                                                 │   │
│   └─────────────┘     │  ┌─────────┐  ┌──────────┐  ┌───────────────┐   │   │
│                       │  │ Gateway │  │ DataHub  │  │   Services    │   │   │
│                       │  │  (API)  │  │ (内存)   │  │ (SLO/告警)    │   │   │
│                       │  └─────────┘  └──────────┘  └───────────────┘   │   │
│                       │                     │                           │   │
│                       │              ┌──────┴──────┐                    │   │
│                       │              │   数据库    │                    │   │
│                       │              │(SQLite/MySQL)│                   │   │
│                       └──────────────┴──────────────┴───────────────────┘   │
│                                          │                                  │
│            ┌─────────────────────────────┼─────────────────────────────┐    │
│            │                             │                             │    │
│            ▼                             ▼                             ▼    │
│   ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   │  Agent (集群A)  │         │  Agent (集群B)  │         │  Agent (集群N)  │
│   │                 │         │                 │         │                 │
│   │  ┌───────────┐  │         │  ┌───────────┐  │         │  ┌───────────┐  │
│   │  │  Source   │  │         │  │  Source   │  │         │  │  Source   │  │
│   │  │ ├─ 事件   │  │         │  │ ├─ 事件   │  │         │  │ ├─ 事件   │  │
│   │  │ ├─ 快照   │  │         │  │ ├─ 快照   │  │         │  │ ├─ 快照   │  │
│   │  │ └─ 指标   │  │         │  │ └─ 指标   │  │         │  │ └─ 指标   │  │
│   │  ├───────────┤  │         │  ├───────────┤  │         │  ├───────────┤  │
│   │  │  Executor │  │         │  │  Executor │  │         │  │  Executor │  │
│   │  └───────────┘  │         │  └───────────┘  │         │  └───────────┘  │
│   └────────┬────────┘         └────────┬────────┘         └────────┬────────┘
│            │                           │                           │        │
│            ▼                           ▼                           ▼        │
│   ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   │   Kubernetes    │         │   Kubernetes    │         │   Kubernetes    │
│   │     集群 A      │         │     集群 B      │         │     集群 N      │
│   │ ┌─────────────┐ │         │ ┌─────────────┐ │         │ ┌─────────────┐ │
│   │ │   Metrics   │ │         │ │   Metrics   │ │         │ │   Metrics   │ │
│   │ │ (DaemonSet) │ │         │ │ (DaemonSet) │ │         │ │ (DaemonSet) │ │
│   │ └─────────────┘ │         │ └─────────────┘ │         │ └─────────────┘ │
│   └─────────────────┘         └─────────────────┘         └─────────────────┘
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 数据流

AtlHyper 由四个模块组成，各有独立的数据流：

### 1. Agent 数据流（4 条流）

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Agent → Master 数据流                              │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [事件流]                                                                │
│  K8s Watch ──▶ 异常过滤 ──▶ DataHub ──▶ Pusher ──▶ Master               │
│  • 检测: CrashLoop、OOM、ImagePull、NodeNotReady 等                      │
│                                                                          │
│  [快照流]                                                                │
│  SDK.List() ──▶ Snapshot ──▶ Pusher ──▶ Master                          │
│  • 资源: Pod、Node、Deployment、Service、Ingress 等                      │
│                                                                          │
│  [指标流]                                                                │
│  Metrics DaemonSet ──▶ Agent Gateway ──▶ Receiver ──▶ Pusher ──▶ Master │
│  • 节点指标: 每个节点的 CPU、内存、磁盘、网络                              │
│                                                                          │
│  [命令流]                                                                │
│  Master ──▶ Agent Gateway ──▶ Executor ──▶ K8s SDK ──▶ 结果 ──▶ Master  │
│  • 操作: 重启 Pod、调整副本、隔离节点等                                    │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2. Metrics DaemonSet 数据流

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     指标采集器（每个节点）                                  │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                   │
│  │ /proc/stat  │    │/proc/meminfo│    │/proc/diskstats│                 │
│  │ /proc/net   │    │   syscall   │    │/proc/mounts │                   │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                   │
│         │                  │                  │                          │
│         ▼                  ▼                  ▼                          │
│  ┌─────────────────────────────────────────────────────┐                 │
│  │              指标采集器 (Go)                         │                 │
│  │  • CPU: 使用率、每核心、负载均值                      │                 │
│  │  • 内存: 已用、可用、缓存、缓冲区                     │                 │
│  │  • 磁盘: 空间、IO 速率、IOPS、利用率                  │                 │
│  │  • 网络: 每接口的字节数/包数（入/出）                  │                 │
│  └──────────────────────────┬──────────────────────────┘                 │
│                             │                                            │
│                             ▼                                            │
│                    POST /metrics/push                                    │
│                             │                                            │
│                             ▼                                            │
│                    ┌─────────────────┐                                   │
│                    │  Agent (同节点)  │                                   │
│                    └─────────────────┘                                   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 3. Master 数据流（3 条流）

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Master 数据流                                     │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [1. 集群快照 — 内存存储 (DataHub)]                                       │
│  ─────────────────────────────────────────────────────────────────────── │
│  Agent ──▶ AgentSDK ──▶ Processor ──▶ DataHub (内存)                     │
│                                            │                             │
│  用途: Pod/Node 实时查询                    ◀── Web API 查询              │
│  保留: 仅最新快照                                                         │
│                                                                          │
│  [2. 命令下发 — 消息队列]                                                 │
│  ─────────────────────────────────────────────────────────────────────── │
│  用户/AI ──▶ API ──▶ CommandBus ──▶ Agent 执行                           │
│                          │                                               │
│  用途: 远程操作          Agent ──▶ 结果 ──▶ CommandBus ──▶ API           │
│  保留: 临时                                                               │
│                                                                          │
│  [3. 持久化数据 — 数据库]                                                 │
│  ─────────────────────────────────────────────────────────────────────── │
│  Agent ──▶ Processor ──┬──▶ 事件 ──▶ DB (event_history)                  │
│                        ├──▶ SLO 指标 ──▶ DB (slo_* 表)                   │
│                        └──▶ 节点指标 ──▶ DB (node_metrics_history)       │
│                                    │                                     │
│  用途: 历史分析                    ◀── 趋势/SLO API 查询                  │
│  保留: 30-180 天（可配置）                                                │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 4. Web 前端数据流

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Web 前端流程                                      │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                   │
│  │   浏览器    │    │  Next.js    │    │   Master    │                   │
│  │             │───▶│  中间件     │───▶│   Gateway   │                   │
│  │             │◀───│  (代理)     │◀───│   (API)     │                   │
│  └─────────────┘    └─────────────┘    └─────────────┘                   │
│                                                                          │
│  • 认证: JWT token 存储在 localStorage                                   │
│  • API 代理: /api/v2/* → Master:8080 (运行时配置)                        │
│  • 状态: Zustand 全局状态管理                                             │
│  • 实时: 可配置间隔的轮询                                                 │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 界面截图

### 集群总览
实时展示集群健康状态、资源使用情况和最近告警。

![集群总览](img/overview.png)

### Pod 管理
跨命名空间列表、过滤和管理 Pod，展示详细状态。

![Pod 管理](img/cluster_pod.png)

### 告警面板
查看和分析集群告警，支持过滤和 AI 智能分析。

![告警面板](img/cluster_alert.png)

### 节点指标
详细的节点级指标与历史趋势图表。

![节点指标](img/system_metrics.png)

### SLO 监控
基于 Ingress 指标追踪服务水平目标。

![SLO 总览](img/workbench_slo_overview.png)

![SLO 详情](img/workbench_slo.png)

---

## 部署

### 环境要求

- Go 1.21+
- Node.js 18+
- Kubernetes 集群（用于 Agent 部署）
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

### Kubernetes 部署（Helm）

```bash
# 添加 Helm 仓库（如已发布）
helm repo add atlhyper https://charts.atlhyper.io

# 安装 Master
helm install atlhyper-master atlhyper/atlhyper \
  --set master.admin.username=admin \
  --set master.admin.password=<YOUR_PASSWORD> \
  --set master.jwt.secret=<YOUR_SECRET>

# 安装 Agent（每个集群）
helm install atlhyper-agent atlhyper/atlhyper-agent \
  --set agent.clusterId=production \
  --set agent.masterUrl=http://atlhyper-master:8081
```

### Kubernetes 部署（YAML）

部署顺序: **Master → Agent → Metrics → Web**

```bash
cd deploy/k8s

# 1. 创建命名空间和配置
kubectl apply -f atlhyper-config.yaml

# 2. 部署 Master
kubectl apply -f atlhyper-Master.yaml

# 3. 部署 Agent
kubectl apply -f atlhyper-agent.yaml

# 4. 部署 Metrics (DaemonSet)
kubectl apply -f atlhyper-metrics.yaml

# 5. 部署 Web
kubectl apply -f atlhyper-web.yaml

# 6. (可选) Traefik IngressRoute
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
| `MASTER_DB_TYPE` | 否 | `sqlite` | 数据库类型 |
| `MASTER_DB_DSN` | 否 | - | MySQL/PostgreSQL 连接串 |
| `MASTER_LOG_LEVEL` | 否 | `info` | 日志级别 |

#### Agent 配置

| 参数 | 必填 | 说明 |
|------|------|------|
| `--cluster-id` | 是 | 集群唯一标识 |
| `--master` | 是 | Master AgentSDK 地址 |

#### Metrics DaemonSet

Metrics 采集器自动部署为 DaemonSet 并上报到本地 Agent。通过 ConfigMap 配置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `METRICS_AGENT_URL` | `http://atlhyper-agent:8082` | Agent 指标端点 |
| `METRICS_PUSH_INTERVAL` | `15s` | 推送间隔 |

---

## 项目结构

```
atlhyper/
├── atlhyper_master_v2/       # Master（中央控制）
│   ├── gateway/              # HTTP API (Web + AgentSDK)
│   ├── datahub/              # 内存数据存储
│   ├── database/             # 持久化存储 (SQLite/MySQL)
│   ├── service/              # 业务逻辑 (SLO, 告警)
│   ├── ai/                   # AI 助手集成
│   └── config/               # 配置管理
│
├── atlhyper_agent_v2/        # Agent（集群代理）
│   ├── source/               # 数据源
│   │   ├── event/            # K8s 事件监听
│   │   ├── snapshot/         # 资源快照
│   │   └── metrics/          # 指标接收
│   ├── executor/             # 命令执行
│   ├── sdk/                  # K8s 操作
│   └── pusher/               # 数据推送调度
│
├── atlhyper_metrics_v2/      # 指标采集器 (DaemonSet)
│   ├── collector/            # CPU、内存、磁盘、网络
│   └── pusher/               # 推送到 Agent
│
├── atlhyper_web/             # Web 前端
│   ├── src/app/              # Next.js 页面
│   ├── src/components/       # React 组件
│   ├── src/api/              # API 客户端
│   └── src/i18n/             # 国际化
│
├── model_v2/                 # 共享数据模型
├── cmd/                      # 入口程序
└── deploy/                   # 部署配置
    ├── helm/                 # Helm charts
    └── k8s/                  # K8s manifests
```

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

# AtlHyper

Kubernetes 多集群监控与运维管理平台

## 项目简介

AtlHyper 是一个面向轻量级 Kubernetes 环境的监控与管理平台，采用 Master-Agent 架构，支持多集群统一管理、实时资源监控、异常事件检测、远程运维操作。

## 系统架构

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              AtlHyper Platform                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────┐     ┌─────────────────────────────────────────────────┐   │
│   │  Web UI     │────▶│              Master                              │   │
│   │  (Next.js)  │◀────│                                                  │   │
│   └─────────────┘     │  ┌─────────┐  ┌──────────┐  ┌───────────────┐   │   │
│                       │  │ Gateway │  │Repository│  │   Service     │   │   │
│                       │  │ (API)   │  │ (Store)  │  │ (Alert/Log)   │   │   │
│                       │  └─────────┘  └──────────┘  └───────────────┘   │   │
│                       └──────────────────┬──────────────────────────────┘   │
│                                          │                                   │
│            ┌─────────────────────────────┼─────────────────────────────┐     │
│            │                             │                             │     │
│            ▼                             ▼                             ▼     │
│   ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   │  Agent (集群A)  │         │  Agent (集群B)  │         │  Agent (集群N)  │
│   │                 │         │                 │         │                 │
│   │  source/        │         │  source/        │         │  source/        │
│   │  ├─ event/      │         │  ├─ event/      │         │  ├─ event/      │
│   │  ├─ snapshot/   │         │  ├─ snapshot/   │         │  ├─ snapshot/   │
│   │  └─ metrics/    │         │  └─ metrics/    │         │  └─ metrics/    │
│   │  pusher/        │         │  pusher/        │         │  pusher/        │
│   │  executor/      │         │  executor/      │         │  executor/      │
│   │  sdk/ ──────────┼────┐    │  sdk/           │         │  sdk/           │
│   └─────────────────┘    │    └─────────────────┘         └─────────────────┘
│                          │
│                          ▼
│                   ┌─────────────┐
│                   │ Kubernetes  │
│                   │   Cluster   │
│                   └─────────────┘
└─────────────────────────────────────────────────────────────────────────────┘
```

## 技术栈

| 组件        | 技术                               | 说明                       |
| ----------- | ---------------------------------- | -------------------------- |
| **Master**  | Go + Gin + SQLite                  | 中央管控、数据聚合、UI API |
| **Agent**   | Go + controller-runtime            | 集群数据采集、命令执行     |
| **Metrics** | Go                                 | 节点指标采集 (DaemonSet)   |
| **Web**     | Next.js 16 + TypeScript + Tailwind | 管理界面                   |

## 项目结构

```
atlhyper/
├── atlhyper_master/          # 主控端
│   ├── gateway/              # HTTP API (UI + Ingest + Control)
│   ├── repository/           # 数据存储 (SQLite + Memory)
│   ├── service/              # 业务服务 (告警、日志)
│   ├── store/                # 内存存储
│   └── config/               # 配置管理
│
├── atlhyper_agent/           # 集群代理
│   ├── source/               # 核心数据处理层
│   │   ├── event/            # 事件处理 (watcher/abnormal/datahub)
│   │   ├── snapshot/         # 资源快照 (Pod/Node/Service/...)
│   │   └── metrics/          # 外部指标 (receiver + store)
│   ├── pusher/               # 数据推送调度
│   ├── executor/             # 命令执行 (Control Loop + Dispatcher)
│   ├── gateway/              # 网络通信 (HTTP Client/Server)
│   ├── sdk/                  # K8s 操作抽象层
│   ├── bootstrap/            # 启动引导
│   └── config/               # 配置管理
│
├── atlhyper_metrics/         # 指标采集器 (DaemonSet)
│   ├── collect/              # 指标采集
│   └── push/                 # 推送到 Agent
│
├── atlhyper_web/             # Web 前端
│   ├── src/app/              # 页面路由
│   ├── src/components/       # UI 组件
│   ├── src/api/              # API 调用
│   ├── src/store/            # 状态管理 (Zustand)
│   └── src/i18n/             # 国际化 (中文/日语)
│
├── model/                    # 共享数据模型
│   ├── transport/            # Agent-Master 传输模型
│   ├── k8s/                  # K8s 资源模型
│   └── collect/              # 指标采集模型
│
├── cmd/                      # 入口程序
│   ├── atlhyper_master/
│   ├── atlhyper_agent/
│   └── atlhyper_metrics/
│
└── deploy/                   # 部署配置 (Helm/K8s manifests)
```

## Agent 数据流

```
┌──────────────────────────────────────────────────────────────────────────┐
│                            四条数据流                                     │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [事件流 - I型]                                                          │
│  watcher/ ──▶ abnormal/ ──▶ datahub/ ──▶ pusher/ ──▶ Master             │
│                                                                          │
│  [快照流 - I型]                                                          │
│  snapshot/ (SDK.ListXXX) ──▶ pusher/ ──▶ Master                         │
│                                                                          │
│  [指标流 - U型]                                                          │
│  外部插件 ──▶ gateway/ ──▶ metrics/receiver ──▶ store ──▶ pusher ──▶ Master
│                                                                          │
│  [命令流 - 独立]                                                         │
│  Master ──▶ gateway/ ──▶ executor/ ──▶ SDK ──▶ K8s ──▶ (回执) ──▶ Master │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

## 功能特性

### 资源监控

- **Pod**: 列表、详情、日志、重启
- **Node**: 列表、详情、隔离/解除隔离
- **Deployment**: 列表、副本调整、镜像更新
- **Service/Namespace/Ingress/ConfigMap**: 列表与详情

### 异常检测

- Pod: CrashLoopBackOff, OOMKilled, ImagePullBackOff
- Node: NotReady, MemoryPressure, DiskPressure
- Deployment: 副本数不匹配
- Service: 无后端 Pod

### 告警通知

- 邮件告警 (SMTP)
- Slack 告警 (Webhook)

### 用户权限

- 三级角色: Viewer / Operator / Admin
- JWT 认证
- 操作审计日志

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- Kubernetes 集群 (Agent 运行环境)

### 启动 Master

```bash
cd cmd/atlhyper_master
go run main.go
# 监听 :8080
```

### 启动 Agent (集群内)

```bash
cd cmd/atlhyper_agent
go run main.go --cluster-id=<CLUSTER_ID> --master=http://<MASTER_IP>:8080
# 监听 :8082
```

### 启动 Web

```bash
cd atlhyper_web
npm install && npm run dev
# 访问 http://localhost:3000
```

## API 接口

### UI API (前端调用)

| 方法 | 路径                      | 说明     |
| ---- | ------------------------- | -------- |
| POST | /uiapi/auth/login         | 用户登录 |
| POST | /uiapi/pod/overview       | Pod 列表 |
| POST | /uiapi/ops/pod/restart    | 重启 Pod |
| POST | /uiapi/ops/node/cordon    | 隔离节点 |
| POST | /uiapi/ops/workload/scale | 调整副本 |

### Ingest API (Agent 上报)

| 方法 | 路径                     | 说明          |
| ---- | ------------------------ | ------------- |
| POST | /ingest/events           | 上报事件      |
| POST | /ingest/podlist          | 上报 Pod 列表 |
| POST | /ingest/metrics/snapshot | 上报指标      |

### Control API (命令下发)

| 方法 | 路径              | 说明           |
| ---- | ----------------- | -------------- |
| GET  | /ingest/ops/watch | 长轮询获取命令 |
| POST | /ingest/ops/ack   | 命令执行回执   |

## 配置文件

### Master (config.yaml)

```yaml
server:
  port: 8080

jwt:
  secret_key: "your-secret-key"
  token_expiry: 24h
```

### Agent (config.yaml)

```yaml
cluster:
  id: "cluster-1"
  master_url: "http://master:8080"

server:
  port: 8082
```

## License

MIT

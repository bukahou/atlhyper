# AtlHyper Agent 架构设计文档

## 一、Agent 整体架构

### 1.1 架构全景图

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│                                  AtlHyper Agent                                       │
│                                                                                      │
│  ┌────────────────────────────────────────────────────────────────────────────────┐  │
│  │                               SDK (K8s 抽象层)                                  │  │
│  │    CoreClient | RuntimeClient | MetricsClient | Operators | ResourceLister     │  │
│  └────────────────────────────────────────────────────────────────────────────────┘  │
│                                         ▲                                            │
│  ═══════════════════════════════════════╪════════════════════════════════════════   │
│                                         │                                            │
│  ┌──────────────────────────────────────┴─────────────────────────────────────────┐  │
│  │                                                                                │  │
│  │                          Source (核心数据处理层)                                │  │
│  │                                                                                │  │
│  │  ┌──────────────────────────────────────────────────────────────────────────┐  │  │
│  │  │                          event/ (事件处理模块)                            │  │  │
│  │  │                                                                          │  │  │
│  │  │   ┌─────────────┐    ┌─────────────┐    ┌─────────────────────────────┐  │  │  │
│  │  │   │  watcher/   │    │  abnormal/  │    │         datahub/            │  │  │  │
│  │  │   │             │    │             │    │                             │  │  │  │
│  │  │   │ Pod/Node    │───▶│ 异常检测    │───▶│  Cleaner (清洗逻辑)         │  │  │  │
│  │  │   │ Svc/Deploy  │    │ 规则库      │    │         │                   │  │  │  │
│  │  │   │ Event/Endpt │    │             │    │         ▼                   │  │  │  │
│  │  │   │             │    │             │    │  Pool (eventPool/cleaned)   │  │  │  │
│  │  │   └─────────────┘    └─────────────┘    └──────────────┬──────────────┘  │  │  │
│  │  │                                                        │                 │  │  │
│  │  └────────────────────────────────────────────────────────┼─────────────────┘  │  │
│  │                                                           │                    │  │
│  │  ┌────────────────────────────────────┐                   │                    │  │
│  │  │         snapshot/ (资源快照)       │                   │                    │  │
│  │  │                                    │                   │                    │  │
│  │  │  Pod/Node/Svc/Deploy/NS/Ing/CM     │                   │                    │  │
│  │  │  SDK.ListXXX() (实时拉取,无存储)   │                   │                    │  │
│  │  └──────────────────┬─────────────────┘                   │                    │  │
│  │                     │                                     │                    │  │
│  │  ┌──────────────────┼─────────────────┐                   │                    │  │
│  │  │         metrics/ (外部指标)        │                   │                    │  │
│  │  │                  │                 │                   │                    │  │
│  │  │  Receiver ◀──────┼─────────────────┼───────────────────┼────────────────────┼──┼── U型入口
│  │  │      │           │                 │                   │                    │  │
│  │  │      ▼           │                 │                   │                    │  │
│  │  │  Store (指标池)  │                 │                   │                    │  │
│  │  └──────────────────┼─────────────────┘                   │                    │  │
│  │                     │                                     │                    │  │
│  └─────────────────────┼─────────────────────────────────────┼────────────────────┘  │
│                        │                                     │                       │
│  ══════════════════════╪═════════════════════════════════════╪═══════════════════   │
│                        │                                     │                       │
│                        │         非核心业务逻辑层             │                       │
│                        │                                     │                       │
│                        ▼                                     ▼                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              pusher/ (数据推送)                                  │ │
│  │                                                                                 │ │
│  │   EventsPusher(5s)  |  SnapshotPushers(25-55s)  |  MetricsPusher(5s)            │ │
│  └───────────────────────────────────────┬─────────────────────────────────────────┘ │
│                                          │                                           │
│                                          ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              gateway/ (网络通信)                                 │ │
│  │                                                                                 │ │
│  │   HTTP Client (出站)                    HTTP Server (入站) :8082                │ │
│  │   /events, /podlist, /metrics ──▶ Master    /metrics/v1 ◀── 外部插件 (U型入口)  │ │
│  │   /ops/watch ◀── Master                     /ops/ack ──▶ Master                 │ │
│  └───────────────────────────────────────┬─────────────────────────────────────────┘ │
│                                          │                                           │
│  ════════════════════════════════════════╪═══════════════════════════════════════   │
│                                          │                                           │
│                                          │ (独立流程)                                │
│                                          ▼                                           │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐ │
│  │                          executor/ (命令执行 - 独立流程)                         │ │
│  │                                                                                 │ │
│  │   Gateway ◀─── Master (命令下发)                                                │ │
│  │      │                                                                          │ │
│  │      ▼                                                                          │ │
│  │   Control Loop (长轮询) ───▶ Dispatcher ───▶ SDK ───▶ K8s                       │ │
│  │      │                                                                          │ │
│  │      ▼                                                                          │ │
│  │   Gateway ───▶ Master (结果回执)                                                │ │
│  │                                                                                 │ │
│  │   支持命令: PodRestart | NodeCordon | ScaleWorkload | UpdateImage | PodGetLogs  │ │
│  └─────────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐ │
│  │                              bootstrap/ (启动引导)                               │ │
│  └─────────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 四条数据流

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                                  四条流程                                            │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│  【事件流】 source/event/ 内部完成                                                   │
│                                                                                     │
│   watcher/ ──▶ abnormal/ ──▶ datahub/ ──▶ pusher/ ──▶ gateway/ ──▶ Master          │
│                                                                                     │
│  ──────────────────────────────────────────────────────────────────────────────────│
│                                                                                     │
│  【快照流】                                                                          │
│                                                                                     │
│   source/snapshot/ ──▶ pusher/ ──▶ gateway/ ──▶ Master                             │
│                                                                                     │
│  ──────────────────────────────────────────────────────────────────────────────────│
│                                                                                     │
│  【外部指标流 - U型】                                                                │
│                                                                                     │
│   外部插件 ──▶ gateway/ ──▶ source/metrics/ ──▶ pusher/ ──▶ gateway/ ──▶ Master    │
│                                                                                     │
│  ──────────────────────────────────────────────────────────────────────────────────│
│                                                                                     │
│  【命令执行流 - 独立】                                                               │
│                                                                                     │
│   Master ──▶ gateway/ ──▶ executor/ ──▶ SDK ──▶ K8s                                │
│                              │                                                      │
│                              ▼                                                      │
│   Master ◀── gateway/ ◀── (结果回执)                                               │
│                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 二、模块详细说明

### 2.1 Source 层 (核心数据处理)

#### 2.1.1 event/ (事件处理模块)

watcher/ + abnormal/ + datahub/ 三者嵌套，构成完整的事件处理链路。

| 子模块 | 职责 | 输入 | 输出 |
|--------|------|------|------|
| **watcher/** | 监听 K8s 资源变更 | K8s Watch API | 资源对象 |
| **abnormal/** | 异常检测规则库 | 资源对象 | 异常原因 |
| **datahub/** | 事件清洗与存储 | 异常事件 | cleanedEventPool |

**Watcher 监听的资源与异常检测:**

| Watcher | 资源类型 | 检测的异常状态 | 严重级别 |
|---------|----------|---------------|---------|
| PodWatcher | corev1.Pod | CrashLoopBackOff, OOMKilled, ImagePullBackOff | critical/warning |
| NodeWatcher | corev1.Node | NotReady, MemoryPressure, DiskPressure | critical/warning |
| DeploymentWatcher | appsv1.Deployment | Replicas != ReadyReplicas | warning |
| ServiceWatcher | corev1.Service | 无后端 Pod | warning |
| EventWatcher | corev1.Event | Type="Warning" | warning |
| EndpointWatcher | discoveryv1.EndpointSlice | Endpoints 为空 | warning |

**DataHub 清洗配置:**

| 配置项 | 默认值 | 说明 |
|--------|-------|------|
| RetentionRawDuration | 10min | 原始事件保留时间 |
| RetentionCleanedDuration | 30min | 清洗池保留时间 |
| 去重键 | Kind\|NS\|Name\|Reason | 聚合标识 |

#### 2.1.2 snapshot/ (资源快照模块)

实时从 K8s API 拉取资源列表，**无存储**。

| 资源 | 函数 | 附加信息 |
|------|------|---------|
| Pod | ListPods() | 包含 CPU/Memory 指标 |
| Node | ListNodes() | - |
| Service | ListServices() | - |
| Deployment | ListDeployments() | - |
| Namespace | ListNamespaces() | - |
| Ingress | ListIngresses() | - |
| ConfigMap | ListConfigMaps() | - |

#### 2.1.3 metrics/ (外部指标模块)

U型数据流：Gateway 入 → Receiver → Store → Pusher → Gateway 出

| 组件 | 职责 |
|------|------|
| Receiver | 接收外部插件推送的指标 |
| Store | 指标数据池 (metricsStore) |

---

### 2.2 Pusher 层 (数据推送)

| Pusher | 间隔 | 数据源 | API 路径 |
|--------|------|--------|----------|
| Events | 5s | datahub/cleanedPool | /events/v1/eventlog |
| Metrics | 5s | metrics/store | /metrics/snapshot |
| Pod | 25s | snapshot/pod | /podlist |
| Node | 30s | snapshot/node | /nodelist |
| Service | 35s | snapshot/service | /servicelist |
| Namespace | 40s | snapshot/namespace | /namespacelist |
| Ingress | 45s | snapshot/ingress | /ingresslist |
| Deployment | 50s | snapshot/deployment | /deploymentlist |
| ConfigMap | 55s | snapshot/configmap | /configmaplist |

---

### 2.3 Gateway 层 (网络通信)

| 方向 | 端点 | 用途 |
|------|------|------|
| 出站 | POST /events/v1/eventlog | 推送事件 |
| 出站 | POST /podlist, /nodelist... | 推送快照 |
| 出站 | POST /metrics/snapshot | 推送指标 |
| 出站 | POST /ops/ack | 命令回执 |
| 入站 | POST /metrics/v1/snapshot | 接收外部指标 (U型入口) |
| 入站 | POST /ops/watch | 接收命令 (长轮询) |

---

### 2.4 Executor 层 (命令执行 - 独立流程)

与 Source 数据流完全无关，是 Agent 接收并执行 Master 下发命令的独立通道。

**流程:**
```
Master ──▶ Gateway ──▶ Control Loop ──▶ Dispatcher ──▶ SDK ──▶ K8s
                              │
                              ▼
Master ◀── Gateway ◀── (结果回执)
```

**支持的命令:**

| 命令类型 | Target 参数 | Args 参数 | SDK 调用 |
|---------|------------|----------|---------|
| PodRestart | ns, pod | - | Pods().RestartPod() |
| NodeCordon | node | - | Nodes().CordonNode() |
| NodeUncordon | node | - | Nodes().UncordonNode() |
| ScaleWorkload | ns, name, kind | replicas | Deployments().ScaleDeployment() |
| UpdateImage | ns, name, kind | newImage | Deployments().UpdateDeploymentImage() |
| PodGetLogs | ns, pod | container, tailLines | Pods().GetPodLogs() |

---

## 三、目录结构

### 3.1 当前目录结构

```
atlhyper_agent/
│
├── sdk/                              # SDK 层 (K8s 抽象)
│   ├── interfaces.go
│   ├── models.go
│   ├── provider.go
│   ├── registry.go
│   ├── global.go
│   └── k8s/
│       ├── provider.go
│       ├── lister.go
│       ├── operators.go
│       ├── metrics.go
│       └── cluster.go
│
├── source/                           # 核心数据处理层
│   │
│   ├── event/                        # 事件处理模块 (嵌套)
│   │   │
│   │   ├── watcher/                  # 监听
│   │   │   ├── register.go
│   │   │   ├── pod/
│   │   │   ├── node/
│   │   │   ├── service/
│   │   │   ├── deployment/
│   │   │   ├── event/
│   │   │   └── endpoint/
│   │   │
│   │   ├── abnormal/                 # 异常检测
│   │   │   ├── pod_abnormal.go
│   │   │   ├── node_abnormal.go
│   │   │   ├── deployment_abnormal.go
│   │   │   ├── service_abnormal.go
│   │   │   ├── event_abnormal.go
│   │   │   └── endpoint_abnormal.go
│   │   │
│   │   └── datahub/                  # 事件数据中心
│   │       ├── cleaner.go            # 清洗逻辑
│   │       └── collector.go          # 收集逻辑
│   │
│   ├── snapshot/                     # 资源快照模块
│   │   ├── pod/
│   │   ├── node/
│   │   ├── service/
│   │   ├── deployment/
│   │   ├── namespace/
│   │   ├── ingress/
│   │   └── configmap/
│   │
│   └── metrics/                      # 外部指标模块
│       ├── metrics_receiver.go       # 接收处理
│       └── store.go                  # 指标数据池
│
├── pusher/                           # 数据推送 (非核心)
│   ├── bootstrap.go
│   ├── generic.go
│   ├── sources.go
│   └── clusterid.go
│
├── gateway/                          # 网络通信 (非核心)
│   ├── http_client.go
│   ├── http_server.go
│   └── routes.go
│
├── executor/                         # 命令执行 (独立流程)
│   ├── control.go                    # Control Loop
│   ├── dispatcher.go                 # 命令分发
│   └── handlers/                     # 命令处理器
│
├── bootstrap/                        # 启动引导
│   ├── manager.go
│   └── health.go
│
├── config/                           # 配置
│
├── model/                            # 数据模型
│
└── main.go
```

### 3.2 重构变更记录

| 原位置 | 现位置 | 变更说明 |
|---------|---------|---------|
| source/watcher/ | source/event/watcher/ | 嵌套到 event/ 下 |
| source/abnormal/ | source/event/abnormal/ | 嵌套到 event/ 下 |
| logic/cleaner/ | source/event/datahub/ | 重命名并移动 |
| source/snapshot/ | source/snapshot/ | 保持不变 |
| source/receiver/ | source/metrics/ | 重命名 |
| store/ | source/metrics/store.go | 合并到 metrics 模块 |
| logic/pusher/ | pusher/ | 提升为顶层目录 |
| gateway/ | gateway/ | 保持不变 |
| logic/control/ + logic/executor/ | executor/ | 合并为独立模块 |
| bootstrap/ | bootstrap/ | 保持不变 |

---

## 四、模块分类

### 4.1 核心层 vs 非核心层

| 层级 | 模块 | 职责 | 属性 |
|------|------|------|------|
| **source/event/** | watcher/ | 监听 K8s 资源变更 | 核心 |
| | abnormal/ | 异常检测规则 | 核心 |
| | datahub/ | 事件清洗+存储 | 核心 |
| **source/snapshot/** | - | 资源快照拉取 | 核心 |
| **source/metrics/** | - | 外部指标接收+存储 | 核心 |
| **pusher/** | - | 数据推送调度 | 非核心 |
| **gateway/** | - | 网络通信 | 非核心 |
| **executor/** | - | 命令执行 (独立流程) | 独立 |
| **bootstrap/** | - | 启动引导 | 非核心 |

### 4.2 数据存储分布

| 数据类型 | 存储位置 | 存储方式 |
|---------|---------|---------|
| 异常事件 | source/event/datahub/pool.go | 包内变量 (eventPool/cleanedEventPool) |
| 资源快照 | 无存储 | 实时从 SDK 拉取 |
| 外部指标 | source/metrics/store.go | 内存 Store |

---

## 五、SDK 重构完成记录

### 5.1 已完成

- [x] 创建 `sdk/interfaces.go` - 核心接口定义
- [x] 创建 `sdk/models.go` - 数据模型
- [x] 创建 `sdk/provider.go` - SDKProvider 聚合接口
- [x] 创建 `sdk/registry.go` - 注册机制
- [x] 创建 `sdk/global.go` - 全局单例
- [x] 创建 `sdk/k8s/provider.go` - K8sProvider 主体
- [x] 创建 `sdk/k8s/lister.go` - ResourceLister 实现
- [x] 创建 `sdk/k8s/operators.go` - Pod/Node/Deployment 操作
- [x] 创建 `sdk/k8s/metrics.go` - MetricsProvider 实现
- [x] 创建 `sdk/k8s/cluster.go` - ClusterInfo 实现
- [x] 创建 `sdk/k8s/init.go` - 自动注册
- [x] 迁移所有调用点
- [x] 删除 `k8sclient/` 目录
- [x] 编译验证通过

### 5.2 SDK 使用方式

```go
import "AtlHyper/atlhyper_agent/sdk"
import _ "AtlHyper/atlhyper_agent/sdk/k8s"  // 自动注册

// 初始化
sdk.Init("kubernetes", sdk.ProviderConfig{Kubeconfig: "..."})

// 资源操作
sdk.Get().Pods().RestartPod(ctx, sdk.ObjectKey{Namespace: "ns", Name: "pod"})
sdk.Get().Nodes().CordonNode(ctx, "node-1")
sdk.Get().ListPods(ctx, "namespace")

// 底层客户端访问
sdk.Get().CoreClient()       // 原生 K8s 操作
sdk.Get().RuntimeClient()    // Watcher 使用
sdk.Get().MetricsClient()    // Metrics 操作
```

---

## 六、目录重构完成记录

### 6.1 重构步骤（已完成）

1. [x] 创建 `source/event/` 目录
2. [x] 移动 `source/watcher/` → `source/event/watcher/`
3. [x] 移动 `source/abnormal/` → `source/event/abnormal/`
4. [x] 创建 `source/event/datahub/` 并迁移 `logic/cleaner/`
5. [x] 重命名 `source/receiver/` → `source/metrics/`
6. [x] 迁移 `store/` 中 metrics 相关代码到 `source/metrics/store.go`
7. [x] 移动 `logic/pusher/` → `pusher/`
8. [x] 合并 `logic/control/` + `logic/executor/` → `executor/`
9. [x] 清理空目录和旧代码
10. [x] 更新所有 import 路径
11. [x] 编译验证通过

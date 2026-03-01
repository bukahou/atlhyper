# K8s 元数据展示增强（Phase 0）

> 状态：活跃
> 创建：2026-02-27
> 前置：OTel Operator 部署（已完成）
> 后续：[跨信号穿透式导航](../future/cross-signal-navigation-design.md)（Phase 1+）

---

## 1. 背景

OTel Operator + k8sattributes processor 已部署（2026-02-27），ClickHouse 中 Traces 和 Logs 的 `ResourceAttributes` 现在携带完整的 K8s 元数据：

| ResourceAttribute | otel_traces | otel_logs | 来源 |
|--------------------|:-:|:-:|------|
| `k8s.pod.name` | ✅ | ✅ | Operator Downward API |
| `k8s.node.name` | ✅ | ✅ | Operator Downward API |
| `k8s.deployment.name` | ✅ | ✅ | Operator Downward API |
| `k8s.namespace.name` | ✅ | ✅ | Operator Downward API |
| `k8s.pod.uid` | ✅ | ✅ | Operator Downward API |
| `k8s.replicaset.name` | ✅ | ✅ | Operator Downward API |

**问题**：数据已在 ClickHouse 中，但后端模型和前端类型未更新，用户看不到这些信息。

---

## 2. 代码层缺失

### 2.1 后端 — SpanResource

**文件**: `model_v3/apm/trace.go`

```go
// 当前：只提取 4 个字段
type SpanResource struct {
    ServiceVersion string `json:"serviceVersion,omitempty"`
    InstanceId     string `json:"instanceId,omitempty"`
    PodName        string `json:"podName,omitempty"`
    ClusterName    string `json:"clusterName,omitempty"`
    // 缺失: NodeName, DeploymentName, NamespaceName
}
```

### 2.2 后端 — Log Entry

**文件**: `model_v3/log/log.go`

```go
// Resource 是原始 map，已包含所有 K8s 字段，无需改模型
type Entry struct {
    // ...
    Resource map[string]string `json:"resource"`
    // 前端可直接通过 resource["k8s.pod.name"] 访问
}
```

### 2.3 前端 — SpanResource 类型

**文件**: `atlhyper_web/src/types/model/apm.ts`

```typescript
// 当前：缺失 nodeName, deploymentName, namespaceName
export interface SpanResource {
    serviceVersion?: string;
    instanceId?: string;
    podName?: string;
    clusterName?: string;
}
```

---

## 3. 设计

> **原则**：数据已在 ClickHouse 中，只需要在模型层提取 + 前端展示。
> APM 和 Logs 两侧独立修改，无交叉依赖。

### 3.1 后端改动

#### 3.1.1 SpanResource 增加 K8s 字段

**文件**: `model_v3/apm/trace.go`

```go
type SpanResource struct {
    ServiceVersion string `json:"serviceVersion,omitempty"`
    InstanceId     string `json:"instanceId,omitempty"`
    PodName        string `json:"podName,omitempty"`
    NodeName       string `json:"nodeName,omitempty"`       // 新增
    DeploymentName string `json:"deploymentName,omitempty"` // 新增
    NamespaceName  string `json:"namespaceName,omitempty"`  // 新增
    ClusterName    string `json:"clusterName,omitempty"`
}
```

#### 3.1.2 trace.go 提取新字段

**文件**: `atlhyper_agent_v2/repository/ch/query/trace.go`（约 190 行）

```go
s.Resource = apm.SpanResource{
    ServiceVersion: resAttrs["service.version"],
    InstanceId:     resAttrs["service.instance.id"],
    PodName:        resAttrs["k8s.pod.name"],
    NodeName:       resAttrs["k8s.node.name"],           // 新增
    DeploymentName: resAttrs["k8s.deployment.name"],     // 新增
    NamespaceName:  resAttrs["k8s.namespace.name"],      // 新增
    ClusterName:    resAttrs["k8s.cluster.name"],
}
```

> Log Entry 的 `Resource map[string]string` 已包含所有 K8s 字段（原始 map），无需改后端模型。

### 3.2 前端改动

#### 3.2.1 SpanResource 类型更新

**文件**: `atlhyper_web/src/types/model/apm.ts`

```typescript
export interface SpanResource {
    serviceVersion?: string;
    instanceId?: string;
    podName?: string;
    nodeName?: string;           // 新增
    deploymentName?: string;     // 新增
    namespaceName?: string;      // 新增
    clusterName?: string;
}
```

#### 3.2.2 APM SpanDrawer 展示 K8s 元数据

**文件**: `atlhyper_web/src/app/observe/apm/components/TraceWaterfall.tsx`

在 SpanDrawer 的 Metadata 区域增加 K8s 上下文信息：

```
┌──────────────────────────────────────────┐
│ Span: GET /api/media/bangumi            │
│ Service: geass-gateway                   │
│                                          │
│ K8s Context                              │
│ ┌──────────────────────────────────────┐ │
│ │ Pod:        geass-gateway-77964-5zr  │ │
│ │ Node:       desk-one                 │ │
│ │ Deployment: geass-gateway            │ │
│ │ Namespace:  geass                    │ │
│ └──────────────────────────────────────┘ │
│                                          │
│ [Timeline] [Metadata] [Logs]             │
└──────────────────────────────────────────┘
```

#### 3.2.3 Logs LogDetail 展示 K8s 元数据

**文件**: `atlhyper_web/src/app/observe/logs/components/LogDetail.tsx`

从 `entry.resource` map 中提取 K8s 信息展示：

```typescript
const podName = entry.resource["k8s.pod.name"];
const nodeName = entry.resource["k8s.node.name"];
const deploymentName = entry.resource["k8s.deployment.name"];
const namespaceName = entry.resource["k8s.namespace.name"];
```

#### 3.2.4 i18n

| 键 | 中文 | 日文 |
|----|------|------|
| `common.k8sContext` | "K8s 上下文" | "K8s コンテキスト" |
| `common.podName` | "Pod" | "Pod" |
| `common.nodeName` | "节点" | "ノード" |
| `common.deploymentName` | "Deployment" | "Deployment" |
| `common.namespaceName` | "命名空间" | "ネームスペース" |

---

## 4. 文件变更清单

| 文件 | 操作 | 改动 |
|------|------|------|
| `model_v3/apm/trace.go` | 修改 | SpanResource 增加 3 个字段 |
| `atlhyper_agent_v2/repository/ch/query/trace.go` | 修改 | 提取 3 个新 ResourceAttributes |
| `atlhyper_web/src/types/model/apm.ts` | 修改 | SpanResource 增加 3 个可选字段 |
| `atlhyper_web/src/app/observe/apm/components/TraceWaterfall.tsx` | 修改 | SpanDrawer 展示 K8s Context |
| `atlhyper_web/src/app/observe/logs/components/LogDetail.tsx` | 修改 | LogDetail 展示 K8s 元数据 |
| `atlhyper_web/src/types/i18n.ts` | 修改 | 增加 K8s 上下文翻译键 |
| `atlhyper_web/src/i18n/locales/zh.ts` | 修改 | 中文翻译 |
| `atlhyper_web/src/i18n/locales/ja.ts` | 修改 | 日文翻译 |
| **合计** | **8 个文件** | |

---

## 5. 验证

```bash
# 后端构建
go build ./model_v3/... && go build ./atlhyper_agent_v2/...

# 前端构建
cd atlhyper_web && npm run build
```

端到端验证：
1. APM 页面 → 选择 Trace → 点击 Span → SpanDrawer 应显示 Pod/Node/Deployment/Namespace
2. Logs 页面 → 展开日志详情 → 应显示 K8s 元数据

# OTel Operator 部署指南

OTel Operator 实现 Java 应用自动注入 OpenTelemetry Agent，并通过 k8sattributes processor 为所有遥测数据补充 K8s 元数据，支撑跨信号关联（Traces ↔ Logs ↔ K8s）。

---

## 架构概览

```
                    ┌─────────────────────────────────────────────────────┐
                    │                  OTel Operator                      │
                    │           (Webhook 自动注入 Java Agent)              │
                    └──────────────┬──────────────────────────────────────┘
                                   │ 注入
                    ┌──────────────▼──────────────────────────────────────┐
                    │              Java 应用 Pod                          │
                    │  ┌────────────────┐  ┌───────────────────────────┐  │
                    │  │  init container │  │    业务容器                │  │
                    │  │  (Agent JAR)    │──│  JAVA_TOOL_OPTIONS        │  │
                    │  └────────────────┘  │  + K8s 元数据 (Downward)   │  │
                    │                      └────────────┬──────────────┘  │
                    └───────────────────────────────────┼─────────────────┘
                                                        │ OTLP (gRPC)
                    ┌───────────────────────────────────▼─────────────────┐
                    │              OTel Collector                          │
                    │  ┌──────────────┐  ┌──────────┐  ┌──────────────┐  │
                    │  │ k8sattributes│→ │ resource  │→ │  clickhouse  │  │
                    │  │  (K8s API)   │  │(cluster)  │  │  (exporter)  │  │
                    │  └──────────────┘  └──────────┘  └──────────────┘  │
                    └─────────────────────────────────────────────────────┘
```

**两层 K8s 元数据注入：**

| 层 | 机制 | 适用对象 |
|----|------|---------|
| Operator 自动注入 | Downward API → ResourceAttributes | Java 应用（通过 annotation 触发） |
| k8sattributes processor | Collector 查询 K8s API 补充 | 所有遥测数据（Go 手动埋点的兜底） |

---

## 前置条件

| 组件 | 要求 | 验证命令 |
|------|------|---------|
| cert-manager | 已安装 | `kubectl get pods -n cert-manager` |
| OTel Collector | 已运行 | `kubectl get pods -n atlhyper -l app=otel-collector` |
| ServiceAccount | `otel-collector` 存在 | `kubectl get sa otel-collector -n atlhyper` |

---

## 部署步骤

### Step 1: 安装 OTel Operator

```bash
kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
```

验证：

```bash
kubectl wait --for=condition=available \
  deployment/opentelemetry-operator-controller-manager \
  -n opentelemetry-operator-system --timeout=120s
```

### Step 2: 创建 RBAC（集群级资源）

k8sattributes processor 需要读取 K8s API 获取 Pod/Node/Deployment 信息。

```bash
kubectl apply -f atlhyper-otel-rbac.yaml
```

文件内容（`atlhyper-otel-rbac.yaml`）：

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: otel-collector
rules:
  - apiGroups: [""]
    resources: ["pods", "namespaces", "nodes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: otel-collector
subjects:
  - kind: ServiceAccount
    name: otel-collector
    namespace: atlhyper
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: otel-collector
```

> **注意**：此文件为集群级资源，不可加入 kustomization（会被强制覆盖 namespace），需单独 apply。

### Step 3: 部署 Instrumentation CR + 更新 Collector 配置

```bash
kubectl apply -k atlhyper/
```

此步骤包含：

**Instrumentation CR**（`atlhyper-otel-instrumentation.yaml`）：

```yaml
apiVersion: opentelemetry.io/v1alpha1
kind: Instrumentation
metadata:
  name: atlhyper-instrumentation
  namespace: atlhyper
spec:
  java:
    image: ghcr.io/open-telemetry/opentelemetry-operator/autoinstrumentation-java:2.14.0
  exporter:
    endpoint: http://otel-collector.atlhyper.svc:4317
  propagators:
    - tracecontext
    - baggage
  resource:
    addK8sUIDAttributes: true
  env:
    - name: OTEL_EXPORTER_OTLP_PROTOCOL
      value: "grpc"
    - name: OTEL_METRICS_EXPORTER
      value: "none"
    - name: OTEL_LOGS_EXPORTER
      value: "otlp"
```

**Collector 配置更新**（k8sattributes processor 加入 traces 和 logs pipeline）：

```yaml
processors:
  k8sattributes:
    auth_type: "serviceAccount"
    extract:
      metadata:
        - k8s.pod.name
        - k8s.pod.uid
        - k8s.deployment.name
        - k8s.namespace.name
        - k8s.node.name
        - k8s.pod.start_time
    pod_association:
      - sources:
          - from: resource_attribute
            name: k8s.pod.name
      - sources:
          - from: resource_attribute
            name: k8s.pod.ip
      - sources:
          - from: connection

pipelines:
  traces:
    processors: [memory_limiter, k8sattributes, resource, filter/traces-health]
  logs:
    processors: [memory_limiter, k8sattributes, resource]
  metrics:
    # 不变，node_exporter 自带节点名
```

部署后重启 Collector 加载新配置：

```bash
kubectl rollout restart deployment/otel-collector -n atlhyper
```

### Step 4: 迁移 Java 应用到 Operator 自动注入

以 Geass 微服务为例，每个 Deployment 需要：

**添加 annotation**（指定 Instrumentation CR + 目标容器）：

```yaml
spec:
  template:
    metadata:
      annotations:
        instrumentation.opentelemetry.io/inject-java: "atlhyper/atlhyper-instrumentation"
        instrumentation.opentelemetry.io/container-names: "geass-gateway"
```

**删除手动注入的全部内容**：

| 删除项 | 说明 |
|--------|------|
| `initContainers` | elastic-apm-init、otel-agent-init |
| `env` 中的 `JAVA_OPTS` | 手动 `-javaagent:...` |
| `env` 中的 `OTEL_SERVICE_NAME` | Operator 自动从 deployment name 派生 |
| `volumeMounts` | apm-agent、otel-agent |
| `volumes` | apm-agent、otel-agent emptyDir |

**删除 ConfigMap 中的 OTel 配置**（改由 Instrumentation CR 管理）：

```
删除: ELASTIC_APM_SERVER_URL, ELASTIC_APM_ENVIRONMENT, ELASTIC_APM_USE_ELASTIC_EXCEPTIONS
删除: OTEL_EXPORTER_OTLP_ENDPOINT, OTEL_EXPORTER_OTLP_PROTOCOL, OTEL_RESOURCE_ATTRIBUTES
删除: OTEL_METRICS_EXPORTER, OTEL_LOGS_EXPORTER
```

**保留**：`envFrom`（业务配置）、`resources`、`probes`。

部署：

```bash
kubectl apply -k Geass/
```

---

## Linkerd 兼容性

### 问题：Operator 默认注入到 Linkerd sidecar

使用 Linkerd 服务网格时，`linkerd-proxy` sidecar 会被注入为 Pod 的第一个容器。OTel Operator 默认会向所有容器注入 Java Agent，导致 Agent 被注入到 `linkerd-proxy`（Rust 二进制）而非业务容器。

### 解决：container-names annotation

必须在每个 Deployment 中显式指定目标容器：

```yaml
annotations:
  instrumentation.opentelemetry.io/inject-java: "atlhyper/atlhyper-instrumentation"
  instrumentation.opentelemetry.io/container-names: "<业务容器名>"
```

不添加 `container-names` 时，业务容器不会被注入 OTel Agent，遥测数据将完全丢失。

---

## 验证

### 1. Operator 状态

```bash
kubectl get pods -n opentelemetry-operator-system
kubectl get instrumentation -n atlhyper
```

### 2. Collector k8sattributes 初始化

```bash
kubectl logs -n atlhyper deploy/otel-collector -c otel-collector | head -20
# 应看到: k8s filtering {"kind": "processor", "name": "k8sattributes", "pipeline": "traces/logs"}
```

### 3. Java Agent 自动注入

```bash
# 检查 init container
kubectl get pod -n geass -l app=geass-gateway \
  -o jsonpath='{.items[0].spec.initContainers[*].name}'
# 应包含: opentelemetry-auto-instrumentation-java

# 检查 JAVA_TOOL_OPTIONS（确认注入到业务容器）
kubectl get pod -n geass -l app=geass-gateway \
  -o jsonpath='{.items[0].spec.containers[?(@.name=="geass-gateway")].env[?(@.name=="JAVA_TOOL_OPTIONS")].value}'
# 应输出: -javaagent:/otel-auto-instrumentation-java-geass-gateway/javaagent.jar

# 检查 linkerd-proxy 未被注入
kubectl get pod -n geass -l app=geass-gateway \
  -o jsonpath='{.items[0].spec.containers[?(@.name=="linkerd-proxy")].env[?(@.name=="JAVA_TOOL_OPTIONS")].value}'
# 应无输出

# 应用日志确认 Agent 加载
kubectl logs -n geass deploy/geass-gateway -c geass-gateway | head -3
# 应看到: Picked up JAVA_TOOL_OPTIONS: -javaagent:...
# 应看到: opentelemetry-javaagent - version: 2.14.0
```

### 4. ClickHouse 数据验证

```sql
-- Traces 包含 K8s 元数据
SELECT ServiceName,
       ResourceAttributes['k8s.pod.name']        AS pod_name,
       ResourceAttributes['k8s.node.name']        AS node_name,
       ResourceAttributes['k8s.deployment.name']  AS deployment_name,
       ResourceAttributes['k8s.namespace.name']   AS namespace_name
FROM atlhyper.otel_traces
WHERE ServiceName LIKE 'geass-%'
ORDER BY Timestamp DESC
LIMIT 5;

-- Logs 同样验证
SELECT ServiceName,
       ResourceAttributes['k8s.pod.name']        AS pod_name,
       ResourceAttributes['k8s.node.name']        AS node_name
FROM atlhyper.otel_logs
WHERE ServiceName LIKE 'geass-%'
ORDER BY Timestamp DESC
LIMIT 5;
```

全部字段非空即为成功。

---

## 文件清单

| 文件 | 类型 | 部署方式 | 说明 |
|------|------|---------|------|
| `atlhyper-otel-instrumentation.yaml` | Instrumentation CR | kustomize | Java 自动注入定义 |
| `atlhyper-otel-rbac.yaml` | ClusterRole/Binding | 单独 apply | k8sattributes 的 K8s API 权限 |
| `atlhyper-otel-config.yaml` | ConfigMap | kustomize | Collector 配置（含 k8sattributes） |
| `kustomization.yaml` | Kustomization | - | 引用 instrumentation |

---

## 故障排查

| 现象 | 原因 | 解决 |
|------|------|------|
| Pod 无 init container | Operator webhook 未生效 | 检查 Operator 是否运行、annotation 是否正确 |
| Agent 注入到 linkerd-proxy | 缺少 container-names | 添加 `container-names` annotation |
| Collector 启动报 k8sattributes 权限错误 | RBAC 未创建 | `kubectl apply -f atlhyper-otel-rbac.yaml` |
| Traces 无 K8s 元数据 | k8sattributes 未加入 pipeline | 检查 Collector config 的 processors 顺序 |
| Instrumentation CR 创建失败 | cert-manager 未就绪 | `kubectl get pods -n cert-manager` |

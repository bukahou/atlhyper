# Agent 泛用型服务网格监控（Kustomize Overlay + OTel Transform + Agent 代码改造）

> 从 Linkerd 专用监控改造为泛用型（Linkerd/Istio 通用）。Agent 代码不感知底层网格类型，通过 OTel Collector 做 metric 重命名和 label 语义映射，Kustomize overlay 做网格切换。

---

## 1. 背景

- Agent 现有 mesh 查询代码（`atlhyper_agent_v2/repository/ch/query/slo.go` + `summary.go`）硬编码 Linkerd 专属：metric 名 `response_total`、label `deployment/tls/direction`、MV 名 `mv_linkerd_*`
- 集群当前**未装任何网格**（Linkerd/Istio 均无），mesh 查询代码实际处于"待使用"状态
- 需求：agent 改成泛用型，未来接 Istio/Linkerd/Cilium 等任何兼容 mesh 只需改 OTel 配置，agent 代码零改动

## 2. 方案：Kustomize Overlay + OTel Transform

### 2.1 整体架构

```
Istio/Linkerd sidecar (或 viz) ──► OTel Collector ──► ClickHouse ──► Agent ──► Master ──► Web
                                        ↑                            ↑
                                        │                            │
                         按 overlay 启用对应 scrape + transform      只查 mesh_request_total
                         （Linkerd/Istio 互斥，但配置对称）           （不感知底层网格）
```

### 2.2 Kustomize Overlay 设计（config 仓库）

路径：`clusters/zgmf-x10a/apps/atlhyper/`

```
apps/atlhyper/
├── kustomization.yaml                   # 顶层：选 base + 一个 mesh overlay
├── base/                                # 与网格无关的资源
│   ├── kustomization.yaml
│   ├── atlhyper-config.yaml
│   ├── atlhyper-otel-instrumentation.yaml
│   ├── atlhyper-otel-rbac.yaml
│   ├── atlhyper-clickhouse.yaml
│   ├── atlhyper-otel-collector.yaml     # 只含 Deployment/Service/RBAC
│   ├── atlhyper-node-exporter.yaml
│   ├── atlhyper-Master.yaml
│   ├── atlhyper-agent.yaml
│   ├── atlhyper-web.yaml
│   └── atlhyper-traefik.yaml
└── mesh/
    ├── none/
    │   ├── kustomization.yaml
    │   └── atlhyper-otel-config.yaml    # OTel ConfigMap（基础段，无 mesh scrape/transform）
    ├── linkerd/
    │   ├── kustomization.yaml
    │   ├── atlhyper-otel-config.yaml    # 基础段 + linkerd-prometheus scrape + metricstransform/linkerd_to_mesh
    │   └── atlhyper-otel-authz.yaml     # Linkerd Server/AuthorizationPolicy
    └── istio/
        ├── kustomization.yaml
        └── atlhyper-otel-config.yaml    # 基础段 + istio-sidecars scrape + transform/istio_to_mesh
```

**切换网格 = 改顶层 kustomization.yaml 一行**：`- mesh/istio` ↔ `- mesh/linkerd` ↔ `- mesh/none`

### 2.3 统一契约

#### Metric 名（ClickHouse 里的最终名字）
| 用途 | 统一名 | Linkerd 原名 | Istio 原名 |
|------|--------|------|------|
| 请求计数 | `mesh_request_total` | `response_total` | `istio_requests_total` |
| 延迟直方图 | `mesh_request_duration_ms_bucket` | `response_latency_ms_bucket` | `istio_request_duration_milliseconds_bucket` |

#### Label schema
| 统一 label | 含义 | Linkerd 来源 | Istio 来源（OTTL 条件） |
|-----------|------|-------------|----------------------|
| `namespace` | 工作负载 ns | `namespace`（原样） | reporter=destination → destination_workload_namespace<br>reporter=source → source_workload_namespace |
| `workload` | 工作负载名 | `deployment` → `workload` | reporter=destination → destination_workload<br>reporter=source → source_workload |
| `dst_namespace` | 目标 ns（edge 用） | `dst_namespace` | 仅 reporter=source：destination_workload_namespace |
| `dst_workload` | 目标 workload | `dst_deployment` → `dst_workload` | 仅 reporter=source：destination_workload |
| `direction` | inbound/outbound | 原样 | reporter=destination → inbound<br>reporter=source → outbound |
| `status_code` | HTTP 状态码 | 原样 | `response_code` → `status_code` |
| `mtls` | true/false | `tls` → `mtls` | `connection_security_policy == "mutual_tls"` → true |

### 2.4 OTel Transform Processor 设计

#### Linkerd（简单 rename，`metricstransform` processor）
```yaml
metricstransform/linkerd_to_mesh:
  transforms:
    - include: response_total
      action: update
      new_name: mesh_request_total
      operations:
        - { action: update_label, label: deployment,     new_label: workload }
        - { action: update_label, label: dst_deployment, new_label: dst_workload }
        - { action: update_label, label: tls,            new_label: mtls }
    - include: response_latency_ms_bucket
      action: update
      new_name: mesh_request_duration_ms_bucket
      operations: [ 同上 ]
```

#### Istio（条件映射，`transform` OTTL processor）
```yaml
transform/istio_to_mesh:
  metric_statements:
    - context: datapoint
      conditions:
        - 'metric.name == "istio_requests_total" or metric.name == "istio_request_duration_milliseconds_bucket"'
      statements:
        - set(attributes["workload"],      attributes["destination_workload"])           where attributes["reporter"] == "destination"
        - set(attributes["namespace"],     attributes["destination_workload_namespace"]) where attributes["reporter"] == "destination"
        - set(attributes["workload"],      attributes["source_workload"])                where attributes["reporter"] == "source"
        - set(attributes["namespace"],     attributes["source_workload_namespace"])      where attributes["reporter"] == "source"
        - set(attributes["direction"], "inbound")  where attributes["reporter"] == "destination"
        - set(attributes["direction"], "outbound") where attributes["reporter"] == "source"
        - set(attributes["dst_workload"],  attributes["destination_workload"])           where attributes["reporter"] == "source"
        - set(attributes["dst_namespace"], attributes["destination_workload_namespace"]) where attributes["reporter"] == "source"
        - set(attributes["status_code"], attributes["response_code"])
        - set(attributes["mtls"], "true")  where attributes["connection_security_policy"] == "mutual_tls"
        - set(attributes["mtls"], "false") where attributes["connection_security_policy"] != "mutual_tls"
    - context: metric
      statements:
        - set(name, "mesh_request_total")              where name == "istio_requests_total"
        - set(name, "mesh_request_duration_ms_bucket") where name == "istio_request_duration_milliseconds_bucket"
```

## 3. Agent 代码改造（Phase 6）

### 3.1 SQL rename（4 处）

文件：`atlhyper_agent_v2/repository/ch/query/slo.go`
1. `ListServiceSLO`：`mv_linkerd_response_total` → `mv_mesh_request_total`；`mv_linkerd_latency_bucket` → `mv_mesh_latency_bucket`；label `deployment` → `workload`、`tls='true'` → `mtls='true'`
2. `ListServiceEdges`：同上，`dst_deployment` → `dst_workload`
3. `GetSLOTimeSeries`：原始表查询的 `MetricName = 'response_total'` → `'mesh_request_total'`；删除 Linkerd 专属 filter（`target_addr NOT LIKE '%:4191'`、`route_name != 'probe'`）

文件：`atlhyper_agent_v2/repository/ch/summary.go`
4. `GetSLOSummary` 的 mesh 段：同类 rename + 删 filter

### 3.2 ClickHouse MV 更新（config 仓库）

当前 `mv_linkerd_*` MV 的 DDL **不在本项目代码中**（Explore 调研确认），估计在 ClickHouse init-db.sh 或手建。需要：
- 新建 `mv_mesh_request_total` + `mv_mesh_latency_bucket`（从 `otel_metrics_sum` 按统一 label 聚合）
- 旧的 `mv_linkerd_*` 保留过渡（不读即是孤儿 MV，不影响）

## 4. 实施顺序（Phase 0 → Phase 6）

### Phase 0：本机工具（5 min）
- 下载 istioctl 1.25.1 到 `~/istio-1.25.1/`
- 改 `config/local/ubuntu/env/infra.env` 加 ISTIO PATH

### Phase 1：Config 仓库重构（30 min，本地完成，不推送）
- 按 §2.2 重构目录：建 `base/`、`mesh/{none,linkerd,istio}/`
- 拆 OTel ConfigMap：`atlhyper-otel-collector.yaml` 只留 Deployment/Service/RBAC；ConfigMap data 移到各 mesh overlay
- 写 3 份完整的 `atlhyper-otel-config.yaml`（共享基础段 + 各自的 mesh 段）
- 顶层 `kustomization.yaml` 暂设 `mesh/none`
- 本地 `kustomize build .` 验证输出，和旧版 diff 只有 mesh 相关差异

### Phase 2：推送 config 仓库 + 验证 none（5 min + 1 min CD）
- 推送到 main
- Deployer 30-60s 应用
- 验证：`kubectl -n atlhyper get cm otel-collector-config -o yaml | grep -E 'linkerd|istio'` 无输出
- 业务 Pod 全 Running

### Phase 3：部署 Istio（15 min）
- `istioctl install --set profile=default --skip-confirmation`
- `kubectl label ns geass-v2-api istio-injection=enabled`（只这两个 ns，atlhyper 不动）
- `kubectl label ns geass-v2-web istio-injection=enabled`
- `kubectl -n geass-v2-{api,web} rollout restart deployment`
- 验证：业务 Pod 变 2/2
- **例外**：`atlhyper-agent` / `node-exporter` hostNetwork=true，不注入，无需处理（因为这两个 Pod 的 ns 是 atlhyper，本来就没打 label）

### Phase 4：切 mesh/istio overlay（1 min + 1 min CD）
- 顶层 kustomization 改 `- mesh/istio`
- 推送 config 仓库
- OTel Collector 自动重启并加载新配置

### Phase 5：实测数据链路（5 min）
- 访问 `https://geass-api.bukahou.com` 产生流量
- ClickHouse 直查：
  ```sql
  SELECT MetricName, count() FROM atlhyper.otel_metrics_sum
  WHERE MetricName = 'mesh_request_total' AND TimeUnix >= now() - INTERVAL 5 MINUTE
  GROUP BY MetricName
  ```
- 期望返回非零行数
- 采集若干行实际 label，确认 workload/namespace/direction/mtls 等字段符合预期

### Phase 6：Agent 代码改造（30 min）
- slo.go 4 处 SQL rename（见 §3.1）
- summary.go 2 处同类改动
- ClickHouse MV 建 `mv_mesh_request_total` + `mv_mesh_latency_bucket`
- `go build` + `go test ./atlhyper_agent_v2/repository/ch/query/`
- commit，推送
- 刷新 Web `/observe/slo` 看 mesh 数据

## 5. 验证清单

| # | 验证项 | 命令/操作 | 通过标准 |
|---|-------|----------|---------|
| V1 | istioctl PATH | `which istioctl` | 路径正确 |
| V2 | kustomize 重构本地正确 | `kustomize build .` | 输出合理，与旧版 diff 仅 mesh 差异 |
| V3 | mesh/none 切换后无回退 | `kubectl -n atlhyper get pods` | 全 Running |
| V4 | Istio sidecar 注入 | `kubectl -n geass-v2-api get pods` | 2/2 |
| V5 | OTel transform 生效 | `kubectl -n atlhyper get cm otel-collector-config -o yaml` | 含 `istio-sidecars` + `transform/istio_to_mesh` |
| V6 | ClickHouse 有统一 metric | SELECT 查询 | `count() > 0` |
| V7 | Agent 编译 + 测试 | `go build && go test` | PASS |
| V8 | Web mesh 页面 | 刷新 `/observe/slo` 服务网格 tab | 数据显示正常 |

## 6. 回滚方案

### 局部回滚
- **Istio 出问题**：顶层改 `mesh/none` + 取消 ns label + rollout restart + `istioctl uninstall --purge`
- **OTel 配置出问题**：顶层改 `mesh/none`，CD 自动恢复无 mesh 版 ConfigMap
- **Agent 代码出问题**：`git revert <Phase 6 commit>`

### 完全回滚到改造前
1. config 仓库：`git revert <Phase 1 重构 commit>`
2. Agent 代码：`git revert <Phase 6 commit>`

## 7. 关键约束与风险

### 约束
- OTel `prometheus` receiver 的 ConfigMap data 是**多行字符串**，Kustomize 不能精细合并 → 3 个 mesh overlay 各有完整 config（接受基础段重复）
- OTel Collector 镜像需要**带 contrib 组件**（`transform` processor 在 contrib 中）。现有 `otel-collector` Deployment 需确认是 contrib 版本

### 风险
| 风险 | 缓解 |
|------|------|
| kustomize 语法错误导致 CD 破坏集群 | Phase 1 本地 `kustomize build` 验证；Phase 2 先切 mesh/none（对现有行为零变更）检验重构正确 |
| OTel Collector 重启丢失内存状态 | batch 队列会丢少量，影响可忽略（30s scrape interval，下次补齐） |
| 切换到 mesh/istio 但 Istio 没装完，OTel scrape 空转 | 无影响（scrape 目标空，无数据写入；等装完自动生效） |
| Agent 代码改造前 mesh 页面空白（Phase 4→6 之间） | 预期行为（集群本来也没数据），用户无感知 |
| ClickHouse MV DDL 存放位置未知 | Phase 5 前需定位（config 仓库 `init-db.sh` 或手建），必要时加到 ClickHouse 初始化逻辑 |

## 8. 文件变更清单

### config 仓库（`~/AtlHyper/GitHub/config/`）
```
local/ubuntu/env/infra.env                                          [修改] 加 ISTIO PATH
clusters/zgmf-x10a/apps/atlhyper/                                   [重构]
├── kustomization.yaml                                                [修改] 顶层
├── base/                                                             [新增目录]
│   ├── kustomization.yaml
│   └── 现有 11 个 yaml 文件（从原位置 git mv，除 otel-config）
├── mesh/
│   ├── none/kustomization.yaml                                       [新增]
│   ├── none/atlhyper-otel-config.yaml                                [新增]
│   ├── linkerd/kustomization.yaml                                    [新增]
│   ├── linkerd/atlhyper-otel-config.yaml                             [新增]
│   ├── linkerd/atlhyper-otel-authz.yaml                              [新增，从 base 挪入]
│   ├── istio/kustomization.yaml                                      [新增]
│   └── istio/atlhyper-otel-config.yaml                               [新增]
└── DEPLOY.md + *.pem                                                 [保持原位]
```

### atlhyper 主仓库（`~/work/github/atlhyper/`）
```
atlhyper_agent_v2/repository/ch/query/slo.go                         [修改] 4 处 SQL
atlhyper_agent_v2/repository/ch/summary.go                           [修改] 2 处 SQL
docs/design/active/agent-mesh-generic-monitoring-design.md           [新增] 本文档
docs/tasks/active/tracker.md                                         [修改] 新增任务条目
```

## 9. 上下文恢复要点（切 session 必读）

若对话切换，按以下顺序恢复上下文：
1. 读本文档 §2.2 的目录结构
2. 读 tracker.md 的 "Agent 泛用型 Mesh 监控" 条目看当前 Phase
3. 检查 `config` 仓库的 `clusters/zgmf-x10a/apps/atlhyper/` 是否已重构
4. 检查 `atlhyper` 主仓库的 `slo.go` 是否已 rename（grep `mv_mesh_request_total`）
5. 检查集群：`kubectl get ns | grep istio`、`kubectl -n geass-v2-api get pods` 确认部署进度
6. 按未完成的 Phase 继续推进

## 10. 和前次任务的关系

本任务是"Counter Reset 修复"（commit 5004efd 那条链）之后的独立任务，不依赖，不冲突。但复用了一些刚建立的基础设施：
- Agent ClickHouse SQL 的结构已通过 Counter Reset 任务走过一遍（`CounterRateExpr` 常量）
- Agent SLO 查询链路（snapshot → push → Master → Web）已验证端到端
- 所以本次只需要 rename + label 映射，数据链路不动

## 附录：OTel Collector 镜像要求

确认当前 Deployment 使用的镜像支持 `transform` processor：
```bash
kubectl -n atlhyper describe deploy otel-collector | grep Image:
```

要求镜像是 `otel/opentelemetry-collector-contrib:*`（不是裸 `otel/opentelemetry-collector`）。如当前用的是 core 版，需要改 Deployment 镜像为 contrib 版（Phase 1 顺便处理）。

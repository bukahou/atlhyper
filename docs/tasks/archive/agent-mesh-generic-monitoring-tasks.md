# Agent 泛用型 Mesh 监控（Kustomize Overlay + OTel Transform + 代码去 Linkerd 化）— 已完成

> 原设计文档: [agent-mesh-generic-monitoring-design.md](../../design/archive/agent-mesh-generic-monitoring-design.md)
> 完成日期: 2026-04-18

## 成果

Agent 不再感知底层网格类型。未来切换 Linkerd / Istio / Cilium 任意 mesh 只改 config 仓库的 mesh overlay（一行 kustomization.yaml），Agent Go 代码零改动。

## Step 1：修改配置（Config 重构 + 切 mesh/none 验证）✅

- Phase 0 本机工具（config 仓库 commit 5e8f90a）
  - istioctl 1.25.1 装到 `~/istio-1.25.1/`
  - `infra.env` 加 ISTIO_HOME + PATH（仅当目录存在时导出，优雅不污染）
- Phase 1 Config 仓库重构（config 仓库 commit 2bd01ce）
  - 平铺 → `base/` + `mesh/{none,linkerd,istio}/` overlay 结构
  - 9 个 yaml git rename 到 base/（100% 相似度）
  - OTel ConfigMap 从 base 中剥离，各 mesh overlay 独立提供完整版本
  - authz 搬到 mesh/linkerd/（仅 linkerd 需要）
- Phase 2 推送 + apply + 切 mesh/none 验证
  - 23 个资源，kubectl apply -k . 顶层一次部署
  - ConfigMap 扫 linkerd/istio 关键字全空 → 重构行为等价旧版

## Step 2：部署 Istio 并验证 ✅

- Phase 3 部署 Istio
  - 首装用默认大 sidecar（2000m limits）撞 ResourceQuota 16，rollout 卡 5 个 Deployment
  - 重装 `--set values.global.proxy.resources.limits.cpu=100m` 小 sidecar
  - 临时扩 quota 16→24 完成 rollout，auth/web 再 restart 换小 sidecar，quota 缩回 16
  - 最终 13+2 Pod 全 2/2 带小 sidecar，quota used=12.3
- Phase 4 切 mesh/istio overlay（config 仓库 3 commit 踩坑）
  - 97c4749: 顶层切 `- mesh/istio`
  - 23dca26: 移除 v0.99+ 才有的 `conditions` 字段（当前镜像 v0.96 不支持）
  - 353db81: scrape 改走 istio-proxy 声明的 15090 端口 + `metrics_path: /stats/prometheus`
- Phase 5 ClickHouse 实测
  - `atlhyper.otel_metrics_sum` 有 `mesh_request_total`，7 个统一 label 正确填充
  - Istio 原生 `istio_requests_total` 已被 transform 完全重命名（ClickHouse 里无残留）

## Step 3：修改代码并测试 ✅

- Phase 6 Agent 代码改造（主仓库 commit 9dc824a，用户本地 build → Docker Hub v0.3.9）
  - `repository/ch/query/slo.go`：3 个查询函数（ListServiceSLO / ListServiceEdges / GetSLOTimeSeries）
  - `repository/ch/summary.go`：GetSLOSummary mesh 段
  - `mv_linkerd_*` → `atlhyper.otel_metrics_sum`（raw 表查询，未建 MV）
  - label: `deployment` → `workload` / `tls` → `mtls` / `dst_deployment` → `dst_workload`
  - 删除 Linkerd 专属 filter（`target_addr NOT LIKE '%:4191'` / `route_name != 'probe'`）
  - `GetSLOTimeSeries` 接口简化：name 占位符 2 个 → 1 个
  - 验证：Web `/observe/slo` 服务网格页面有数据

## 附带发现 & 小坑

1. **OTel Collector v0.96 不支持 transform processor 的 `conditions` 字段**
   - 必须每条 statement 自带 where 子句过滤
2. **istio-proxy 容器只声明 15090 端口**
   - kubernetes_sd 自动用 15090，不要硬替换到 15020（反而丢 IP）
3. **ResourceQuota limits.cpu=16 对默认大 sidecar 不够**
   - Istio 默认 sidecar limits.cpu=2000m，每 Pod 多吃 2 core 非常夸张
   - 家庭集群建议全局设置小 sidecar（100m/128Mi 足够）
4. **ClickHouse Linkerd MV DDL 位置**
   - 在 `base/atlhyper-clickhouse.yaml` 的 init-db.sh 里（设计文档里标记为"待定位"的东西找到了）

## 留作后续独立任务

1. **histogram 数据未进 ClickHouse**（P50/P95/P99 延迟分布当前为 0）
   - OTel Prometheus receiver 对只含 `_bucket`（无 `_count/_sum`）的 Prometheus histogram 会丢弃
   - 修复方案：metric_relabel_configs 的 keep regex 改为保留完整 histogram 三联
   - 需要同步调整 transform rename 规则（name 没有 `_bucket` 后缀）
2. **`mv_mesh_request_total` / `mv_mesh_latency_bucket` MV 未建**
   - 家庭集群 raw 表查询性能够用，生产化时再加预聚合
   - DDL 可照抄 `base/atlhyper-clickhouse.yaml` 里的 `mv_linkerd_*`，改 table 源 + label 名
3. **GetSLOTimeSeries 接口签名变化**
   - 原 `Query(ctx, query, name, name)` → 新 `Query(ctx, query, name)`
   - Handler / Service 层调用方已同步，但如果未来有其他调用方需注意

## Commit 清单

### atlhyper 主仓库
```
9dc824a fix(agent): 服务网格查询改用统一契约 mesh_request_total（未 push）
faa8217 docs: 新增泛用型 Mesh 监控任务设计
80f02c6 docs(tasks): 按三大步分组
ed8fa04 docs(tasks): Phase 0/1 标记完成
dd8683b docs(tasks): Phase 2 完成，Step 1 闭环通过
d3a3c14 docs(tasks): Step 2 全 3 Phase 闭环
```

### config 仓库
```
5e8f90a local: infra.env 加 istio 环境变量
d220dff chore(atlhyper): 镜像升级 v0.3.8 + CH endpoint native 9000
2bd01ce refactor(atlhyper): kustomize overlay 结构
97c4749 feat(atlhyper): 切换 mesh overlay none -> istio
23dca26 fix(otel-istio): 移除 v0.99+ 独有的 conditions 字段
2645c70 fix(otel-istio): scrape address pod_ip 中间态（被下一 commit 覆盖）
353db81 fix(otel-istio): scrape 走 15090 /stats/prometheus
+ 用户手动 bump 到 v0.3.9 并 push Docker Hub + apply 集群
```

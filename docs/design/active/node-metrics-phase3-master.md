# 节点指标迁移 Phase 3：Master 适配

> 状态：**已修订** — 基于真实数据和 TDD 规范调整
> 创建：2026-02-11 | 修订：2026-02-11
> 前置：`node-metrics-phase2-agent.md`（Agent 已输出 OTel 节点指标）
> TDD 规范：`node-metrics-tdd.md`（**权威参考**）
> 共享合约：`model_v2/node_metrics.go`（NodeMetricsSnapshot，**已扩展**）

## 1. 概要

Agent 改造后，`ClusterSnapshot.NodeMetrics` 的基础结构 `map[string]*NodeMetricsSnapshot` 不变。
新增字段（PSI/TCP/System/VMStat/NTP/Softnet）以 JSON 可选字段方式传输，向后兼容。

**Phase 3 工作量**：

| 项目 | 工作量 | 说明 |
|------|--------|------|
| Master 代码改动 | **极小** | 新字段自动序列化/反序列化，主链路无需修改 |
| 前端适配 | **中等** | 新增 PSI/TCP/System 卡片展示，适配真实数据格式 |
| 部署清理 | 下线旧 atlhyper_metrics | 确认新链路稳定后执行 |

---

## 2. 数据流验证

```
Agent ClusterSnapshot.NodeMetrics (map[string]*NodeMetricsSnapshot)
  ↓ gzip + HTTP POST /agent/snapshot
AgentSDK (snapshot.go) → Processor → DataHub Store
  ↓ OnSnapshotReceived 回调
MetricsPersistService.Sync(clusterID)
  ↓ 读 Store → 持久化
  ├── UpsertLatest()     → node_metrics_latest 表 (snapshot_json 包含新字段)
  └── InsertHistory()    → node_metrics_history 表 (不含新字段，结构不变)
  ↓
Query 层 → Gateway API → 前端
```

**新字段影响分析**：

| 层级 | 影响 | 说明 |
|------|------|------|
| AgentSDK | 无 | 序列化 ClusterSnapshot，新字段自动包含 |
| Processor | 无 | 写入 DataHub，不关心具体字段 |
| MetricsPersist UpsertLatest | 无 | JSON 序列化完整 snapshot，新字段自动写入 |
| MetricsPersist InsertHistory | 无 | `ToDataPoint()` 只提取基础指标，不涉及新字段 |
| Query/Gateway | **小改** | API 返回完整 snapshot JSON，新字段自动包含；前端需适配 |

---

## 3. 文件夹结构

```
atlhyper_master_v2/
├── master.go                              不动
├── agentsdk/snapshot.go                   不动
├── processor/processor.go                 不动
├── service/sync/metrics_persist.go        不动
├── database/interfaces.go                 不动
├── database/sqlite/node_metrics.go        不动
├── gateway/handler/node.go                不动 (API 返回完整 JSON)
└── (无新增/修改文件)

atlhyper_web/src/
├── app/cluster/[id]/nodes/               ← 需适配新字段展示
├── app/style-preview/metrics/            ← 已有 mock 实现，需对齐真实数据格式
└── components/node/                      ← 可能需新增/修改组件
```

---

## 4. 字段变化影响

### 4.1 正常填充字段（无变化）

| 字段 | 来源 | Master 使用点 |
|------|------|--------------|
| CPU.UsagePercent | OTel rate | MetricsPersist → latest/history |
| Memory.UsagePercent | OTel gauge | MetricsPersist → latest/history |
| Disk.UsagePercent | OTel gauge | MetricsPersist → latest/history |
| Temperature.CPUTemp | OTel gauge | MetricsPersist → latest/history |

### 4.2 新增字段（自动传输）

| 字段 | JSON key | 类型 | 前端用途 |
|------|----------|------|----------|
| PSI | `psi` | PSIMetrics | PSI 压力卡片 |
| TCP | `tcp` | TCPMetrics | TCP 连接状态卡片 |
| System | `system` | SystemMetrics | 系统资源卡片（conntrack/filefd/entropy） |
| VMStat | `vmstat` | VMStatMetrics | 页错误/Swap I/O 监控 |
| NTP | `ntp` | NTPMetrics | 时间同步状态 |
| Softnet | `softnet` | SoftnetMetrics | 软中断丢包监控 |

### 4.3 变为空值的字段

| 字段 | 旧值 | 新值 | 影响 |
|------|------|------|------|
| Processes | Top N 进程列表 | `nil` | 前端进程列表空态展示 |
| CPU.Model | CPU 型号字符串 | `""` | 前端隐藏此项或显示 "-" |
| Network.IPAddress | 宿主机 IP | `""` | 前端显示 "-" |
| Network.MACAddress | MAC 地址 | `""` | 前端显示 "-" |

---

## 5. 前端适配

### 5.1 真实数据 vs Mock 数据差异

| 项目 | style-preview Mock 数据 | 真实数据 |
|------|------------------------|----------|
| PSI | 10s/60s/300s 三个窗口百分比 | **单一百分比**（rate 近似值） |
| TCP | established/timeWait/closeWait/listen/synRecv/finWait | **只有** currEstab/timeWait/orphan/alloc/inUse/socketsUsed |
| CPU.model | "Intel i5-8500" | `""`（空） |
| CPU.cores/threads | 6/6 | 8/8（真实核数不同于假设） |
| Entropy | 各节点不同值 | 全部 256 bits（Linux 5.x+ 固定值） |
| Swap | 部分节点有 Swap | **全部无 Swap**（SwapTotal=0） |

### 5.2 前端修改清单

1. **PSI 卡片简化**：从三窗口改为单数字显示（`cpu_some_percent`, `io_some_percent` 等）
2. **TCP 卡片调整**：移除 closeWait/listen/synRecv/finWait，只显示可用字段
3. **CPU 型号**：字段为空时隐藏或显示架构（可从 uname_info 的 machine 推导 x86_64/aarch64）
4. **进程列表**：空态展示（`processes === null` guard）
5. **style-preview Mock 数据更新**：对齐真实数据格式

---

## 6. API 响应格式

> 详见 `node-metrics-tdd.md` 第 9 节

Master Gateway 已有的 3 个 API 端点无需修改：
- `GET /api/v2/node-metrics` — 自动包含新字段
- `GET /api/v2/node-metrics/{nodeName}` — 自动包含新字段
- `GET /api/v2/node-metrics/{nodeName}/history` — 不变（历史数据不含新字段）

---

## 7. 清理任务

待 Phase 2 Agent 部署并稳定运行后执行：

### 7.1 下线 atlhyper-metrics DaemonSet

```bash
# 确认新指标数据正常
# ... 在 Master API 验证各节点数据

# 禁用 atlhyper-metrics（添加不可能匹配的 nodeSelector）
kubectl -n atlhyper patch daemonset atlhyper-metrics \
  -p '{"spec":{"template":{"spec":{"nodeSelector":{"disabled":"true"}}}}}'

# 观察 1-2 天，确认降级逻辑不被触发

# 彻底删除
kubectl -n atlhyper delete daemonset atlhyper-metrics
```

### 7.2 可选：清理 Agent 代码

待确认完全不需要推送模式后：

| 操作 | 文件 |
|------|------|
| 删除 ReceiverClient 启动/停止逻辑 | `agent.go` |
| 删除 receiver 包 | `sdk/impl/receiver/server.go` |
| 删除 ReceiverClient 接口 | `sdk/interfaces.go` |
| 简化 MetricsRepository | `repository/metrics/metrics.go`（移除降级逻辑） |

### 7.3 可选：删除 atlhyper_metrics_v2 源码

```
atlhyper_metrics_v2/    # 整个目录可删除
```

---

## 8. 验证清单

### 8.1 数据完整性

```bash
# Master API 查询节点指标
curl -s http://master:8080/api/v2/node-metrics?cluster_id=xxx | jq '.nodes[0]'

# 检查新增字段
# - psi.cpu_some_percent: 0-100 之间
# - tcp.curr_estab: >= 0
# - system.conntrack_entries: > 0
# - vmstat.pgfault_ps: >= 0
# - ntp.synced: true
# - softnet.dropped: >= 0
```

### 8.2 前端页面

| 页面 | 检查点 |
|------|--------|
| Overview | 节点资源卡片显示 CPU/Memory/Disk 使用率（不变） |
| 节点详情 | PSI 卡片显示压力百分比 |
| 节点详情 | TCP 卡片显示连接状态 |
| 节点详情 | 系统资源卡片显示 conntrack/filefd/entropy |
| 节点详情 | 进程列表空态展示 |
| 节点详情 | CPU 型号为空时显示架构标识 |

---

## 9. 总结

| 项目 | 工作量 |
|------|--------|
| Master 代码改动 | **零** |
| 前端适配 | 中等（PSI/TCP 卡片简化 + mock 数据对齐） |
| 部署清理 | 下线 atlhyper-metrics DaemonSet |
| 代码清理 | 可选，删除 ReceiverClient + atlhyper_metrics_v2 |

Master 的"零改动"得益于：
1. JSON 序列化自动包含新字段
2. `ToDataPoint()` 历史数据不涉及新字段
3. SQLite `snapshot_json` 列存储完整 JSON，无需 schema 变更

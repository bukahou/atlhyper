# 节点指标迁移 Phase 2：Agent 改造

> 状态：**已修订** — 基于真实数据和 TDD 规范调整
> 创建：2026-02-11 | 修订：2026-02-11
> 前置：`node-metrics-phase1-infra.md`（OTel 端点已输出 `otel_node_*` 指标）✅ 已完成
> 后续：`node-metrics-phase3-master.md`
> TDD 规范：`node-metrics-tdd.md`（**权威参考**，本文档需与 TDD 文档保持一致）
> 共享合约：`model_v2/node_metrics.go`（NodeMetricsSnapshot，**需扩展**）

## 1. 概要

改造 Agent 的节点指标采集流程，从 ReceiverClient 推送模式切换为 OTelClient 拉取模式。

**核心变化**：

| 特性 | 当前 | 目标 |
|------|------|------|
| 数据源 | atlhyper_metrics_v2 HTTP POST | OTel Collector `:8889/metrics` |
| SDK | ReceiverClient（被动接收） | OTelClient（主动拉取） |
| 数据流 | push → 内存缓存 → GetAll() | pull → rate 计算 → 转换 → GetAll() |
| 同步触发 | 外部推送触发 | Scheduler 定时调用 Sync() |
| 数据模型 | NodeMetricsSnapshot（基础） | NodeMetricsSnapshot（**扩展** PSI/TCP/System/VMStat/NTP/Softnet） |

**不变的**：
- `SnapshotService.Collect()` 调用 `metricsRepo.GetAll()` 不变
- Master 收到的 `ClusterSnapshot.NodeMetrics` 格式兼容（新增字段 JSON 可选）

---

## 2. 真实数据关键发现

> 详见 `node-metrics-tdd.md` 第 1 节

基于 Phase 1 部署后的真实数据验证，以下设计假设需要调整：

| 发现 | 设计调整 |
|------|----------|
| `node_cpu_info` 不存在 | CPU 型号留空，核数从 `cpu_seconds_total` label 推导 |
| `node_tcp_connection_states` 不存在 | TCP 状态仅 CurrEstab/tw/orphan/alloc/inuse/socketsUsed |
| PSI 为累积 counter（非百分比窗口） | rate 计算近似百分比 |
| 大量 shm/tmpfs 文件系统噪音 | 过滤规则：只保留 `/dev/` 开头设备 |
| 大量 veth/flannel/cni 虚拟网络接口 | 过滤规则：排除 lo/veth/flannel/cni/cali |
| 磁盘有 dm-* device-mapper 设备 | 过滤规则：排除 dm-* 避免重复计算 |
| softnet 指标是 per-cpu counter | 解析时所有 CPU 求和 |
| `node_hwmon_temp_crit_celsius` 未在白名单 | 需补充到 OTel ConfigMap（Phase 1 补丁） |

---

## 3. 文件夹结构

```
atlhyper_agent_v2/
├── agent.go                              ← 修改: MetricsRepository 初始化传入 OTelClient
│
├── config/
│   └── types.go                          ← 修改: MetricsConfig 新增 OTel 相关字段
│
├── sdk/
│   ├── interfaces.go                     ← 修改: OTelClient 新增 ScrapeNodeMetrics()
│   ├── types.go                          ← 修改: 新增 OTelNodeRawMetrics 等类型
│   └── impl/
│       ├── otel/
│       │   ├── client.go                 ← 修改: 实现 ScrapeNodeMetrics()
│       │   ├── parser.go                 不动: SLO 指标解析
│       │   ├── node_parser.go            ← NEW: node_exporter 指标解析
│       │   └── node_parser_test.go       ← NEW: 解析器测试
│       │
│       ├── receiver/                     不动 (保留降级用)
│       ├── k8s/                          不动
│       └── ingress/                      不动
│
├── repository/
│   ├── interfaces.go                     ← 修改: MetricsRepository 新增 Sync()
│   │
│   ├── metrics/
│   │   ├── metrics.go                    ← 重写: OTel 拉取 + 降级逻辑
│   │   ├── metrics_test.go              ← NEW: 集成测试
│   │   ├── converter.go                  ← NEW: OTelNodeRawMetrics → NodeMetricsSnapshot
│   │   ├── converter_test.go            ← NEW: 转换器测试
│   │   ├── rate.go                       ← NEW: counter rate 计算器
│   │   ├── rate_test.go                 ← NEW: rate 计算器测试
│   │   └── filter.go                    ← NEW: 过滤规则
│   │
│   ├── k8s/                              不动
│   └── slo/                              不动
│
├── service/                              不动
├── scheduler/
│   └── scheduler.go                      ← 修改: 新增 MetricsSync 循环
│
└── gateway/                              不动

model_v2/
└── node_metrics.go                       ← 修改: 新增 PSI/TCP/System/VMStat/NTP/Softnet 结构体

testdata/                                 ← NEW: 测试数据目录
├── otel_desk_zero.txt                    ← NEW: desk-zero OTel 格式原始指标
├── otel_raspi_zero.txt                   ← NEW: raspi-zero OTel 格式原始指标
└── otel_all_nodes.txt                    ← NEW: 全部 6 节点指标
```

---

## 4. 共享模型扩展 (model_v2/node_metrics.go)

> 详见 `node-metrics-tdd.md` 第 4.1-4.2 节

在 `NodeMetricsSnapshot` 新增 6 个结构体字段：

```go
type NodeMetricsSnapshot struct {
    // ... 现有字段全部保留 ...

    PSI      PSIMetrics      `json:"psi"`
    TCP      TCPMetrics      `json:"tcp"`
    System   SystemMetrics   `json:"system"`
    VMStat   VMStatMetrics   `json:"vmstat"`
    NTP      NTPMetrics      `json:"ntp"`
    Softnet  SoftnetMetrics  `json:"softnet"`
}
```

新增结构体：`PSIMetrics`、`TCPMetrics`、`SystemMetrics`、`VMStatMetrics`、`NTPMetrics`、`SoftnetMetrics`。
完整定义见 TDD 文档第 4.2 节。

**向后兼容性**：旧版 Agent 不发送这些字段 → Master JSON 反序列化时为零值 → 不影响现有功能。

---

## 5. SDK 层改造

### 5.1 OTelClient 接口扩展 (`sdk/interfaces.go`)

```go
type OTelClient interface {
    // 现有
    ScrapeMetrics(ctx context.Context) (*OTelRawMetrics, error)
    IsHealthy(ctx context.Context) bool

    // 新增
    ScrapeNodeMetrics(ctx context.Context) (map[string]*OTelNodeRawMetrics, error)
}
```

### 5.2 新增类型 (`sdk/types.go`)

`OTelNodeRawMetrics` 完整定义见 TDD 文档第 4.3 节。

关键设计决策：
- `CPUSecondsTotal` key 格式 `"cpu:mode"` (如 `"0:idle"`)
- PSI 存原始 counter 值（seconds），rate 计算在 Repository 层
- Softnet 在解析时就做 per-cpu 求和
- 无 `CPUModel`（node_exporter 不提供）

### 5.3 node_parser.go 实现

**解析策略**：
1. 逐行扫描，只处理 `otel_node_` 前缀的行
2. 去除 `otel_` 前缀后匹配指标名
3. 提取 label（instance, cpu, mode, device, mountpoint, fstype, chip, sensor 等）
4. 按 `instance` label 分组填充 `OTelNodeRawMetrics`
5. NodeName 从 `node_uname_info{nodename=...}` 提取
6. **在解析阶段应用过滤规则**：
   - 文件系统：只保留 `/dev/` 开头
   - 网络接口：排除 lo/veth/flannel/cni/cali
   - 磁盘 I/O：排除 dm-*

**测试**：见 TDD 文档第 6 节。

---

## 6. Repository 层改造

### 6.1 接口扩展 (`repository/interfaces.go`)

```go
type MetricsRepository interface {
    GetAll() map[string]*model_v2.NodeMetricsSnapshot
    Sync(ctx context.Context) error  // 新增
}
```

### 6.2 主逻辑 (`repository/metrics/metrics.go`) — 重写

```go
type metricsRepository struct {
    otel     sdk.OTelClient
    receiver sdk.ReceiverClient  // 降级用

    mu       sync.RWMutex
    prev     map[string]*sdk.OTelNodeRawMetrics
    prevTime time.Time
    current  map[string]*model_v2.NodeMetricsSnapshot
}
```

**Sync 流程**：
1. 调用 `otel.ScrapeNodeMetrics()` 拉取原始数据
2. 失败时降级到 `receiver.GetAllNodeMetrics()`
3. 成功时：如有上次数据 (`prev`)，计算 rate 并转换为 `NodeMetricsSnapshot`
4. 首次 Sync 只存原始值，不输出快照
5. 更新 `prev` 和 `prevTime`

### 6.3 converter.go — 转换逻辑

将 `OTelNodeRawMetrics(cur)` + `OTelNodeRawMetrics(prev)` + `elapsed` → `NodeMetricsSnapshot`。

**各模块计算逻辑**：

| 模块 | 类型 | 计算方式 |
|------|------|----------|
| CPU usage | counter → rate | 聚合所有核 delta(total-idle)/delta(total)*100 |
| CPU per-core | counter → rate | 每核 (total-idle)/total*100 |
| CPU freq | gauge | 所有核平均值 Hz→MHz |
| Memory | gauge | Used = Total - Available; Percent = Used/Total*100 |
| Disk space | gauge | Used = Size - Avail; Percent = Used/Size*100 |
| Disk I/O rate | counter → rate | delta(bytes)/elapsed |
| Disk IOPS | counter → rate | delta(ops)/elapsed |
| Disk IO util | counter → rate | delta(io_time)/elapsed*100 (capped at 100) |
| Network rate | counter → rate | delta(bytes)/elapsed |
| Temperature | gauge | CPUTemp = coretemp temp1(x86) 或 max(thermal_zone/adc)(arm64) |
| PSI | counter → rate*100 | delta(seconds)/elapsed*100 |
| TCP/Socket | gauge | 直接取值 |
| System | gauge | 直接取值 |
| VMStat | counter → rate | delta(count)/elapsed |
| NTP | gauge | 直接取值; Synced = status==1 |
| Softnet | gauge(已累积) | 直接取值 |
| Uptime | 计算 | now() - BootTime |

**测试**：见 TDD 文档第 7 节。

### 6.4 rate.go — Counter Rate 计算器

```go
func counterRate(cur, prev, elapsed float64) float64
func counterDelta(cur, prev float64) float64
```

处理 counter reset（cur < prev 时返回 0）。

**测试**：见 TDD 文档第 8 节。

### 6.5 filter.go — 过滤规则

```go
func shouldKeepFilesystem(device, fstype, mountpoint string) bool
func shouldKeepNetwork(device string) bool
func shouldKeepDiskIO(device string) bool
```

**测试**：见 TDD 文档第 5 节。

---

## 7. Scheduler 层改造

在 Scheduler 新增独立的 MetricsSync 循环：

```go
func (s *Scheduler) runMetricsSyncLoop()   // 每 15 秒
func (s *Scheduler) syncMetrics()           // 执行同步
```

**设计选择**：
- 独立 15s 循环（与快照循环 10s 解耦）
- 超时 10s（留 5s 余量）
- 首次立即执行（存首次原始值）

---

## 8. agent.go 初始化改造

```go
// MetricsRepository: 优先用 OTel，降级用 Receiver
if cfg.SLO.Enabled && otelClient != nil {
    metricsRepo = metricsrepo.NewMetricsRepository(otelClient, receiverClient)
} else if receiverClient != nil {
    metricsRepo = metricsrepo.NewLegacyMetricsRepository(receiverClient)
}

// Scheduler 增加 metricsRepo 参数
sched := scheduler.New(schedCfg, snapshotSvc, commandSvc, masterGw, metricsRepo)
```

---

## 9. 降级策略

```
正常: OTelClient.ScrapeNodeMetrics() → rate 计算 → NodeMetricsSnapshot
  ↓ OTel 失败
降级: ReceiverClient.GetAllNodeMetrics() → 直接使用
  ↓ Receiver 也无数据
兜底: GetAll() 返回空 map → SnapshotService 正常处理
```

---

## 10. TDD 实施顺序

```
 1. ✅ Phase 1 基础设施已部署（node_exporter + OTel Collector）
 2. 补充 OTel 白名单 (node_hwmon_temp_crit_celsius)
 3. model_v2/node_metrics.go: 新增 PSI/TCP/System/VMStat/NTP/Softnet 结构体
 4. 创建 testdata/ 测试数据文件
 5. sdk/types.go: 新增 OTelNodeRawMetrics 等类型
 6. sdk/interfaces.go: OTelClient 新增 ScrapeNodeMetrics
 7. 编写 node_parser_test.go → 实现 node_parser.go (TDD)
 8. sdk/impl/otel/client.go: 实现 ScrapeNodeMetrics
 9. repository/metrics/filter.go: 过滤规则
10. 编写 rate_test.go → 实现 rate.go (TDD)
11. 编写 converter_test.go → 实现 converter.go (TDD)
12. 编写 metrics_test.go → 实现 metrics.go (TDD)
13. scheduler/scheduler.go: 新增 MetricsSync 循环
14. agent.go: 初始化参数调整
15. go build 编译验证
16. 真实数据端到端验证
```

---

## 11. 风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| 首次 Sync 无数据 | rate 需要两次采样 | 首次只存原始值，第二次才输出 |
| counter reset | 节点重启后 counter 归零 | counterRate 检测 cur < prev 返回 0 |
| arm64 温度芯片名不同 | CPUTemp 取值逻辑需适配 | converter 按优先级匹配多种芯片名 |
| OTel 不可用 | 无节点指标 | 降级到 ReceiverClient |
| ScrapeNodeMetrics 与 ScrapeMetrics 竞争 | 并发 HTTP GET | fetch() 无状态，可并发 |
| 新字段向后兼容 | 旧 Master 不认识新字段 | JSON 可选字段，零值不影响 |

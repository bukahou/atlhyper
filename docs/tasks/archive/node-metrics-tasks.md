# Node Metrics 硬件指标监控 - 任务追踪

> 设计文档:
> - 总览: `docs/design/node-metrics-design.md`
> - Metrics: `docs/design/atlhyper-metrics-v2.md`
> - Agent: `docs/design/node-metrics-agent.md`
> - Master: `docs/design/node-metrics-master.md`

---

## 任务概览

| 阶段 | 组件 | 任务 | 状态 |
|------|------|------|------|
| 1 | 共用 | 数据模型定义 | ✅ 完成 |
| 2 | Web | 前端 UI + Mock | ✅ 完成 |
| 3 | Metrics | atlhyper_metrics_v2 采集器 | ✅ 完成 |
| 4 | Agent | 接收 + 聚合到 Snapshot | ✅ 完成 |
| 5 | Master | 持久化 + API | ✅ 完成 |
| 6 | Web | 替换 Mock 为 API | ✅ 完成 |
| 7 | 部署 | DaemonSet + Dockerfile | ✅ 完成 |

---

## Phase 1: 共用数据模型 ✅ 完成

### 1.1 model_v2/node_metrics.go [新增]

- [x] `NodeMetricsSnapshot` - 节点指标快照
- [x] `CPUMetrics` - CPU 指标（使用率、每核、负载、型号、主频）
- [x] `MemoryMetrics` - 内存指标（总量、已用、可用、Swap、缓存）
- [x] `DiskMetrics` - 磁盘指标（设备、挂载点、空间、I/O 速率、IOPS、IOUtil）
- [x] `NetworkMetrics` - 网络指标（接口、IP、MAC、状态、速率、错误/丢包）
- [x] `TemperatureMetrics` - 温度指标（CPU 温度、传感器列表）
- [x] `SensorReading` - 传感器读数
- [x] `ProcessMetrics` - 进程指标（PID、名称、用户、状态、CPU/内存、线程）
- [x] `MetricsDataPoint` - 历史数据点（用于趋势图）
- [x] `ClusterMetricsSummary` - 集群汇总统计

### 1.2 model_v2/snapshot.go [修改]

- [x] `ClusterSnapshot` 添加 `NodeMetrics map[string]*NodeMetricsSnapshot` 字段

---

## Phase 2: 前端 UI + Mock ✅ 完成

- [x] `src/types/node-metrics.ts` - 类型定义
- [x] `src/app/system/metrics/mock/data.ts` - Mock 数据
- [x] `src/app/system/metrics/page.tsx` - 页面
- [x] `src/app/system/metrics/components/` - UI 组件
  - [x] CPUCard, MemoryCard, DiskCard, NetworkCard
  - [x] TemperatureCard, ProcessTable, ResourceChart

---

## Phase 3: atlhyper_metrics_v2 采集器 ✅ 完成

### 3.1 项目结构

```
atlhyper_metrics_v2/
├── cmd/main.go
├── config/
│   ├── types.go
│   └── config.go
├── collector/
│   ├── interfaces.go
│   ├── cpu.go
│   ├── memory.go
│   ├── disk.go
│   ├── network.go
│   ├── temperature.go
│   └── process.go
├── aggregator/
│   └── snapshot.go
├── pusher/
│   └── http.go
└── utils/
    ├── types.go
    └── procfs.go
```

### 3.2 config/ 配置模块

- [x] `config/types.go`
  - [x] Config 结构体
  - [x] Paths 子结构（ProcRoot、SysRoot、HostRoot）
  - [x] Collect 子结构（TopProcesses）
- [x] `config/config.go`
  - [x] 从环境变量加载
  - [x] 默认值处理

### 3.3 collector/ 采集器

- [x] `collector/interfaces.go`
  - [x] Collector 接口定义
  - [x] CPUCollector / MemoryCollector / DiskCollector 等接口

- [x] `collector/cpu.go`
  - [x] 读取 /proc/stat（总使用率 + 每核）
  - [x] 读取 /proc/loadavg（负载）
  - [x] 读取 /proc/cpuinfo（型号、主频）
  - [x] 后台采样循环（1s）+ 差值计算

- [x] `collector/memory.go`
  - [x] 读取 /proc/meminfo
  - [x] 解析 MemTotal、MemAvailable、MemFree
  - [x] 解析 SwapTotal、SwapFree、Cached、Buffers
  - [x] 计算使用率

- [x] `collector/disk.go`
  - [x] 读取 /proc/mounts（挂载点列表）
  - [x] 读取 /proc/diskstats（I/O 统计）
  - [x] 过滤虚拟文件系统
  - [x] Statfs 获取空间使用
  - [x] 计算 I/O 速率、IOPS、IOUtil
  - [x] 支持多磁盘

- [x] `collector/network.go`
  - [x] 读取 /proc/net/dev（流量统计）
  - [x] 过滤虚拟接口（lo, docker, veth, br-）
  - [x] 获取 IP / MAC 地址（net.Interfaces）
  - [x] 读取 /sys/class/net/{iface}/operstate（状态）
  - [x] 读取 /sys/class/net/{iface}/speed（速度）
  - [x] 计算速率（bytes/s, packets/s）
  - [x] 支持多网卡

- [x] `collector/temperature.go`
  - [x] 遍历 /sys/class/hwmon/（所有传感器）
  - [x] 解析 name、temp*_input、temp*_max、temp*_crit、temp*_label
  - [x] 识别 CPU 温度（coretemp, k10temp）
  - [x] 回退到 thermal_zone（树莓派等）

- [x] `collector/process.go`
  - [x] 扫描 /proc/[pid]/stat
  - [x] 读取 /proc/[pid]/status（UID, VmRSS, Threads）
  - [x] 读取 /proc/[pid]/cmdline（完整命令）
  - [x] 计算 CPU 使用率（差值法）
  - [x] 计算内存百分比
  - [x] 过滤线程（Tgid != PID）
  - [x] 解析用户名（带缓存）
  - [x] TopK 排序
  - [x] 后台采样循环（3s）

### 3.4 utils/ 工具

- [x] `utils/types.go`
  - [x] CPURawStats（差值计算用）
  - [x] DiskRawStats
  - [x] NetRawStats
  - [x] ProcRawStats

- [x] `utils/procfs.go`
  - [x] /proc 文件解析工具函数

### 3.5 aggregator/ 聚合

- [x] `aggregator/snapshot.go`
  - [x] 组合所有采集器结果
  - [x] 构建 NodeMetricsSnapshot

### 3.6 pusher/ 推送

- [x] `pusher/http.go`
  - [x] HTTP POST 到 Agent
  - [x] 重试逻辑（3 次）
  - [x] 超时处理（5s）

### 3.7 cmd/ 入口

- [x] `cmd/main.go`
  - [x] 初始化配置
  - [x] 初始化所有采集器
  - [x] 定时触发采集 + 推送（5s）
  - [x] 优雅退出（signal handling）

---

## Phase 4: Agent 对接 ✅ 完成

### 4.1 SDK 层 [新增]

- [x] `sdk/interfaces_metrics.go`
  - [x] MetricsReceiver 接口定义
  - [x] POST /metrics/node 路由

- [x] `sdk/impl/metrics.go`
  - [x] MetricsReceiverImpl 实现
  - [x] HandleNodeMetrics() 处理函数
  - [x] JSON 解析 → 内存存储

### 4.2 Repository 层 [新增]

- [x] `repository/metrics_repository.go`
  - [x] MetricsRepository 结构体
  - [x] `data map[string]*NodeMetricsSnapshot`（按节点名）
  - [x] `Save(snapshot)` - 覆盖式存储
  - [x] `Get(nodeName)` - 获取指定节点
  - [x] `GetAll()` - 获取所有节点
  - [x] `Count()` - 返回节点数
  - [x] sync.RWMutex 并发保护

### 4.3 Service 层 [修改]

- [x] `service/snapshot_service.go`
  - [x] 注入 metricsRepo
  - [x] BuildSnapshot() 中添加 `snapshot.NodeMetrics = s.metricsRepo.GetAll()`

### 4.4 初始化 [修改]

- [x] `agent.go`
  - [x] 创建 MetricsRepository
  - [x] 注入到 SnapshotService
  - [x] 创建 SDK Server
  - [x] 启动 SDK Server（端口 8082）

---

## Phase 5: Master 对接 ✅ 完成

### 5.1 Database 层

- [x] `database/interfaces.go` [修改]
  - [x] NodeMetricsLatest 数据模型
  - [x] NodeMetricsHistory 数据模型
  - [x] NodeMetricsRepository 接口
  - [x] NodeMetricsDialect 接口

- [x] `database/sqlite/node_metrics.go` [新增]
  - [x] NodeMetricsDialect 实现
  - [x] CreateLatestTable() - 实时数据表
  - [x] CreateHistoryTable() - 趋势数据表
  - [x] CreateHistoryIndexes() - 索引
  - [x] UpsertLatest() / GetLatest() / ListLatest()
  - [x] InsertHistory() / GetHistory()
  - [x] CleanupHistory()

- [x] `database/repo/node_metrics.go` [新增]
  - [x] NodeMetricsRepo 实现
  - [x] Migrate() - 表迁移
  - [x] UpsertLatest() - JSON 序列化存储
  - [x] GetLatest() / ListLatest() - JSON 反序列化
  - [x] InsertHistory() / GetHistory()
  - [x] CleanupHistory()

### 5.2 Service 层

- [x] `service/sync/metrics_persist.go` [新增]
  - [x] MetricsPersistService 结构体
  - [x] 注入 datahub.Store + NodeMetricsRepository
  - [x] Start() / Stop()
  - [x] Sync(clusterID) - 从 DataHub 读取 → 写入 SQLite
    - [x] UpsertLatest（每次同步）
    - [x] InsertHistory（5 分钟采样）
  - [x] shouldSample() - 采样间隔判断
  - [x] snapshotToDataPoint() - 提取关键指标
  - [x] cleanupLoop() - 定期清理（30 天）

### 5.3 Processor 层 [微调]

- [x] `processor/processor.go`
  - [x] 在 onSnapshotReceived 回调中添加 MetricsPersistService.Sync

### 5.4 Gateway 层

- [x] `gateway/handler/node_metrics.go` [新增]
  - [x] NodeMetricsHandler 结构体
  - [x] 注入 database.NodeMetricsRepository（直接用 database 层）
  - [x] GetClusterNodeMetrics() - GET /clusters/{id}/node-metrics
  - [x] GetNodeMetricsDetail() - GET /clusters/{id}/node-metrics/{nodeName}
  - [x] GetNodeMetricsHistory() - GET /clusters/{id}/node-metrics/{nodeName}/history
  - [x] calculateSummary() - 汇总统计

- [x] `gateway/routes.go` [修改]
  - [x] 注册 /api/v2/clusters/:clusterId/node-metrics 路由

### 5.5 初始化 [修改]

- [x] `master.go`
  - [x] 创建 NodeMetricsDialect
  - [x] 创建 NodeMetricsRepo + 执行 Migrate()
  - [x] 创建 MetricsPersistService
  - [x] 配置 Processor 回调
  - [x] 创建 NodeMetricsHandler 并注册路由

---

## Phase 6: 前端 API 对接 ✅ 完成

### 6.1 API 模块 [新增]

- [x] `src/api/node-metrics.ts`
  - [x] `getClusterNodeMetrics(clusterId)` - 集群汇总 + 节点列表
  - [x] `getNodeMetricsDetail(clusterId, nodeName)` - 单节点详情
  - [x] `getNodeMetricsHistory(clusterId, nodeName, hours)` - 趋势数据

### 6.2 页面修改

- [x] `src/app/system/metrics/page.tsx` [修改]
  - [x] 删除 mock import
  - [x] useEffect 调用 getClusterNodeMetrics()
  - [x] 实时刷新（setInterval 5s）
  - [x] 错误处理 + 加载状态
  - [x] 节点点击 → 调用 getNodeMetricsDetail()

### 6.3 趋势图组件

- [x] `src/app/system/metrics/components/TrendChart.tsx`
  - [x] 调用 getNodeMetricsHistory()
  - [x] 支持 24h / 7d / 30d 切换
  - [x] 显示 CPU / 内存 / 磁盘 / 网络趋势
  - [x] 使用 recharts 绑定数据

---

## Phase 7: 部署配置 ✅ 完成

### 7.1 Dockerfile [新增]

- [x] `deploy/docker/Dockerfile.metrics`
  - [x] 多阶段构建
  - [x] 基于 alpine
  - [x] 入口：/atlhyper_metrics_v2

### 7.2 K8s 部署 [新增]

- [x] `deploy/k8s/atlhyper-metrics.yaml`
  - [x] DaemonSet 配置
  - [x] hostPID: true
  - [x] 容忍所有污点（tolerations: Exists）
  - [x] 环境变量 NODE_NAME（Downward API）
  - [x] 环境变量 AGENT_ADDR（Service DNS）
  - [x] 挂载卷：/host_proc ← /proc
  - [x] 挂载卷：/host_sys ← /sys
  - [x] 挂载卷：/host_root ← /
  - [x] 资源限制：50m-200m CPU, 64Mi-128Mi Mem

### 7.3 构建脚本 [新增]

- [x] `deploy/scripts/build_metrics.sh`
  - [x] 编译二进制
  - [x] 构建 Docker 镜像

---

## 验证清单

- [ ] CPU 使用率准确（对比 htop）
- [ ] 内存使用率准确（对比 free）
- [ ] 磁盘 I/O 速率准确（对比 iostat）
- [ ] 网络速率准确（对比 iftop）
- [ ] 温度读数正常
- [ ] Top 进程排序正确
- [ ] 历史趋势图显示正常
- [ ] 多节点数据正确聚合
- [ ] 5 分钟采样正常工作
- [ ] 30 天数据清理正常

---

## 依赖关系

```
Phase 1 (模型)
    │
    ├──▶ Phase 3 (Metrics 采集器)
    │
    ├──▶ Phase 4 (Agent 对接)
    │
    └──▶ Phase 5 (Master 对接)
                │
                └──▶ Phase 6 (Web API 对接)

Phase 3 + 4 + 5 ──────────▶ Phase 7 (部署)
```

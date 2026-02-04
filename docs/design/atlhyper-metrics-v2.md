# atlhyper_metrics_v2 - 节点指标采集器设计文档

> 总览文档: `docs/design/node-metrics-design.md`

## 1. 概述

### 1.1 功能描述

`atlhyper_metrics_v2` 是运行在 K8s 集群中的节点级硬件指标采集器，负责：
- 采集节点硬件指标（CPU、内存、磁盘、网络、温度、进程）
- 定时推送到 `atlhyper_agent`（单实例，通过 Service 访问）
- 支持多设备（多磁盘、多网卡）

### 1.2 模块定位

`atlhyper_metrics_v2` 是 AtlHyper 项目的子模块，使用项目根目录的 `go.mod`：

```
AtlHyper/                          # 项目根目录
├── go.mod                         # 总 mod 文件
├── go.sum
├── model_v2/                      # 共用数据模型
│   └── node_metrics.go
├── atlhyper_metrics_v2/           # 本模块
│   ├── cmd/
│   ├── config/
│   ├── collector/
│   └── ...
├── atlhyper_agent_v2/
└── atlhyper_master_v2/
```

### 1.3 部署方式

以 **DaemonSet** 形式部署，每个节点运行一个 Pod。Agent 为**单实例**部署。

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                  K8s Cluster                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│   Node 1                    Node 2                    Node N                     │
│  ┌───────────────────┐     ┌───────────────────┐     ┌───────────────────┐      │
│  │atlhyper_metrics_v2│     │atlhyper_metrics_v2│     │atlhyper_metrics_v2│      │
│  │   (DaemonSet)     │     │   (DaemonSet)     │ ... │   (DaemonSet)     │      │
│  │                   │     │                   │     │                   │      │
│  │ 读取 /proc /sys   │     │ 读取 /proc /sys   │     │ 读取 /proc /sys   │      │
│  └─────────┬─────────┘     └─────────┬─────────┘     └─────────┬─────────┘      │
│            │                         │                         │                │
│            │ POST /metrics/node      │                         │                │
│            └─────────────────────────┼─────────────────────────┘                │
│                                      ▼                                          │
│              ┌───────────────────────────────────────────────────┐              │
│              │              atlhyper_agent (单实例)               │              │
│              │                                                   │              │
│              │  接收各节点 Metrics → 聚合到 ClusterSnapshot       │              │
│              └───────────────────────────┬───────────────────────┘              │
│                                          │ RESTful                              │
│                                          ▼                                      │
│              ┌───────────────────────────────────────────────────┐              │
│              │                 atlhyper_master                    │              │
│              └───────────────────────────────────────────────────┘              │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

**挂载宿主机目录：**
- `/host_proc` ← `/proc`
- `/host_sys` ← `/sys`
- `/host_root` ← `/`

### 1.3 与 atlhyper_metrics 的区别

| 项目 | atlhyper_metrics (v1) | atlhyper_metrics_v2 |
|------|----------------------|---------------------|
| 模型包 | `model/collect`（已丢失） | `model_v2/node_metrics.go` |
| 磁盘 | 单磁盘 | 多磁盘 + I/O 统计 |
| 网络 | 单接口 | 多接口 + 完整统计 |
| 进程 | 简化版 | 完整版（状态、线程、启动时间） |
| 温度 | 基础 | 传感器详情 |

---

## 2. 架构设计

### 2.1 项目结构

```
AtlHyper/                               # 项目根目录
├── go.mod                              # 总 mod 文件（共用）
├── go.sum
│
├── model_v2/                           # 共用数据模型
│   ├── node_metrics.go                 # NodeMetricsSnapshot 等结构
│   └── snapshot.go                     # ClusterSnapshot（含 NodeMetrics 字段）
│
├── atlhyper_metrics_v2/                # 本模块
│   ├── cmd/
│   │   └── main.go                     # 入口
│   │
│   ├── config/
│   │   ├── config.go                   # 配置加载（环境变量 + 默认值）
│   │   └── types.go                    # 配置结构体
│   │
│   ├── collector/                      # 采集器（核心）
│   │   ├── interfaces.go               # Collector 接口定义
│   │   ├── cpu.go                      # CPU 采集（差值计算）
│   │   ├── memory.go                   # 内存采集
│   │   ├── disk.go                     # 磁盘采集（多磁盘 + I/O）
│   │   ├── network.go                  # 网络采集（多网卡）
│   │   ├── temperature.go              # 温度采集（hwmon + thermal_zone）
│   │   └── process.go                  # 进程采集（TopK）
│   │
│   ├── aggregator/
│   │   └── snapshot.go                 # 聚合所有采集器结果为 NodeMetricsSnapshot
│   │
│   ├── pusher/
│   │   └── http.go                     # HTTP 推送到 Agent（重试 + 超时）
│   │
│   └── utils/
│       ├── types.go                    # 差值计算用类型（CPURawStats 等）
│       └── procfs.go                   # /proc 解析工具函数
│
├── atlhyper_agent_v2/                  # Agent 模块
└── atlhyper_master_v2/                 # Master 模块
```

### 2.2 模块依赖

```
atlhyper_metrics_v2
    │
    └── model_v2/node_metrics.go    # 共用数据模型
            │
            └── NodeMetricsSnapshot
            └── CPUMetrics
            └── MemoryMetrics
            └── DiskMetrics
            └── NetworkMetrics
            └── TemperatureMetrics
            └── ProcessMetrics
```

**import 示例：**

```go
package main

import (
    "AtlHyper/atlhyper_metrics_v2/collector"
    "AtlHyper/atlhyper_metrics_v2/config"
    "AtlHyper/model_v2"
)
```

### 2.3 数据流

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        atlhyper_metrics_v2 内部                             │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                          Scheduler (5s)                               │  │
│  └───────────────────────────────┬──────────────────────────────────────┘  │
│                                  │                                         │
│                                  ▼                                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  │
│  │   CPU   │ │ Memory  │ │  Disk   │ │ Network │ │  Temp   │ │ Process │  │
│  │Collector│ │Collector│ │Collector│ │Collector│ │Collector│ │Collector│  │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘  │
│       │          │          │          │          │          │           │
│       └──────────┴──────────┴────┬─────┴──────────┴──────────┘           │
│                                  │                                        │
│                                  ▼                                        │
│                     ┌────────────────────────┐                            │
│                     │      Aggregator        │                            │
│                     │  → NodeMetricsSnapshot │                            │
│                     └───────────┬────────────┘                            │
│                                 │                                         │
│                                 ▼                                         │
│                     ┌────────────────────────┐                            │
│                     │        Pusher          │                            │
│                     │  POST /metrics/node    │                            │
│                     └───────────┬────────────┘                            │
│                                 │                                         │
└─────────────────────────────────┼─────────────────────────────────────────┘
                                  │ HTTP
                                  ▼
                     ┌────────────────────────┐
                     │    atlhyper_agent      │
                     │      :8082             │
                     └────────────────────────┘
```

### 2.4 采集周期

| 指标类型 | 采集周期 | 说明 |
|----------|----------|------|
| CPU 使用率 | 1s 内部采样 | 需要两次采样计算差值（见下文说明） |
| 进程 Top | 3s 内部采样 | 避免频繁扫描 /proc |
| 其他指标 | 5s | 与推送周期同步 |
| **推送周期** | **5s** | 推送到 Agent |

### 2.5 为什么 CPU 需要差值计算

Linux `/proc/stat` 提供的是**累计 jiffies**（CPU 时钟周期数），不是瞬时使用率：

```
# /proc/stat 内容示例
cpu  10132153 290696 3084719 46828483 16683 0 25195 0 0 0
     ^user    ^nice  ^system ^idle    ^iowait ...
```

这些是从系统启动以来的累计值。要计算使用率必须：

1. T1 时刻读取 `total_1 = user + nice + system + idle + ...`，`idle_1`
2. T2 时刻读取 `total_2`，`idle_2`
3. 计算：`usage = (total_delta - idle_delta) / total_delta * 100%`

这是 `htop`、`top`、`vmstat` 等所有工具的标准做法，没有捷径。

同理，磁盘 I/O 速率、网络速率、进程 CPU 使用率也需要差值计算。

---

## 3. 数据模型

### 3.1 共用模型（model_v2/node_metrics.go）

```go
package model_v2

import "time"

// ==================== 节点指标快照 ====================

// NodeMetricsSnapshot 节点硬件指标快照
type NodeMetricsSnapshot struct {
    NodeName     string             `json:"node_name"`
    Timestamp    time.Time          `json:"timestamp"`
    CPU          CPUMetrics         `json:"cpu"`
    Memory       MemoryMetrics      `json:"memory"`
    Disks        []DiskMetrics      `json:"disks"`
    Networks     []NetworkMetrics   `json:"networks"`
    Temperature  TemperatureMetrics `json:"temperature"`
    TopProcesses []ProcessMetrics   `json:"top_processes"`
}

// ==================== CPU 指标 ====================

type CPUMetrics struct {
    UsagePercent float64   `json:"usage_percent"`   // 总使用率 (0-100)
    CoreCount    int       `json:"core_count"`      // 逻辑核心数
    CoreUsages   []float64 `json:"core_usages"`     // 每核使用率
    LoadAvg1     float64   `json:"load_avg_1"`      // 1分钟负载
    LoadAvg5     float64   `json:"load_avg_5"`      // 5分钟负载
    LoadAvg15    float64   `json:"load_avg_15"`     // 15分钟负载
    Model        string    `json:"model"`           // CPU 型号
    Frequency    int       `json:"frequency"`       // 基础主频 (MHz)
}

// ==================== 内存指标 ====================

type MemoryMetrics struct {
    TotalBytes       uint64  `json:"total_bytes"`
    UsedBytes        uint64  `json:"used_bytes"`
    AvailableBytes   uint64  `json:"available_bytes"`
    UsagePercent     float64 `json:"usage_percent"`
    SwapTotalBytes   uint64  `json:"swap_total_bytes"`
    SwapUsedBytes    uint64  `json:"swap_used_bytes"`
    SwapUsagePercent float64 `json:"swap_usage_percent"`
    Cached           uint64  `json:"cached"`
    Buffers          uint64  `json:"buffers"`
}

// ==================== 磁盘指标 ====================

type DiskMetrics struct {
    Device         string  `json:"device"`          // 设备名 (sda, nvme0n1)
    MountPoint     string  `json:"mount_point"`     // 挂载点
    FsType         string  `json:"fs_type"`         // 文件系统类型
    TotalBytes     uint64  `json:"total_bytes"`
    UsedBytes      uint64  `json:"used_bytes"`
    AvailableBytes uint64  `json:"available_bytes"`
    UsagePercent   float64 `json:"usage_percent"`
    ReadBytesPS    float64 `json:"read_bytes_ps"`   // 读取速率 bytes/s
    WriteBytesPS   float64 `json:"write_bytes_ps"`  // 写入速率 bytes/s
    IOPS           float64 `json:"iops"`            // 每秒 I/O 操作数
    IOUtil         float64 `json:"io_util"`         // I/O 利用率 (0-100)
}

// ==================== 网络指标 ====================

type NetworkMetrics struct {
    Interface   string  `json:"interface"`       // 接口名 (eth0, ens192)
    IPAddress   string  `json:"ip_address"`      // IPv4 地址
    MACAddress  string  `json:"mac_address"`     // MAC 地址
    Status      string  `json:"status"`          // up/down
    Speed       int     `json:"speed"`           // 链路速度 (Mbps)
    RxBytesPS   float64 `json:"rx_bytes_ps"`     // 接收速率 bytes/s
    TxBytesPS   float64 `json:"tx_bytes_ps"`     // 发送速率 bytes/s
    RxPacketsPS float64 `json:"rx_packets_ps"`   // 接收包数/s
    TxPacketsPS float64 `json:"tx_packets_ps"`   // 发送包数/s
    RxErrors    uint64  `json:"rx_errors"`       // 接收错误累计
    TxErrors    uint64  `json:"tx_errors"`       // 发送错误累计
    RxDropped   uint64  `json:"rx_dropped"`      // 接收丢包累计
    TxDropped   uint64  `json:"tx_dropped"`      // 发送丢包累计
}

// ==================== 温度指标 ====================

type TemperatureMetrics struct {
    CPUTemp    float64         `json:"cpu_temp"`     // CPU 当前温度 (°C)
    CPUTempMax float64         `json:"cpu_temp_max"` // CPU 最高阈值
    Sensors    []SensorReading `json:"sensors"`      // 所有传感器
}

type SensorReading struct {
    Name     string   `json:"name"`              // 传感器名 (coretemp, k10temp)
    Label    string   `json:"label"`             // 标签 (Core 0, Tctl)
    Temp     float64  `json:"temp"`              // 当前温度
    High     *float64 `json:"high,omitempty"`    // 高温阈值
    Critical *float64 `json:"critical,omitempty"` // 临界阈值
}

// ==================== 进程指标 ====================

type ProcessMetrics struct {
    PID        int     `json:"pid"`
    Name       string  `json:"name"`           // 进程名 (comm)
    User       string  `json:"user"`           // 用户名
    State      string  `json:"state"`          // R/S/D/Z/T
    CPUPercent float64 `json:"cpu_percent"`    // CPU 使用率
    MemPercent float64 `json:"mem_percent"`    // 内存使用率
    MemBytes   uint64  `json:"mem_bytes"`      // 内存使用 (bytes)
    Threads    int     `json:"threads"`        // 线程数
    StartTime  string  `json:"start_time"`     // 启动时间 (ISO8601)
    Command    string  `json:"command"`        // 完整命令行
}

// ==================== 历史数据点 ====================

type MetricsDataPoint struct {
    Timestamp    int64   `json:"timestamp"`       // Unix timestamp (ms)
    CPUUsage     float64 `json:"cpu_usage"`
    MemUsage     float64 `json:"mem_usage"`
    DiskUsage    float64 `json:"disk_usage"`
    Temperature  float64 `json:"temperature"`
    NetRxBytesPS float64 `json:"net_rx_bytes_ps"`
    NetTxBytesPS float64 `json:"net_tx_bytes_ps"`
}

// ==================== 集群汇总 ====================

type ClusterMetricsSummary struct {
    TotalNodes    int     `json:"total_nodes"`
    ReadyNodes    int     `json:"ready_nodes"`
    MetricsNodes  int     `json:"metrics_nodes"`
    AvgCPU        float64 `json:"avg_cpu"`
    AvgMemory     float64 `json:"avg_memory"`
    MaxTemp       float64 `json:"max_temp"`
    MaxTempNode   string  `json:"max_temp_node"`
    MaxDisk       float64 `json:"max_disk"`
    MaxDiskNode   string  `json:"max_disk_node"`
    MaxDiskMount  string  `json:"max_disk_mount"`
    WarningNodes  int     `json:"warning_nodes"`
    TotalCores    int     `json:"total_cores"`
    TotalMemBytes uint64  `json:"total_mem_bytes"`
    UsedMemBytes  uint64  `json:"used_mem_bytes"`
}
```

### 3.2 工具类型（atlhyper_metrics_v2/utils/types.go）

```go
package utils

// CPURawStats CPU 原始统计（用于计算差值）
type CPURawStats struct {
    Total uint64   // 总 jiffies
    Idle  uint64   // 空闲 jiffies
    Cores []uint64 // 每核总 jiffies
    Idles []uint64 // 每核空闲 jiffies
}

// DiskRawStats 磁盘原始统计
type DiskRawStats struct {
    Device     string
    ReadBytes  uint64
    WriteBytes uint64
    ReadOps    uint64
    WriteOps   uint64
    IoTicks    uint64 // 用于计算 IOUtil
}

// NetRawStats 网络原始统计
type NetRawStats struct {
    Interface string
    RxBytes   uint64
    TxBytes   uint64
    RxPackets uint64
    TxPackets uint64
}

// ProcRawStats 进程原始统计
type ProcRawStats struct {
    PID   int
    UTime uint64
    STime uint64
}
```

---

## 4. 采集器实现

### 4.1 Collector 接口

```go
// collector/interfaces.go

package collector

import "context"

// Collector 采集器接口
type Collector interface {
    // Name 返回采集器名称
    Name() string
    // Collect 执行采集
    Collect(ctx context.Context) error
}

// CPUCollector CPU 采集器
type CPUCollector interface {
    Collector
    Get() model_v2.CPUMetrics
}

// MemoryCollector 内存采集器
type MemoryCollector interface {
    Collector
    Get() model_v2.MemoryMetrics
}

// DiskCollector 磁盘采集器
type DiskCollector interface {
    Collector
    Get() []model_v2.DiskMetrics
}

// NetworkCollector 网络采集器
type NetworkCollector interface {
    Collector
    Get() []model_v2.NetworkMetrics
}

// TemperatureCollector 温度采集器
type TemperatureCollector interface {
    Collector
    Get() model_v2.TemperatureMetrics
}

// ProcessCollector 进程采集器
type ProcessCollector interface {
    Collector
    Get() []model_v2.ProcessMetrics
}
```

### 4.2 CPU 采集

```go
// collector/cpu.go

package collector

import (
    "bufio"
    "os"
    "runtime"
    "strconv"
    "strings"
    "sync"
    "time"
)

type cpuCollector struct {
    mu       sync.RWMutex
    procRoot string

    // 上一次采样
    lastStats CPURawStats
    lastTime  time.Time

    // 当前结果
    metrics model_v2.CPUMetrics
}

func NewCPUCollector(procRoot string) *cpuCollector {
    c := &cpuCollector{
        procRoot: procRoot,
    }
    // 读取静态信息（型号、主频）
    c.readCPUInfo()
    return c
}

func (c *cpuCollector) Name() string { return "cpu" }

func (c *cpuCollector) Collect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 1. 读取 /proc/stat
    stats, err := c.readProcStat()
    if err != nil {
        return err
    }

    // 2. 计算使用率（需要有上一次采样数据）
    if !c.lastTime.IsZero() {
        c.calculateUsage(stats)
    }

    // 3. 读取 /proc/loadavg
    c.readLoadAvg()

    // 4. 更新基线
    c.lastStats = stats
    c.lastTime = time.Now()

    return nil
}

func (c *cpuCollector) readCPUInfo() {
    // 从 /proc/cpuinfo 读取型号和主频
    f, err := os.Open(c.procRoot + "/cpuinfo")
    if err != nil {
        return
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "model name") {
            parts := strings.SplitN(line, ":", 2)
            if len(parts) == 2 {
                c.metrics.Model = strings.TrimSpace(parts[1])
            }
        } else if strings.HasPrefix(line, "cpu MHz") {
            parts := strings.SplitN(line, ":", 2)
            if len(parts) == 2 {
                if mhz, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err == nil {
                    c.metrics.Frequency = int(mhz)
                }
            }
        }
    }
    c.metrics.CoreCount = runtime.NumCPU()
}

func (c *cpuCollector) readProcStat() (CPURawStats, error) {
    var stats CPURawStats

    f, err := os.Open(c.procRoot + "/stat")
    if err != nil {
        return stats, err
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Fields(line)
        if len(fields) < 5 {
            continue
        }

        if fields[0] == "cpu" {
            // 总 CPU: cpu user nice system idle iowait irq softirq steal guest guest_nice
            for i := 1; i < len(fields); i++ {
                val, _ := strconv.ParseUint(fields[i], 10, 64)
                stats.Total += val
                if i == 4 { // idle
                    stats.Idle = val
                }
            }
        } else if strings.HasPrefix(fields[0], "cpu") {
            // 每核 CPU: cpu0, cpu1, ...
            var coreTotal, coreIdle uint64
            for i := 1; i < len(fields); i++ {
                val, _ := strconv.ParseUint(fields[i], 10, 64)
                coreTotal += val
                if i == 4 {
                    coreIdle = val
                }
            }
            stats.Cores = append(stats.Cores, coreTotal)
            stats.Idles = append(stats.Idles, coreIdle)
        }
    }

    return stats, nil
}

func (c *cpuCollector) readLoadAvg() {
    data, err := os.ReadFile(c.procRoot + "/loadavg")
    if err != nil {
        return
    }
    fields := strings.Fields(string(data))
    if len(fields) >= 3 {
        c.metrics.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
        c.metrics.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
        c.metrics.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
    }
}

func (c *cpuCollector) calculateUsage(current CPURawStats) {
    // 总使用率
    totalDelta := current.Total - c.lastStats.Total
    idleDelta := current.Idle - c.lastStats.Idle
    if totalDelta > 0 {
        c.metrics.UsagePercent = float64(totalDelta-idleDelta) / float64(totalDelta) * 100
    }

    // 每核使用率
    c.metrics.CoreUsages = make([]float64, len(current.Cores))
    for i := range current.Cores {
        if i >= len(c.lastStats.Cores) {
            continue
        }
        coreDelta := current.Cores[i] - c.lastStats.Cores[i]
        coreIdleDelta := current.Idles[i] - c.lastStats.Idles[i]
        if coreDelta > 0 {
            c.metrics.CoreUsages[i] = float64(coreDelta-coreIdleDelta) / float64(coreDelta) * 100
        }
    }
}

func (c *cpuCollector) Get() model_v2.CPUMetrics {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.metrics
}
```

### 4.3 内存采集

```go
// collector/memory.go

package collector

import (
    "bufio"
    "os"
    "strconv"
    "strings"
    "sync"
)

type memoryCollector struct {
    mu       sync.RWMutex
    procRoot string
    metrics  model_v2.MemoryMetrics
}

func NewMemoryCollector(procRoot string) *memoryCollector {
    return &memoryCollector{procRoot: procRoot}
}

func (c *memoryCollector) Name() string { return "memory" }

func (c *memoryCollector) Collect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    f, err := os.Open(c.procRoot + "/meminfo")
    if err != nil {
        return err
    }
    defer f.Close()

    var m model_v2.MemoryMetrics

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }

        // 值单位是 kB
        val, _ := strconv.ParseUint(fields[1], 10, 64)
        valBytes := val * 1024

        switch fields[0] {
        case "MemTotal:":
            m.TotalBytes = valBytes
        case "MemAvailable:":
            m.AvailableBytes = valBytes
        case "SwapTotal:":
            m.SwapTotalBytes = valBytes
        case "SwapFree:":
            m.SwapUsedBytes = m.SwapTotalBytes - valBytes
        case "Cached:":
            m.Cached = valBytes
        case "Buffers:":
            m.Buffers = valBytes
        }
    }

    // 计算已用内存
    m.UsedBytes = m.TotalBytes - m.AvailableBytes

    // 计算使用率
    if m.TotalBytes > 0 {
        m.UsagePercent = float64(m.UsedBytes) / float64(m.TotalBytes) * 100
    }
    if m.SwapTotalBytes > 0 {
        m.SwapUsagePercent = float64(m.SwapUsedBytes) / float64(m.SwapTotalBytes) * 100
    }

    c.metrics = m
    return nil
}

func (c *memoryCollector) Get() model_v2.MemoryMetrics {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.metrics
}
```

### 4.4 磁盘采集（多磁盘 + I/O）

```go
// collector/disk.go

package collector

import (
    "bufio"
    "os"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"
)

type diskCollector struct {
    mu       sync.RWMutex
    procRoot string
    hostRoot string // 宿主机根目录挂载点

    lastStats map[string]DiskRawStats
    lastTime  time.Time

    metrics []model_v2.DiskMetrics
}

func NewDiskCollector(procRoot, hostRoot string) *diskCollector {
    return &diskCollector{
        procRoot:  procRoot,
        hostRoot:  hostRoot,
        lastStats: make(map[string]DiskRawStats),
    }
}

func (c *diskCollector) Name() string { return "disk" }

func (c *diskCollector) Collect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    now := time.Now()

    // 1. 读取挂载点 /proc/mounts
    mounts, err := c.readMounts()
    if err != nil {
        return err
    }

    // 2. 读取 I/O 统计 /proc/diskstats
    ioStats, err := c.readDiskStats()
    if err != nil {
        return err
    }

    // 3. 组装结果
    c.metrics = make([]model_v2.DiskMetrics, 0)

    for _, mount := range mounts {
        // 过滤：只保留真实磁盘分区
        if !c.isRealDisk(mount.Device) {
            continue
        }

        // 获取空间使用
        var statfs syscall.Statfs_t
        mountPath := c.hostRoot + mount.MountPoint
        if err := syscall.Statfs(mountPath, &statfs); err != nil {
            continue
        }

        total := statfs.Blocks * uint64(statfs.Bsize)
        avail := statfs.Bavail * uint64(statfs.Bsize)
        used := total - avail

        dm := model_v2.DiskMetrics{
            Device:         mount.Device,
            MountPoint:     mount.MountPoint,
            FsType:         mount.FsType,
            TotalBytes:     total,
            UsedBytes:      used,
            AvailableBytes: avail,
        }

        if total > 0 {
            dm.UsagePercent = float64(used) / float64(total) * 100
        }

        // 计算 I/O 速率（需要设备名匹配）
        devName := c.extractDevName(mount.Device)
        if io, ok := ioStats[devName]; ok {
            if last, exists := c.lastStats[devName]; exists && !c.lastTime.IsZero() {
                elapsed := now.Sub(c.lastTime).Seconds()
                if elapsed > 0 {
                    dm.ReadBytesPS = float64(io.ReadBytes-last.ReadBytes) / elapsed
                    dm.WriteBytesPS = float64(io.WriteBytes-last.WriteBytes) / elapsed
                    dm.IOPS = float64(io.ReadOps+io.WriteOps-last.ReadOps-last.WriteOps) / elapsed
                    // IOUtil = (io_ticks_delta / elapsed_ms) * 100
                    dm.IOUtil = float64(io.IoTicks-last.IoTicks) / (elapsed * 1000) * 100
                    if dm.IOUtil > 100 {
                        dm.IOUtil = 100
                    }
                }
            }
            c.lastStats[devName] = io
        }

        c.metrics = append(c.metrics, dm)
    }

    c.lastTime = now
    return nil
}

type mountInfo struct {
    Device     string
    MountPoint string
    FsType     string
}

func (c *diskCollector) readMounts() ([]mountInfo, error) {
    f, err := os.Open(c.procRoot + "/mounts")
    if err != nil {
        return nil, err
    }
    defer f.Close()

    var mounts []mountInfo
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        fields := strings.Fields(scanner.Text())
        if len(fields) < 3 {
            continue
        }
        mounts = append(mounts, mountInfo{
            Device:     fields[0],
            MountPoint: fields[1],
            FsType:     fields[2],
        })
    }
    return mounts, nil
}

func (c *diskCollector) readDiskStats() (map[string]DiskRawStats, error) {
    f, err := os.Open(c.procRoot + "/diskstats")
    if err != nil {
        return nil, err
    }
    defer f.Close()

    stats := make(map[string]DiskRawStats)
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        fields := strings.Fields(scanner.Text())
        if len(fields) < 14 {
            continue
        }
        // 字段: major minor name reads_completed reads_merged sectors_read time_reading
        //       writes_completed writes_merged sectors_written time_writing io_in_progress
        //       time_io time_weighted_io
        devName := fields[2]
        readOps, _ := strconv.ParseUint(fields[3], 10, 64)
        sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
        writeOps, _ := strconv.ParseUint(fields[7], 10, 64)
        sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)
        ioTicks, _ := strconv.ParseUint(fields[12], 10, 64)

        stats[devName] = DiskRawStats{
            Device:     devName,
            ReadOps:    readOps,
            WriteOps:   writeOps,
            ReadBytes:  sectorsRead * 512,  // 扇区大小固定 512 bytes
            WriteBytes: sectorsWritten * 512,
            IoTicks:    ioTicks,
        }
    }
    return stats, nil
}

func (c *diskCollector) isRealDisk(device string) bool {
    prefixes := []string{"/dev/sd", "/dev/nvme", "/dev/vd", "/dev/xvd", "/dev/hd"}
    for _, p := range prefixes {
        if strings.HasPrefix(device, p) {
            return true
        }
    }
    return false
}

func (c *diskCollector) extractDevName(device string) string {
    // /dev/sda1 -> sda1, /dev/nvme0n1p1 -> nvme0n1p1
    return strings.TrimPrefix(device, "/dev/")
}

func (c *diskCollector) Get() []model_v2.DiskMetrics {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.metrics
}
```

### 4.5 网络采集（多网卡）

```go
// collector/network.go

package collector

import (
    "bufio"
    "net"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"
)

type networkCollector struct {
    mu       sync.RWMutex
    procRoot string
    sysRoot  string

    lastStats map[string]NetRawStats
    lastTime  time.Time

    metrics []model_v2.NetworkMetrics
}

func NewNetworkCollector(procRoot, sysRoot string) *networkCollector {
    return &networkCollector{
        procRoot:  procRoot,
        sysRoot:   sysRoot,
        lastStats: make(map[string]NetRawStats),
    }
}

func (c *networkCollector) Name() string { return "network" }

func (c *networkCollector) Collect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    now := time.Now()

    // 1. 读取 /proc/net/dev
    devStats, err := c.readNetDev()
    if err != nil {
        return err
    }

    c.metrics = make([]model_v2.NetworkMetrics, 0)

    for iface, stats := range devStats {
        // 过滤虚拟接口
        if c.isVirtualInterface(iface) {
            continue
        }

        nm := model_v2.NetworkMetrics{
            Interface: iface,
            RxErrors:  stats.RxErrors,
            TxErrors:  stats.TxErrors,
            RxDropped: stats.RxDropped,
            TxDropped: stats.TxDropped,
        }

        // 获取 IP/MAC
        nm.IPAddress, nm.MACAddress = c.getAddresses(iface)

        // 获取链路状态和速度
        nm.Status, nm.Speed = c.getLinkInfo(iface)

        // 计算速率
        if last, exists := c.lastStats[iface]; exists && !c.lastTime.IsZero() {
            elapsed := now.Sub(c.lastTime).Seconds()
            if elapsed > 0 {
                nm.RxBytesPS = float64(stats.RxBytes-last.RxBytes) / elapsed
                nm.TxBytesPS = float64(stats.TxBytes-last.TxBytes) / elapsed
                nm.RxPacketsPS = float64(stats.RxPackets-last.RxPackets) / elapsed
                nm.TxPacketsPS = float64(stats.TxPackets-last.TxPackets) / elapsed
            }
        }

        c.lastStats[iface] = stats
        c.metrics = append(c.metrics, nm)
    }

    c.lastTime = now
    return nil
}

type netDevStats struct {
    RxBytes   uint64
    TxBytes   uint64
    RxPackets uint64
    TxPackets uint64
    RxErrors  uint64
    TxErrors  uint64
    RxDropped uint64
    TxDropped uint64
}

func (c *networkCollector) readNetDev() (map[string]netDevStats, error) {
    f, err := os.Open(c.procRoot + "/net/dev")
    if err != nil {
        return nil, err
    }
    defer f.Close()

    stats := make(map[string]netDevStats)
    scanner := bufio.NewScanner(f)
    lineNo := 0
    for scanner.Scan() {
        lineNo++
        if lineNo <= 2 { // 跳过头部
            continue
        }
        line := scanner.Text()
        parts := strings.SplitN(line, ":", 2)
        if len(parts) != 2 {
            continue
        }
        iface := strings.TrimSpace(parts[0])
        fields := strings.Fields(parts[1])
        if len(fields) < 16 {
            continue
        }

        // 字段顺序: rx_bytes rx_packets rx_errs rx_drop rx_fifo rx_frame rx_compressed rx_multicast
        //          tx_bytes tx_packets tx_errs tx_drop tx_fifo tx_colls tx_carrier tx_compressed
        rxBytes, _ := strconv.ParseUint(fields[0], 10, 64)
        rxPackets, _ := strconv.ParseUint(fields[1], 10, 64)
        rxErrors, _ := strconv.ParseUint(fields[2], 10, 64)
        rxDropped, _ := strconv.ParseUint(fields[3], 10, 64)
        txBytes, _ := strconv.ParseUint(fields[8], 10, 64)
        txPackets, _ := strconv.ParseUint(fields[9], 10, 64)
        txErrors, _ := strconv.ParseUint(fields[10], 10, 64)
        txDropped, _ := strconv.ParseUint(fields[11], 10, 64)

        stats[iface] = netDevStats{
            RxBytes:   rxBytes,
            TxBytes:   txBytes,
            RxPackets: rxPackets,
            TxPackets: txPackets,
            RxErrors:  rxErrors,
            TxErrors:  txErrors,
            RxDropped: rxDropped,
            TxDropped: txDropped,
        }
    }
    return stats, nil
}

func (c *networkCollector) isVirtualInterface(name string) bool {
    virtuals := []string{"lo", "docker", "veth", "br-", "virbr", "cni", "flannel", "calico"}
    for _, v := range virtuals {
        if strings.HasPrefix(name, v) {
            return true
        }
    }
    return false
}

func (c *networkCollector) getAddresses(iface string) (ip, mac string) {
    intf, err := net.InterfaceByName(iface)
    if err != nil {
        return
    }
    mac = intf.HardwareAddr.String()

    addrs, err := intf.Addrs()
    if err != nil {
        return
    }
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
            ip = ipnet.IP.String()
            break
        }
    }
    return
}

func (c *networkCollector) getLinkInfo(iface string) (status string, speed int) {
    // 读取 /sys/class/net/{iface}/operstate
    data, err := os.ReadFile(c.sysRoot + "/class/net/" + iface + "/operstate")
    if err == nil {
        status = strings.TrimSpace(string(data))
    }

    // 读取 /sys/class/net/{iface}/speed
    data, err = os.ReadFile(c.sysRoot + "/class/net/" + iface + "/speed")
    if err == nil {
        speed, _ = strconv.Atoi(strings.TrimSpace(string(data)))
    }

    return
}

func (c *networkCollector) Get() []model_v2.NetworkMetrics {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.metrics
}
```

### 4.6 温度采集

```go
// collector/temperature.go

package collector

import (
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
)

type temperatureCollector struct {
    mu      sync.RWMutex
    sysRoot string
    metrics model_v2.TemperatureMetrics
}

func NewTemperatureCollector(sysRoot string) *temperatureCollector {
    return &temperatureCollector{sysRoot: sysRoot}
}

func (c *temperatureCollector) Name() string { return "temperature" }

func (c *temperatureCollector) Collect(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    var m model_v2.TemperatureMetrics
    m.Sensors = make([]model_v2.SensorReading, 0)

    // 遍历 /sys/class/hwmon/hwmon*
    hwmonDir := c.sysRoot + "/class/hwmon"
    entries, err := os.ReadDir(hwmonDir)
    if err != nil {
        return err
    }

    for _, entry := range entries {
        hwmonPath := filepath.Join(hwmonDir, entry.Name())

        // 读取传感器名称
        nameData, err := os.ReadFile(filepath.Join(hwmonPath, "name"))
        if err != nil {
            continue
        }
        sensorName := strings.TrimSpace(string(nameData))

        // 读取所有温度传感器
        files, _ := os.ReadDir(hwmonPath)
        for _, f := range files {
            fname := f.Name()
            if !strings.HasPrefix(fname, "temp") || !strings.HasSuffix(fname, "_input") {
                continue
            }

            // temp1_input -> 1
            idx := strings.TrimPrefix(fname, "temp")
            idx = strings.TrimSuffix(idx, "_input")

            // 读取温度值
            tempData, err := os.ReadFile(filepath.Join(hwmonPath, fname))
            if err != nil {
                continue
            }
            milliDeg, _ := strconv.Atoi(strings.TrimSpace(string(tempData)))
            temp := float64(milliDeg) / 1000.0

            // 读取标签
            label := sensorName
            labelData, err := os.ReadFile(filepath.Join(hwmonPath, "temp"+idx+"_label"))
            if err == nil {
                label = strings.TrimSpace(string(labelData))
            }

            // 读取阈值
            var high, crit *float64
            if data, err := os.ReadFile(filepath.Join(hwmonPath, "temp"+idx+"_max")); err == nil {
                v := float64(mustAtoi(string(data))) / 1000.0
                high = &v
            }
            if data, err := os.ReadFile(filepath.Join(hwmonPath, "temp"+idx+"_crit")); err == nil {
                v := float64(mustAtoi(string(data))) / 1000.0
                crit = &v
            }

            sensor := model_v2.SensorReading{
                Name:     sensorName,
                Label:    label,
                Temp:     temp,
                High:     high,
                Critical: crit,
            }
            m.Sensors = append(m.Sensors, sensor)

            // 识别 CPU 温度
            if (sensorName == "coretemp" || sensorName == "k10temp") && m.CPUTemp == 0 {
                m.CPUTemp = temp
                if high != nil {
                    m.CPUTempMax = *high
                }
            }
        }
    }

    // 回退：thermal_zone（树莓派等）
    if m.CPUTemp == 0 {
        c.readThermalZone(&m)
    }

    c.metrics = m
    return nil
}

func (c *temperatureCollector) readThermalZone(m *model_v2.TemperatureMetrics) {
    base := c.sysRoot + "/class/thermal"
    entries, err := os.ReadDir(base)
    if err != nil {
        return
    }
    for _, e := range entries {
        if !strings.HasPrefix(e.Name(), "thermal_zone") {
            continue
        }
        zonePath := filepath.Join(base, e.Name())

        typeData, err := os.ReadFile(filepath.Join(zonePath, "type"))
        if err != nil {
            continue
        }
        typ := strings.TrimSpace(string(typeData))

        if typ == "cpu-thermal" || typ == "cpu_thermal" || typ == "soc-thermal" {
            tempData, err := os.ReadFile(filepath.Join(zonePath, "temp"))
            if err != nil {
                continue
            }
            milliDeg, _ := strconv.Atoi(strings.TrimSpace(string(tempData)))
            m.CPUTemp = float64(milliDeg) / 1000.0
            break
        }
    }
}

func mustAtoi(s string) int {
    v, _ := strconv.Atoi(strings.TrimSpace(s))
    return v
}

func (c *temperatureCollector) Get() model_v2.TemperatureMetrics {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.metrics
}
```

---

## 5. 配置

### 5.1 配置结构

```go
// config/types.go

package config

import "time"

type Config struct {
    NodeName  string        `yaml:"node_name"`  // 节点名（从环境变量获取）
    AgentAddr string        `yaml:"agent_addr"` // Agent 地址 (默认 localhost:8082)
    Interval  time.Duration `yaml:"interval"`   // 推送间隔 (默认 5s)

    Paths struct {
        ProcRoot string `yaml:"proc_root"` // /proc 挂载点 (默认 /host_proc)
        SysRoot  string `yaml:"sys_root"`  // /sys 挂载点 (默认 /host_sys)
        HostRoot string `yaml:"host_root"` // 宿主机根目录 (默认 /host_root)
    } `yaml:"paths"`

    Collect struct {
        TopProcesses int `yaml:"top_processes"` // Top 进程数 (默认 10)
    } `yaml:"collect"`
}
```

### 5.2 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `NODE_NAME` | 节点名称 | 必须 (通过 Downward API) |
| `AGENT_ADDR` | Agent 地址 | `atlhyper-agent.atlhyper.svc.cluster.local:8082` |
| `METRICS_INTERVAL` | 推送间隔 | `5s` |
| `PROC_ROOT` | /proc 挂载点 | `/host_proc` |
| `SYS_ROOT` | /sys 挂载点 | `/host_sys` |
| `HOST_ROOT` | 宿主机根目录 | `/host_root` |

---

## 6. 部署

> 注：部署配置（DaemonSet、Dockerfile）由项目统一管理，不在本模块目录下。以下为参考配置。

### 6.1 DaemonSet 配置（参考）

```yaml
# 统一部署目录中的配置

apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: atlhyper-metrics
  namespace: atlhyper
  labels:
    app: atlhyper-metrics
spec:
  selector:
    matchLabels:
      app: atlhyper-metrics
  template:
    metadata:
      labels:
        app: atlhyper-metrics
    spec:
      hostPID: true        # 访问宿主机进程

      tolerations:
        - operator: Exists  # 容忍所有污点，确保每个节点都运行

      containers:
        - name: metrics
          image: atlhyper/metrics:v2
          imagePullPolicy: IfNotPresent

          env:
            - name: NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: AGENT_ADDR
              value: "atlhyper-agent.atlhyper.svc.cluster.local:8082"  # 通过 Service 访问单实例 Agent

          volumeMounts:
            - name: proc
              mountPath: /host_proc
              readOnly: true
            - name: sys
              mountPath: /host_sys
              readOnly: true
            - name: root
              mountPath: /host_root
              readOnly: true

          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 128Mi

      volumes:
        - name: proc
          hostPath:
            path: /proc
        - name: sys
          hostPath:
            path: /sys
        - name: root
          hostPath:
            path: /
```

**说明：**
- `hostNetwork: false`（默认）— 不需要宿主机网络，通过 K8s Service 访问 Agent
- `AGENT_ADDR` — 使用 Agent 的 Service DNS 名称
- Agent 是单实例部署，通过 Service 暴露端口 8082

---

## 7. 与 Agent 对接

### 7.1 推送接口

```
POST /metrics/node
Content-Type: application/json

{
  "node_name": "k8s-worker-01",
  "timestamp": "2024-01-20T10:30:00Z",
  "cpu": { ... },
  "memory": { ... },
  "disks": [ ... ],
  "networks": [ ... ],
  "temperature": { ... },
  "top_processes": [ ... ]
}
```

### 7.2 Agent 端接收

Agent 需要新增接收端点（见 `docs/design/node-metrics-agent.md`）。

**Agent 部署说明：**
- Agent 采用单实例部署（Deployment，1 副本）
- 通过 Service `atlhyper-agent:8082` 暴露接收端口
- 各节点的 Metrics DaemonSet 通过 Service DNS 访问 Agent

---

## 8. 复用 v1 代码

以下 v1 代码可以复用，但需要调整：

| v1 文件 | 复用内容 | 需调整 |
|---------|----------|--------|
| `collect/cpu.go` | 后台采样 + TopK 逻辑 | 添加每核使用率、型号、主频 |
| `collect/memory.go` | meminfo 解析 | 添加 Swap、Cached、Buffers |
| `collect/disk.go` | Statfs 调用 | 多磁盘 + I/O 统计 |
| `collect/network.go` | net/dev 解析 | 多网卡 + 完整统计 |
| `collect/temperature.go` | hwmon 解析 | 传感器详情 |
| `config/loader.go` | 配置加载 | 新配置结构 |
| `push/reporter.go` | HTTP 推送 | 更新接口路径 |

---

## 9. 后续扩展

### 9.1 容器级指标

采集 cgroup 级别的资源使用（CPU、内存），用于 Pod 级别监控。

### 9.2 自定义采集器

支持插件化采集器，允许用户扩展采集内容。

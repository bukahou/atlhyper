# 节点指标 TDD 规范

> 状态：**权威文档** — 所有开发和测试以本文档为准
> 创建：2026-02-11
> 数据来源：真实集群 OTel Collector :8889 + node_exporter :9100

本文档是节点指标改造的 TDD（测试驱动开发）权威规范。包含：
1. 真实数据样本（测试输入）
2. Go 模型变更（代码结构）
3. 解析器/转换器测试用例（预期输出）
4. 过滤规则（数据清洗逻辑）
5. API 响应格式（接口合约）

---

## 1. 真实数据 vs 设计假设 — 差异总结

| 项目 | 设计假设 | 真实情况 | 影响 |
|------|----------|----------|------|
| `node_cpu_info` | 存在，可获取 CPU 型号 | **不存在**（node_exporter v1.9.0 无此指标） | CPU.Model 为空，核数从 cpu_seconds_total 推导 |
| `node_tcp_connection_states` | 存在，可获取详细 TCP 状态 | **不存在** | TCP 状态只有 CurrEstab/tw/orphan/alloc |
| PSI 指标 | 可直接获取 10s/60s/300s 百分比 | **只有累积 counter**（seconds_total） | 需 rate 计算得到近似百分比 |
| 文件系统 | 干净的磁盘列表 | **大量 shm/tmpfs 噪音**（k3s 容器沙箱） | 必须过滤，只保留 `/dev/` 开头的真实设备 |
| 网络接口 | 物理接口为主 | **大量 veth/flannel/cni 虚拟接口** | 必须过滤，只保留物理接口 |
| desk-zero CPU | 6C/6T (i5-8500) | **8 个逻辑核**（cpu 0-7） | 核数从数据推导，不硬编码 |
| OTel 前缀 | `otel_` | 确认 `otel_` 前缀 | 解析器需先去除 `otel_` 前缀 |
| `node_hwmon_temp_crit_celsius` | 未考虑 | **存在**（node_exporter 有，但白名单未包含） | 需补充到 OTel 白名单 |
| NodeMetricsSnapshot | 不改动 | **需要扩展**（新增 PSI/TCP/System/VMStat/NTP/Softnet） | 改变 Agent↔Master 数据合约 |

---

## 2. 节点清单与 IP 映射

| 节点 | IP | Instance (OTel label) | 架构 | 内核 | 角色 |
|------|----|-----------------------|------|------|------|
| desk-zero | 192.168.0.130 | `192.168.0.130:9100` | x86_64 | 6.8.0-85-generic | control-plane |
| desk-one | 192.168.0.7 | `192.168.0.7:9100` | x86_64 | 6.8.0-85-generic | worker |
| desk-two | 192.168.0.46 | `192.168.0.46:9100` | x86_64 | 6.8.0-88-generic | worker |
| raspi-zero | 192.168.0.182 | `192.168.0.182:9100` | aarch64 | 6.8.0-1043-raspi | worker |
| raspi-one | 192.168.0.33 | `192.168.0.33:9100` | aarch64 | 6.8.0-1040-raspi | worker |
| raspi-nfs | 192.168.0.153 | `192.168.0.153:9100` | aarch64 | 6.8.0-1043-raspi | worker |

**NodeName 提取方式**：从 `otel_node_uname_info{nodename="desk-zero",...}` 的 `nodename` label 获取。

---

## 3. 真实 Prometheus 原始样本

### 3.1 desk-zero (amd64 代表 — 192.168.0.130)

> 以下为从 node_exporter :9100 直接采集的数据。
> 经 OTel Collector 后，所有指标名添加 `otel_` 前缀，标签增加 `instance` 和 `job`。

#### 系统信息
```
node_uname_info{domainname="(none)",machine="x86_64",nodename="desk-zero",release="6.8.0-85-generic",sysname="Linux",version="#85-Ubuntu SMP PREEMPT_DYNAMIC Thu Sep 18 15:26:59 UTC 2025"} 1
node_boot_time_seconds 1.759572263e+09
```

#### CPU (8 核, counter)
```
node_cpu_seconds_total{cpu="0",mode="idle"} 1.066988737e+07
node_cpu_seconds_total{cpu="0",mode="iowait"} 937.82
node_cpu_seconds_total{cpu="0",mode="irq"} 0
node_cpu_seconds_total{cpu="0",mode="nice"} 95.58
node_cpu_seconds_total{cpu="0",mode="softirq"} 3567.51
node_cpu_seconds_total{cpu="0",mode="steal"} 0
node_cpu_seconds_total{cpu="0",mode="system"} 113450.68
node_cpu_seconds_total{cpu="0",mode="user"} 325095.17
node_cpu_seconds_total{cpu="1",mode="idle"} 1.068617648e+07
node_cpu_seconds_total{cpu="1",mode="user"} 326508.13
node_cpu_seconds_total{cpu="1",mode="system"} 114553.23
... (8 核 × 8 mode = 64 行)

node_cpu_scaling_frequency_hertz{cpu="0"} 1.600893e+09
node_cpu_scaling_frequency_max_hertz{cpu="0"} 3.8e+09
... (8 核)

node_load1 0.56
node_load5 0.41
node_load15 0.28
```

#### 内存 (gauge)
```
node_memory_MemTotal_bytes 3.3528840192e+10
node_memory_MemAvailable_bytes 2.9597294592e+10
node_memory_MemFree_bytes 1.8630926336e+10
node_memory_Cached_bytes 9.811787776e+09
node_memory_Buffers_bytes 9.61503232e+08
node_memory_SwapTotal_bytes 0
node_memory_SwapFree_bytes 0
```

#### 文件系统 (gauge) — 过滤后
```
node_filesystem_size_bytes{device="/dev/mapper/ubuntu--vg-ubuntu--lv",fstype="ext4",mountpoint="/"} 1.05089261568e+11
node_filesystem_avail_bytes{device="/dev/mapper/ubuntu--vg-ubuntu--lv",fstype="ext4",mountpoint="/"} 8.7484624896e+10
node_filesystem_size_bytes{device="/dev/sda2",fstype="ext4",mountpoint="/boot"} 2.040373248e+09
node_filesystem_avail_bytes{device="/dev/sda2",fstype="ext4",mountpoint="/boot"} 1.71165696e+09
node_filesystem_size_bytes{device="/dev/sda1",fstype="vfat",mountpoint="/boot/efi"} 1.124995072e+09
node_filesystem_avail_bytes{device="/dev/sda1",fstype="vfat",mountpoint="/boot/efi"} 1.11855616e+09
```
> 原始数据还有约 30 个 `shm` tmpfs 条目（k3s 容器沙箱），解析器必须过滤掉。

#### 磁盘 I/O (counter)
```
node_disk_read_bytes_total{device="sda"} 4.22935286272e+11
node_disk_written_bytes_total{device="sda"} 8.39274673152e+11
node_disk_reads_completed_total{device="sda"} 1.0320305e+07
node_disk_writes_completed_total{device="sda"} 3.4065747e+07
node_disk_io_time_seconds_total{device="sda"} 12199.301
node_disk_read_bytes_total{device="dm-0"} 4.21990759424e+11
node_disk_written_bytes_total{device="dm-0"} 8.383116288e+11
node_disk_reads_completed_total{device="dm-0"} 1.0316399e+07
node_disk_writes_completed_total{device="dm-0"} 5.574318e+07
node_disk_io_time_seconds_total{device="dm-0"} 62638.450000000004
```

#### 网络 (counter/gauge) — 物理接口
```
node_network_up{device="eno1"} 1
node_network_speed_bytes{device="eno1"} 1.25e+08
node_network_mtu_bytes{device="eno1"} 1500
node_network_receive_bytes_total{device="eno1"} 2.1642817978e+11
node_network_transmit_bytes_total{device="eno1"} 7.65387654753e+11
node_network_receive_packets_total{device="eno1"} 8.09049876e+08
node_network_transmit_packets_total{device="eno1"} 1.033760094e+09
node_network_receive_errs_total{device="eno1"} 0
node_network_transmit_errs_total{device="eno1"} 0
node_network_receive_drop_total{device="eno1"} 5.614329e+06
node_network_transmit_drop_total{device="eno1"} 0
```
> 原始数据还有 cni0, flannel.1, lo, veth* 等虚拟接口（约 30 个），解析器必须过滤。

#### 温度 (gauge)
```
node_hwmon_temp_celsius{chip="platform_coretemp_0",sensor="temp1"} 49
node_hwmon_temp_celsius{chip="platform_coretemp_0",sensor="temp2"} 47
node_hwmon_temp_celsius{chip="platform_coretemp_0",sensor="temp3"} 47
node_hwmon_temp_celsius{chip="platform_coretemp_0",sensor="temp4"} 49
node_hwmon_temp_celsius{chip="platform_coretemp_0",sensor="temp5"} 47
node_hwmon_temp_max_celsius{chip="platform_coretemp_0",sensor="temp1"} 74
node_hwmon_temp_crit_celsius{chip="platform_coretemp_0",sensor="temp1"} 80
```

#### PSI (counter)
```
node_pressure_cpu_waiting_seconds_total 110989.082491
node_pressure_io_stalled_seconds_total 1923.752212
node_pressure_io_waiting_seconds_total 2038.066703
node_pressure_memory_stalled_seconds_total 0.454101
node_pressure_memory_waiting_seconds_total 0.542025
```

#### TCP/Socket (gauge)
```
node_netstat_Tcp_CurrEstab 138
node_sockstat_TCP_alloc 468
node_sockstat_TCP_orphan 0
node_sockstat_TCP_tw 49
node_sockstat_TCP_inuse 69
node_sockstat_sockets_used 782
```

#### 系统资源 (gauge)
```
node_nf_conntrack_entries 7582
node_nf_conntrack_entries_limit 262144
node_filefd_allocated 3840
node_filefd_maximum 9.223372036854776e+18
node_entropy_available_bits 256
```

#### VMStat (counter/gauge)
```
node_vmstat_pgfault 4.434005615e+09
node_vmstat_pgmajfault 5310
node_vmstat_pswpin 0
node_vmstat_pswpout 0
```

#### NTP (gauge)
```
node_timex_offset_seconds 0.000168911
node_timex_sync_status 1
```

#### Softnet (counter, per-cpu)
```
node_softnet_dropped_total{cpu="0"} 0
... (8 核全为 0)
node_softnet_times_squeezed_total{cpu="0"} 2
node_softnet_times_squeezed_total{cpu="5"} 195
... (cpu5 明显高于其他核)
```

### 3.2 raspi-zero (arm64 代表 — 192.168.0.182)

#### 系统信息
```
node_uname_info{domainname="(none)",machine="aarch64",nodename="raspi-zero",release="6.8.0-1043-raspi",sysname="Linux",version="#47-Ubuntu SMP PREEMPT_DYNAMIC Fri Oct 17 22:33:56 UTC 2025"} 1
node_boot_time_seconds 1.766307381e+09
```

#### CPU (4 核)
```
node_cpu_seconds_total{cpu="0",mode="idle"} 4.15087529e+06
node_cpu_seconds_total{cpu="0",mode="iowait"} 42325.23
node_cpu_seconds_total{cpu="0",mode="system"} 83843.33
node_cpu_seconds_total{cpu="0",mode="user"} 194517.06
... (4 核 × 8 mode = 32 行)

node_cpu_scaling_frequency_hertz{cpu="0"} 2.4e+09
node_cpu_scaling_frequency_max_hertz{cpu="0"} 2.4e+09
... (4 核，固定 2.4GHz)

node_load1 0.44
node_load5 0.63
node_load15 0.52
```

#### 内存
```
node_memory_MemTotal_bytes 8.323080192e+09
node_memory_MemAvailable_bytes 6.389919744e+09
node_memory_MemFree_bytes 4.56372224e+08
node_memory_Cached_bytes 5.634650112e+09
node_memory_Buffers_bytes 1.68144896e+08
node_memory_SwapTotal_bytes 0
node_memory_SwapFree_bytes 0
```

#### 文件系统 — 过滤后
```
node_filesystem_size_bytes{device="/dev/nvme0n1p2",fstype="ext4",mountpoint="/"} 1.25470957568e+11
node_filesystem_avail_bytes{device="/dev/nvme0n1p2",fstype="ext4",mountpoint="/"} 9.8538672128e+10
node_filesystem_size_bytes{device="/dev/nvme0n1p1",fstype="vfat",mountpoint="/boot/firmware"} 5.28592896e+08
node_filesystem_avail_bytes{device="/dev/nvme0n1p1",fstype="vfat",mountpoint="/boot/firmware"} 3.33590016e+08
```

#### 磁盘 I/O
```
node_disk_read_bytes_total{device="nvme0n1"} 2.37674955264e+11
node_disk_written_bytes_total{device="nvme0n1"} 2.37861910016e+11
node_disk_reads_completed_total{device="nvme0n1"} 6.320463e+06
node_disk_writes_completed_total{device="nvme0n1"} 1.1141371e+07
node_disk_io_time_seconds_total{device="nvme0n1"} 27152.524
```

#### 网络 — 物理接口
```
node_network_up{device="eth0"} 1
node_network_up{device="wlan0"} 0
node_network_speed_bytes{device="eth0"} 1.25e+08
node_network_mtu_bytes{device="eth0"} 1500
node_network_receive_bytes_total{device="eth0"} 4.9443211363e+10
node_network_transmit_bytes_total{device="eth0"} 3.1131711667e+10
```

#### 温度 — arm64 特殊
```
node_hwmon_temp_celsius{chip="1000120000_pcie_1f000c8000_adc",sensor="temp1"} 56.053
node_hwmon_temp_celsius{chip="nvme_nvme0",sensor="temp1"} 34.85
node_hwmon_temp_celsius{chip="nvme_nvme0",sensor="temp2"} 35.85
node_hwmon_temp_celsius{chip="thermal_thermal_zone0",sensor="temp0"} 53.45
node_hwmon_temp_celsius{chip="thermal_thermal_zone0",sensor="temp1"} 53.45
node_hwmon_temp_max_celsius{chip="nvme_nvme0",sensor="temp1"} 81.85
node_hwmon_temp_crit_celsius{chip="nvme_nvme0",sensor="temp1"} 84.85
```

> **arm64 温度特征**：无 `platform_coretemp_0` 芯片，SoC 温度来自 `adc` 或 `thermal_zone`。

#### PSI (counter)
```
node_pressure_cpu_waiting_seconds_total 40021.859884
node_pressure_io_stalled_seconds_total 47113.340769
node_pressure_io_waiting_seconds_total 48685.192158
node_pressure_memory_stalled_seconds_total 0.640171
node_pressure_memory_waiting_seconds_total 0.656567
```

### 3.3 全节点关键 Gauge 指标对照表

| 指标 | desk-zero | desk-one | desk-two | raspi-zero | raspi-one | raspi-nfs |
|------|-----------|----------|----------|------------|-----------|-----------|
| **MemTotal (GB)** | 31.2 | 31.2 | 31.2 | 7.8 | 7.8 | 7.8 |
| **MemAvailable (GB)** | 27.5 | 25.5 | 26.8 | 6.0 | 6.3 | 6.4 |
| **Load1** | 0.56 | 0.86 | 0.13 | 0.44 | 0.61 | 0.55 |
| **Tcp CurrEstab** | 138 | 55 | 26 | 62 | 22 | 24 |
| **TCP tw** | 49 | 116 | 23 | 31 | 30 | 27 |
| **TCP alloc** | 468 | 374 | 120 | 159 | 127 | 119 |
| **TCP orphan** | 0 | 0 | 0 | 0 | 0 | 0 |
| **Sockets used** | 782 | 501 | 342 | 386 | 340 | 354 |
| **Conntrack** | 7582 | 1637 | 1655 | 1034 | 1567 | 1306 |
| **Conntrack limit** | 262144 | 262144 | 262144 | 131072 | 131072 | 131072 |
| **Filefd alloc** | 3840 | 4512 | 5856 | 2432 | 2336 | 2304 |
| **Entropy** | 256 | 256 | 256 | 256 | 256 | 256 |
| **NTP synced** | 1 | 1 | 1 | 1 | 1 | 1 |
| **SwapTotal** | 0 | 0 | 0 | 0 | 0 | 0 |

---

## 4. Go 模型变更规范

### 4.1 NodeMetricsSnapshot 扩展

在 `model_v2/node_metrics.go` 新增以下字段（**非破坏性变更**，JSON 反序列化向后兼容）：

```go
type NodeMetricsSnapshot struct {
    // ... 现有字段全部保留 ...

    // === 新增字段 (Phase 2) ===
    PSI      PSIMetrics      `json:"psi"`       // 压力信息
    TCP      TCPMetrics      `json:"tcp"`       // TCP 连接状态
    System   SystemMetrics   `json:"system"`    // 系统资源
    VMStat   VMStatMetrics   `json:"vmstat"`    // 虚拟内存统计
    NTP      NTPMetrics      `json:"ntp"`       // 时间同步
    Softnet  SoftnetMetrics  `json:"softnet"`   // 软中断统计
}
```

### 4.2 新增结构体定义

```go
// PSIMetrics 压力信息 (Pressure Stall Information)
// 值为 rate 计算得到的近似百分比 (0-100)
// 表示在上一个采集间隔内，任务因资源不足而等待的时间占比
type PSIMetrics struct {
    CPUSomePercent    float64 `json:"cpu_some_percent"`    // CPU 部分阻塞 %
    MemorySomePercent float64 `json:"memory_some_percent"` // 内存部分阻塞 %
    MemoryFullPercent float64 `json:"memory_full_percent"` // 内存完全阻塞 %
    IOSomePercent     float64 `json:"io_some_percent"`     // IO 部分阻塞 %
    IOFullPercent     float64 `json:"io_full_percent"`     // IO 完全阻塞 %
}

// TCPMetrics TCP 连接状态
type TCPMetrics struct {
    CurrEstab   int64 `json:"curr_estab"`   // 当前 ESTABLISHED 连接数
    TimeWait    int64 `json:"time_wait"`    // TIME_WAIT 数
    Orphan      int64 `json:"orphan"`       // 孤儿连接数
    Alloc       int64 `json:"alloc"`        // 已分配 TCP socket 数
    InUse       int64 `json:"in_use"`       // 正在使用的 TCP socket 数
    SocketsUsed int64 `json:"sockets_used"` // 全局 socket 使用数
}

// SystemMetrics 系统资源指标
type SystemMetrics struct {
    ConntrackEntries int64 `json:"conntrack_entries"` // 连接跟踪条目数
    ConntrackLimit   int64 `json:"conntrack_limit"`   // 连接跟踪上限
    FilefdAllocated  int64 `json:"filefd_allocated"`  // 已分配文件描述符数
    FilefdMaximum    int64 `json:"filefd_maximum"`    // 文件描述符上限
    EntropyAvailable int64 `json:"entropy_available"` // 可用熵 (bits)
}

// VMStatMetrics 虚拟内存统计 (rate, 每秒)
type VMStatMetrics struct {
    PgFaultPS    float64 `json:"pgfault_ps"`    // 页错误/秒
    PgMajFaultPS float64 `json:"pgmajfault_ps"` // 主页错误/秒
    PswpInPS     float64 `json:"pswpin_ps"`     // Swap 换入/秒
    PswpOutPS    float64 `json:"pswpout_ps"`    // Swap 换出/秒
}

// NTPMetrics 时间同步指标
type NTPMetrics struct {
    OffsetSeconds float64 `json:"offset_seconds"` // 时间偏移 (秒)
    Synced        bool    `json:"synced"`         // 是否已同步
}

// SoftnetMetrics 软中断统计 (累积值, 所有 CPU 求和)
type SoftnetMetrics struct {
    Dropped  int64 `json:"dropped"`  // 丢弃的包总数
    Squeezed int64 `json:"squeezed"` // 被挤压次数总数
}
```

### 4.3 OTelNodeRawMetrics 最终定义（基于真实数据调整）

```go
// sdk/types.go 中的 OTelNodeRawMetrics
type OTelNodeRawMetrics struct {
    NodeName string
    Instance string

    // CPU (counter: cpu×mode → seconds)
    CPUSecondsTotal map[string]float64  // key: "0:idle", "0:user", "1:system"...
    CPUCoreCount    int                 // 从 cpu label 去重计数

    // CPU 频率 (gauge: cpu → Hz)
    CPUFreqHertz    map[string]float64  // key: cpu label
    CPUFreqMaxHertz float64             // 所有核最大频率

    // Load (gauge)
    Load1, Load5, Load15 float64

    // Memory (gauge, bytes)
    MemTotal, MemAvailable, MemFree int64
    MemCached, MemBuffers           int64
    SwapTotal, SwapFree             int64

    // Filesystem (gauge)
    Filesystems []FSRawMetrics

    // Disk I/O (counter)
    DiskIO []DiskIORawMetrics

    // Network (counter/gauge)
    Networks []NetRawMetrics

    // Temperature (gauge)
    HWMonTemps []HWMonRawTemp

    // PSI (counter, seconds)
    PSICPUWaiting     float64
    PSIMemoryWaiting  float64
    PSIMemoryStalled  float64
    PSIIOWaiting      float64
    PSIIOStalled      float64

    // TCP/Socket (gauge)
    TCPCurrEstab   int64
    TCPTimeWait    int64
    TCPOrphan      int64
    TCPAlloc       int64
    TCPInUse       int64
    SocketsUsed    int64

    // System (gauge)
    ConntrackEntries int64
    ConntrackLimit   int64
    FilefdAllocated  int64
    FilefdMaximum    int64
    EntropyBits      int64

    // VMStat (counter)
    PgFault    float64
    PgMajFault float64
    PswpIn     float64
    PswpOut    float64

    // NTP (gauge)
    TimexOffsetSeconds float64
    TimexSyncStatus    float64  // 1=synced, 0=not

    // Softnet (counter, 所有 CPU 已求和)
    SoftnetDropped  int64
    SoftnetSqueezed int64

    // System info (from uname_info)
    Machine  string  // "x86_64" | "aarch64"
    Hostname string  // nodename label
    Kernel   string  // release label
    BootTime float64 // unix timestamp
}
```

**与原设计的变化**：
- 移除 `CPUModel`（node_exporter 无此数据）
- 移除 `OS`（uname_info 只有 sysname="Linux"，信息不足，改用 Machine 判断架构）
- `CPUSecondsTotal` key 格式改为 `"cpu:mode"` 更简洁
- 新增 PSI/TCP/System/VMStat/NTP/Softnet 所有字段
- `CPUFreqHertz` 改为 map 存每核频率

---

## 5. 过滤规则

### 5.1 文件系统过滤

**规则**：只保留 device 以 `/dev/` 开头的条目。

```go
func shouldKeepFilesystem(device, fstype, mountpoint string) bool {
    return strings.HasPrefix(device, "/dev/")
}
```

**过滤效果**：

| 节点 | 原始条目数 | 过滤后 | 保留的 |
|------|-----------|--------|--------|
| desk-zero | ~20 | 3 | `/`(ext4), `/boot`(ext4), `/boot/efi`(vfat) |
| raspi-zero | ~12 | 2 | `/`(ext4), `/boot/firmware`(vfat) |
| raspi-nfs | ~15 | 4 | `/`(ext4), `/boot/firmware`(vfat), `/media/.../DATA2T`(ext4), `/media/.../DATA`(ext4) |

### 5.2 网络接口过滤

**规则**：排除虚拟接口，只保留物理/已知接口。

```go
func shouldKeepNetwork(device string) bool {
    // 排除规则
    switch {
    case device == "lo":
        return false
    case strings.HasPrefix(device, "veth"):
        return false
    case strings.HasPrefix(device, "flannel"):
        return false
    case strings.HasPrefix(device, "cni"):
        return false
    case strings.HasPrefix(device, "cali"):
        return false
    }
    return true
}
```

**过滤效果**：

| 节点 | 原始接口数 | 过滤后 | 保留的 |
|------|-----------|--------|--------|
| desk-zero | ~15 | 1 | eno1 |
| raspi-zero | ~8 | 2 | eth0, wlan0 |

### 5.3 磁盘 I/O 过滤

**规则**：排除 `dm-*`（device-mapper）设备，避免与底层物理设备重复计算。

```go
func shouldKeepDiskIO(device string) bool {
    return !strings.HasPrefix(device, "dm-")
}
```

desk-zero 有 `sda` 和 `dm-0`，两者数据几乎相同（dm-0 是 sda 的 LVM 映射），保留 `sda`。

---

## 6. 解析器测试规范 (node_parser_test.go)

### 6.1 测试输入格式

测试输入为 OTel Collector 输出的 Prometheus 文本（带 `otel_` 前缀和 `instance`/`job` label）：

```
otel_node_uname_info{instance="192.168.0.130:9100",job="node-exporter",machine="x86_64",nodename="desk-zero",release="6.8.0-85-generic",...} 1
otel_node_boot_time_seconds{instance="192.168.0.130:9100",job="node-exporter"} 1.759572263e+09
otel_node_cpu_seconds_total{cpu="0",instance="192.168.0.130:9100",job="node-exporter",mode="idle"} 1.066988737e+07
...
```

### 6.2 测试用例

#### Test_ParseNodeMetrics_DeskZero

**输入**：desk-zero 的完整 OTel 指标文本（约 400 行，含 otel_ 前缀）

**断言**：
```go
result := parseNodeMetrics(otelText)
node := result["desk-zero"]

// 基础信息
assert.Equal("desk-zero", node.NodeName)
assert.Equal("192.168.0.130:9100", node.Instance)
assert.Equal("x86_64", node.Machine)
assert.Equal("6.8.0-85-generic", node.Kernel)
assert.InDelta(1.759572263e+09, node.BootTime, 1)

// CPU
assert.Equal(8, node.CPUCoreCount)
assert.Len(node.CPUSecondsTotal, 64) // 8核 × 8模式
assert.InDelta(1.066988737e+07, node.CPUSecondsTotal["0:idle"], 1)
assert.InDelta(325095.17, node.CPUSecondsTotal["0:user"], 1)

// Load
assert.InDelta(0.56, node.Load1, 0.01)

// Memory
assert.Equal(int64(33528840192), node.MemTotal)
assert.Equal(int64(29597294592), node.MemAvailable)

// Filesystem (过滤后)
assert.Len(node.Filesystems, 3) // /, /boot, /boot/efi
rootFS := findFS(node.Filesystems, "/")
assert.Equal("/dev/mapper/ubuntu--vg-ubuntu--lv", rootFS.Device)
assert.Equal(int64(105089261568), rootFS.SizeBytes)
assert.Equal(int64(87484624896), rootFS.AvailBytes)

// Disk I/O (过滤掉 dm-0)
assert.Len(node.DiskIO, 1) // 只有 sda
assert.Equal("sda", node.DiskIO[0].Device)
assert.InDelta(4.22935286272e+11, node.DiskIO[0].ReadBytesTotal, 1)

// Network (过滤后)
assert.Len(node.Networks, 1) // 只有 eno1
assert.Equal("eno1", node.Networks[0].Device)
assert.True(node.Networks[0].Up)
assert.Equal(int64(125000000), node.Networks[0].Speed) // 1Gbps

// Temperature
assert.GreaterOrEqual(len(node.HWMonTemps), 5) // coretemp 5 sensors
cpu_temp := findTemp(node.HWMonTemps, "platform_coretemp_0", "temp1")
assert.InDelta(49.0, cpu_temp.Current, 1)
assert.InDelta(74.0, cpu_temp.Max, 1)

// PSI
assert.InDelta(110989.082491, node.PSICPUWaiting, 1)

// TCP
assert.Equal(int64(138), node.TCPCurrEstab)
assert.Equal(int64(49), node.TCPTimeWait)

// System
assert.Equal(int64(262144), node.ConntrackLimit)
assert.Equal(int64(3840), node.FilefdAllocated)
assert.Equal(int64(256), node.EntropyBits)

// NTP
assert.Equal(float64(1), node.TimexSyncStatus)

// Softnet (所有 CPU 求和)
assert.Equal(int64(0), node.SoftnetDropped)
assert.Equal(int64(204), node.SoftnetSqueezed) // 2+2+1+1+1+195+1+1
```

#### Test_ParseNodeMetrics_RaspiZero

**断言关键差异**：
```go
node := result["raspi-zero"]
assert.Equal("aarch64", node.Machine)
assert.Equal(4, node.CPUCoreCount)
assert.Len(node.CPUSecondsTotal, 32) // 4核 × 8模式

// Filesystem
assert.Len(node.Filesystems, 2) // /, /boot/firmware

// Network
assert.Len(node.Networks, 2) // eth0 (up), wlan0 (down)

// Temperature — arm64 芯片名不同
assert.Greater(len(node.HWMonTemps), 0)
// 有 adc, nvme, thermal_zone 多种芯片

// ConntrackLimit 不同
assert.Equal(int64(131072), node.ConntrackLimit) // raspi 上限更低
```

#### Test_ParseNodeMetrics_MultipleNodes

**输入**：包含 6 个节点所有指标的完整 OTel 文本

**断言**：
```go
result := parseNodeMetrics(fullOtelText)
assert.Len(result, 6) // 6 个节点
for _, name := range []string{"desk-zero","desk-one","desk-two","raspi-zero","raspi-one","raspi-nfs"} {
    assert.Contains(result, name)
    assert.NotEmpty(result[name].NodeName)
    assert.NotEmpty(result[name].Instance)
    assert.Greater(result[name].CPUCoreCount, 0)
    assert.Greater(result[name].MemTotal, int64(0))
}
```

#### Test_ParseNodeMetrics_EmptyInput

```go
result := parseNodeMetrics("")
assert.Empty(result)
```

#### Test_ParseNodeMetrics_NoNodeMetrics

```go
// 只有 SLO 指标，无 node_exporter 指标
result := parseNodeMetrics("otel_response_total{...} 100\n")
assert.Empty(result)
```

---

## 7. 转换器测试规范 (converter_test.go)

### 7.1 测试函数签名

```go
func convertToSnapshot(node string, cur, prev *OTelNodeRawMetrics, elapsed float64) *NodeMetricsSnapshot
```

### 7.2 测试用例

#### Test_ConvertToSnapshot_CPUUsage

**构造**：两次采样间隔 15s。

```go
prev := &OTelNodeRawMetrics{
    CPUSecondsTotal: map[string]float64{
        "0:idle": 1000, "0:user": 100, "0:system": 50,
        "0:iowait": 5, "0:nice": 0, "0:irq": 0, "0:softirq": 2, "0:steal": 0,
    },
    CPUCoreCount: 1,
}
cur := &OTelNodeRawMetrics{
    CPUSecondsTotal: map[string]float64{
        "0:idle": 1010, "0:user": 103, "0:system": 51.5,
        "0:iowait": 5.2, "0:nice": 0, "0:irq": 0, "0:softirq": 2.3, "0:steal": 0,
    },
    CPUCoreCount: 1,
    Load1: 0.5, Load5: 0.4, Load15: 0.3,
    CPUFreqHertz: map[string]float64{"0": 2.4e9},
    CPUFreqMaxHertz: 3.8e9,
}
snap := convertToSnapshot("test", cur, prev, 15.0)

// 总 delta = (1010-1000)+(103-100)+(51.5-50)+(5.2-5)+(2.3-2) = 10+3+1.5+0.2+0.3 = 15
// idle delta = 10
// usage = (15-10)/15 * 100 = 33.33%
assert.InDelta(33.33, snap.CPU.UsagePercent, 0.1)
assert.InDelta(20.0, snap.CPU.UserPercent, 0.1)  // 3/15*100
assert.InDelta(10.0, snap.CPU.SystemPercent, 0.1) // 1.5/15*100
assert.InDelta(66.67, snap.CPU.IdlePercent, 0.1)  // 10/15*100
assert.InDelta(1.33, snap.CPU.IOWaitPercent, 0.1) // 0.2/15*100
assert.InDelta(0.5, snap.CPU.Load1, 0.01)
assert.Equal(1, snap.CPU.Cores)
assert.InDelta(2400, snap.CPU.Frequency, 1) // Hz → MHz
```

#### Test_ConvertToSnapshot_Memory

```go
cur := &OTelNodeRawMetrics{
    MemTotal:     33528840192,
    MemAvailable: 29597294592,
    MemFree:      18630926336,
    MemCached:    9811787776,
    MemBuffers:   961503232,
    SwapTotal:    0,
    SwapFree:     0,
}
snap := convertToSnapshot("test", cur, nil, 15.0)

assert.Equal(int64(33528840192), snap.Memory.Total)
assert.Equal(int64(29597294592), snap.Memory.Available)
used := int64(33528840192 - 29597294592)  // Total - Available
assert.Equal(used, snap.Memory.Used)
percent := float64(used) / float64(33528840192) * 100
assert.InDelta(percent, snap.Memory.UsagePercent, 0.1) // ~11.7%
assert.Equal(int64(0), snap.Memory.SwapUsed)
assert.InDelta(0.0, snap.Memory.SwapPercent, 0.1)
```

#### Test_ConvertToSnapshot_DiskRate

```go
prev := &OTelNodeRawMetrics{
    DiskIO: []DiskIORawMetrics{{
        Device: "sda",
        ReadBytesTotal: 1000000, WrittenBytesTotal: 2000000,
        ReadsCompletedTotal: 100, WritesCompletedTotal: 200,
        IOTimeSecondsTotal: 10.0,
    }},
}
cur := &OTelNodeRawMetrics{
    DiskIO: []DiskIORawMetrics{{
        Device: "sda",
        ReadBytesTotal: 1150000, WrittenBytesTotal: 2300000,
        ReadsCompletedTotal: 110, WritesCompletedTotal: 220,
        IOTimeSecondsTotal: 12.5,
    }},
    Filesystems: []FSRawMetrics{{
        Device: "/dev/sda1", MountPoint: "/", FSType: "ext4",
        SizeBytes: 100e9, AvailBytes: 60e9,
    }},
}
snap := convertToSnapshot("test", cur, prev, 15.0)

assert.Len(snap.Disks, 1)
assert.InDelta(10000, snap.Disks[0].ReadRate, 1)     // 150000/15
assert.InDelta(20000, snap.Disks[0].WriteRate, 1)     // 300000/15
assert.InDelta(0.667, snap.Disks[0].ReadIOPS, 0.01)   // 10/15
assert.InDelta(1.333, snap.Disks[0].WriteIOPS, 0.01)  // 20/15
// IO util = delta(io_time) / elapsed * 100 = 2.5/15*100 = 16.67%
assert.InDelta(16.67, snap.Disks[0].IOUtil, 0.1)
```

#### Test_ConvertToSnapshot_PSI

```go
prev := &OTelNodeRawMetrics{
    PSICPUWaiting:    100.0,
    PSIMemoryWaiting: 0.5,
    PSIMemoryStalled: 0.1,
    PSIIOWaiting:     10.0,
    PSIIOStalled:     5.0,
}
cur := &OTelNodeRawMetrics{
    PSICPUWaiting:    101.5,  // +1.5s in 15s
    PSIMemoryWaiting: 0.5,    // no change
    PSIMemoryStalled: 0.1,
    PSIIOWaiting:     10.75,  // +0.75s in 15s
    PSIIOStalled:     5.3,    // +0.3s in 15s
}
snap := convertToSnapshot("test", cur, prev, 15.0)

// PSI CPU some = 1.5/15 * 100 = 10.0%
assert.InDelta(10.0, snap.PSI.CPUSomePercent, 0.1)
// PSI Memory some = 0/15 * 100 = 0%
assert.InDelta(0.0, snap.PSI.MemorySomePercent, 0.1)
// PSI IO some = 0.75/15 * 100 = 5.0%
assert.InDelta(5.0, snap.PSI.IOSomePercent, 0.1)
// PSI IO full = 0.3/15 * 100 = 2.0%
assert.InDelta(2.0, snap.PSI.IOFullPercent, 0.1)
```

#### Test_ConvertToSnapshot_NoPrev

**当 prev == nil 时**，counter 类指标（CPU usage、disk rate、PSI、VMStat）应为零值，gauge 指标正常填充。

```go
snap := convertToSnapshot("test", cur, nil, 15.0)

assert.InDelta(0, snap.CPU.UsagePercent, 0.01)  // 无 prev 无法算 rate
assert.InDelta(0.5, snap.CPU.Load1, 0.01)       // gauge 正常
assert.Equal(int64(33528840192), snap.Memory.Total) // gauge 正常
assert.InDelta(0, snap.PSI.CPUSomePercent, 0.01)   // 无 prev
```

#### Test_ConvertToSnapshot_Temperature_AMD64

```go
cur := &OTelNodeRawMetrics{
    HWMonTemps: []HWMonRawTemp{
        {Chip: "platform_coretemp_0", Sensor: "temp1", Current: 53, Max: 74, Critical: 80},
        {Chip: "platform_coretemp_0", Sensor: "temp2", Current: 49, Max: 74, Critical: 80},
    },
}
snap := convertToSnapshot("test", cur, nil, 15.0)

assert.InDelta(53.0, snap.Temperature.CPUTemp, 0.1)    // 取 Package (temp1)
assert.InDelta(74.0, snap.Temperature.CPUTempMax, 0.1)
assert.Len(snap.Temperature.Sensors, 2)
```

#### Test_ConvertToSnapshot_Temperature_ARM64

```go
cur := &OTelNodeRawMetrics{
    HWMonTemps: []HWMonRawTemp{
        {Chip: "1000120000_pcie_1f000c8000_adc", Sensor: "temp1", Current: 56.05},
        {Chip: "thermal_thermal_zone0", Sensor: "temp0", Current: 53.45},
        {Chip: "nvme_nvme0", Sensor: "temp1", Current: 34.85, Max: 81.85},
    },
}
snap := convertToSnapshot("test", cur, nil, 15.0)

// arm64: 取 thermal_zone 或 adc 的最高值作为 CPU 温度
assert.InDelta(56.05, snap.Temperature.CPUTemp, 0.1)
```

#### Test_ConvertToSnapshot_Softnet

```go
cur := &OTelNodeRawMetrics{SoftnetDropped: 0, SoftnetSqueezed: 204}
snap := convertToSnapshot("test", cur, nil, 15.0)
assert.Equal(int64(0), snap.Softnet.Dropped)
assert.Equal(int64(204), snap.Softnet.Squeezed)
```

---

## 8. Rate 计算器测试规范 (rate_test.go)

```go
func Test_CounterRate_Normal() {
    assert.InDelta(10.0, counterRate(200, 50, 15), 0.01) // (200-50)/15
}

func Test_CounterRate_Reset() {
    assert.InDelta(0.0, counterRate(10, 100, 15), 0.01) // cur < prev → 0
}

func Test_CounterRate_ZeroElapsed() {
    assert.InDelta(0.0, counterRate(200, 50, 0), 0.01) // elapsed=0 → 0
}

func Test_CounterDelta_Normal() {
    assert.InDelta(150.0, counterDelta(200, 50), 0.01)
}

func Test_CounterDelta_Reset() {
    assert.InDelta(0.0, counterDelta(10, 100), 0.01) // cur < prev → 0
}
```

---

## 9. Master API 响应规范

### 9.1 GET /api/v2/node-metrics

**请求**: `GET /api/v2/node-metrics?cluster_id=xxx`

**响应**:
```json
{
  "summary": {
    "total_nodes": 6,
    "online_nodes": 6,
    "offline_nodes": 0,
    "avg_cpu_usage": 15.2,
    "avg_memory_usage": 22.5,
    "avg_disk_usage": 18.3,
    "max_cpu_usage": 32.1,
    "max_memory_usage": 35.4,
    "max_disk_usage": 28.7,
    "avg_cpu_temp": 48.5,
    "max_cpu_temp": 56.0,
    "total_memory": 131255000000,
    "used_memory": 29532000000,
    "total_disk": 562000000000,
    "used_disk": 103000000000,
    "total_network_rx": 125000,
    "total_network_tx": 89000
  },
  "nodes": [
    {
      "node_name": "desk-zero",
      "timestamp": "2026-02-11T10:30:00Z",
      "hostname": "desk-zero",
      "os": "",
      "kernel": "6.8.0-85-generic",
      "uptime": 6739737,
      "cpu": {
        "usage_percent": 12.5,
        "user_percent": 8.2,
        "system_percent": 3.1,
        "idle_percent": 87.5,
        "iowait_percent": 0.3,
        "per_core": [15.2, 11.3, 10.8, 14.1, 9.7, 18.5, 12.0, 11.4],
        "load_1": 0.56,
        "load_5": 0.41,
        "load_15": 0.28,
        "model": "",
        "cores": 8,
        "threads": 8,
        "frequency": 1500
      },
      "memory": { "...": "..." },
      "disks": [ "..." ],
      "networks": [ "..." ],
      "temperature": { "...": "..." },
      "processes": null,
      "psi": {
        "cpu_some_percent": 0.85,
        "memory_some_percent": 0.001,
        "memory_full_percent": 0.0,
        "io_some_percent": 0.12,
        "io_full_percent": 0.08
      },
      "tcp": {
        "curr_estab": 138,
        "time_wait": 49,
        "orphan": 0,
        "alloc": 468,
        "in_use": 69,
        "sockets_used": 782
      },
      "system": {
        "conntrack_entries": 7582,
        "conntrack_limit": 262144,
        "filefd_allocated": 3840,
        "filefd_maximum": 9223372036854775807,
        "entropy_available": 256
      },
      "vmstat": {
        "pgfault_ps": 12500.5,
        "pgmajfault_ps": 0.02,
        "pswpin_ps": 0,
        "pswpout_ps": 0
      },
      "ntp": {
        "offset_seconds": 0.000168911,
        "synced": true
      },
      "softnet": {
        "dropped": 0,
        "squeezed": 204
      }
    }
  ]
}
```

### 9.2 GET /api/v2/node-metrics/{nodeName}

**响应**：单个 `NodeMetricsSnapshot` JSON（同上节点对象格式）。

### 9.3 GET /api/v2/node-metrics/{nodeName}/history

**响应**：不变，仍使用 `MetricsDataPoint` 结构。新增字段（PSI/TCP/etc.）不存入历史表。

---

## 10. 前端阈值规则

详见 `docs/design/active/node-metrics-mock-data.md` 中的阈值规则定义。

关键调整：
- PSI 百分比现在是单一值（非 10s/60s/300s 窗口），前端显示简化为单数字
- TCP 状态只有 CurrEstab/TimeWait/Orphan/Alloc/InUse/SocketsUsed，无 CloseWait/Listen/SynRecv/FinWait

---

## 11. OTel 白名单补充

当前白名单缺少以下指标，需添加到 ConfigMap：

```
node_hwmon_temp_crit_celsius
```

其他所有指标已在白名单中。

---

## 12. 测试文件清单

| 测试文件 | 测试内容 | 位置 |
|----------|----------|------|
| `node_parser_test.go` | Prometheus 文本 → OTelNodeRawMetrics | `sdk/impl/otel/` |
| `rate_test.go` | counterRate / counterDelta | `repository/metrics/` |
| `converter_test.go` | OTelNodeRawMetrics → NodeMetricsSnapshot | `repository/metrics/` |
| `metrics_test.go` | Sync 集成测试（mock OTelClient） | `repository/metrics/` |
| `node_metrics_test.go` | 模型方法（GetPrimaryDisk 等） | `model_v2/` |

### 测试数据文件

| 文件 | 内容 |
|------|------|
| `testdata/otel_desk_zero.txt` | desk-zero 的完整 OTel 格式指标文本 |
| `testdata/otel_raspi_zero.txt` | raspi-zero 的完整 OTel 格式指标文本 |
| `testdata/otel_all_nodes.txt` | 全部 6 个节点的 OTel 指标文本 |

---

## 13. 实施顺序

```
1. 更新 OTel ConfigMap 白名单 (补充 crit_celsius)
2. 扩展 model_v2/node_metrics.go (新增 PSI/TCP/System/VMStat/NTP/Softnet)
3. 创建测试数据文件 (testdata/)
4. 编写 node_parser_test.go (红灯)
5. 实现 node_parser.go (绿灯)
6. 编写 rate_test.go (红灯)
7. 实现 rate.go (绿灯)
8. 编写 converter_test.go (红灯)
9. 实现 converter.go (绿灯)
10. 编写 metrics_test.go (红灯)
11. 实现 metrics.go (绿灯)
12. 集成测试 (go build + 真实数据验证)
```

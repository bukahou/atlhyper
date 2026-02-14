# 节点指标 Mock 数据 — TDD 基准

> 状态：活跃
> 创建：2026-02-11
> 用途：AI 驱动开发的 TDD 测试基准数据
> 前端预览：`/style-preview/metrics`（`atlhyper_web/src/app/style-preview/metrics/page.tsx`）

## 1. 概要

本文档记录节点指标系统的完整 mock 数据，作为 TDD 开发的基准：

- **数据结构合约**：前后端共用的 `NodeData` 接口定义
- **6 节点完整数据**：覆盖 amd64/arm64、control-plane/worker、单磁盘/多磁盘、单网口/多网口
- **node_exporter 字段映射**：每个字段对应的 Prometheus 指标名
- **阈值与告警规则**：前端展示的颜色阈值逻辑

开发时，后端输出必须能生成与本文档一致的数据结构；前端渲染必须能正确展示本文档中的所有值。

---

## 2. 数据结构合约

### 2.1 TypeScript 接口（前端）

```typescript
interface NodeData {
  // ---- 节点基础信息 ----
  name: string;           // 节点名
  ip: string;             // 主 IP 地址
  role: string;           // "control-plane" | "worker"
  arch: string;           // "amd64" | "arm64"
  os: string;             // 操作系统版本
  kernel: string;         // 内核版本
  uptime: number;         // 运行时长（秒）

  // ---- CPU ----
  cpu: {
    usage: number;        // 总使用率 (%)，= 100 - idle
    user: number;         // 用户态 (%)
    system: number;       // 内核态 (%)
    iowait: number;       // IO 等待 (%)
    idle: number;         // 空闲 (%)
    perCore: number[];    // 每核使用率 (%)，数组长度 = threads
    load1: number;        // 1 分钟负载
    load5: number;        // 5 分钟负载
    load15: number;       // 15 分钟负载
    model: string;        // CPU 型号
    cores: number;        // 物理核心数
    threads: number;      // 逻辑线程数
    freqMHz: number;      // 频率 (MHz)
  };

  // ---- 内存 ----
  memory: {
    total: number;        // 总量（字节）
    used: number;         // 已用（字节）
    available: number;    // 可用（字节）
    free: number;         // 空闲（字节）
    cached: number;       // 缓存（字节）
    buffers: number;      // 缓冲（字节）
    swapTotal: number;    // Swap 总量（字节）
    swapUsed: number;     // Swap 已用（字节）
  };

  // ---- 磁盘（数组，每个挂载点一项）----
  disks: {
    device: string;       // 设备名 (nvme0n1p2, mmcblk0p2, sda1)
    mount: string;        // 挂载点 (/, /data, /mnt/nfs)
    fsType: string;       // 文件系统 (ext4, xfs)
    total: number;        // 总容量（字节）
    used: number;         // 已用（字节）
    avail: number;        // 可用（字节）
    readPS: number;       // 读速率（字节/秒）
    writePS: number;      // 写速率（字节/秒）
    readIOPS: number;     // 读 IOPS
    writeIOPS: number;    // 写 IOPS
    ioUtil: number;       // IO 利用率 (%)
  }[];

  // ---- 网络（数组，每个接口一项）----
  networks: {
    iface: string;        // 接口名 (eno1, eth0, cni0)
    ip: string;           // IP 地址
    status: "up" | "down";// 接口状态
    speed: number;        // 链路速率 (Mbps)
    mtu: number;          // MTU
    rxPS: number;         // 接收速率（字节/秒）
    txPS: number;         // 发送速率（字节/秒）
    rxPktsPS: number;     // 接收包率（包/秒）
    txPktsPS: number;     // 发送包率（包/秒）
    rxErrs: number;       // 接收错误数
    txErrs: number;       // 发送错误数
    rxDrop: number;       // 接收丢包数
    txDrop: number;       // 发送丢包数
  }[];

  // ---- 温度 ----
  temperature: {
    cpuTemp: number;      // CPU 温度 (°C)
    cpuMax: number;       // CPU 最大温度阈值 (°C)
    sensors: {
      chip: string;       // 芯片名 (coretemp, cpu_thermal, nvme, acpitz)
      label: string;      // 传感器标签
      temp: number;       // 当前温度 (°C)
      high?: number;      // 高温阈值 (°C)
      crit?: number;      // 临界阈值 (°C)
    }[];
  };

  // ---- PSI 压力信息（node_exporter 新能力）----
  psi: {
    cpuSome10: number;    // CPU some 压力 10s 窗口 (%)
    cpuSome60: number;    // CPU some 压力 60s 窗口 (%)
    cpuSome300: number;   // CPU some 压力 300s 窗口 (%)
    memSome10: number;    // Memory some 压力 10s 窗口 (%)
    memSome60: number;    // Memory some 压力 60s 窗口 (%)
    memSome300: number;   // Memory some 压力 300s 窗口 (%)
    memFull10: number;    // Memory full 压力 10s 窗口 (%)
    memFull60: number;    // Memory full 压力 60s 窗口 (%)
    memFull300: number;   // Memory full 压力 300s 窗口 (%)
    ioSome10: number;     // IO some 压力 10s 窗口 (%)
    ioSome60: number;     // IO some 压力 60s 窗口 (%)
    ioSome300: number;    // IO some 压力 300s 窗口 (%)
    ioFull10: number;     // IO full 压力 10s 窗口 (%)
    ioFull60: number;     // IO full 压力 60s 窗口 (%)
    ioFull300: number;    // IO full 压力 300s 窗口 (%)
  };

  // ---- 文件描述符 ----
  filefd: {
    allocated: number;    // 已分配数量
    max: number;          // 最大数量
  };

  // ---- Conntrack 连接跟踪 ----
  conntrack: {
    entries: number;      // 当前条目数
    limit: number;        // 最大条目数
  };

  // ---- 熵池 ----
  entropy: number;        // 可用熵值（bits）

  // ---- TCP 连接状态 ----
  tcp: {
    established: number;  // ESTABLISHED 连接数
    timeWait: number;     // TIME_WAIT 连接数
    closeWait: number;    // CLOSE_WAIT 连接数
    listen: number;       // LISTEN 连接数
    synRecv: number;      // SYN_RECV 连接数
    finWait: number;      // FIN_WAIT 连接数
    socketsAlloc: number; // 已分配 socket 数
    socketsUsed: number;  // 已使用 socket 数
    orphans: number;      // 孤儿 socket 数
  };

  // ---- VMStat 虚拟内存统计 ----
  vmstat: {
    pgfault: number;      // 页面错误/秒（minor）
    pgmajfault: number;   // 主页面错误/秒（major，需磁盘读取）
    pswpin: number;       // Swap 读入页数/秒
    pswpout: number;      // Swap 写出页数/秒
  };

  // ---- NTP 时间同步 ----
  timex: {
    offsetSec: number;    // 时间偏移（秒）
    synced: boolean;      // 是否已同步
  };

  // ---- Softnet 网络栈 ----
  softnet: {
    dropped: number;      // 丢弃包数
    squeezed: number;     // 被挤压次数（budget 用尽）
  };
}
```

### 2.2 Go 结构体映射（后端）

后端 `model_v2/node_metrics.go` 中的 `NodeMetricsSnapshot` 需要扩展以下字段来支持 node_exporter 新能力。具体映射见第 4 节。

---

## 3. 完整 Mock 数据

### 3.1 集群总览

| 节点 | IP | 角色 | 架构 | CPU | 内存 | 磁盘数 | 网口数 |
|------|----|------|------|-----|------|--------|--------|
| desk-zero | 192.168.0.130 | control-plane | amd64 | i5-8500 6C/6T | 32GB | 1 (NVMe) | 2 (eno1+cni0) |
| desk-one | 192.168.0.7 | worker | amd64 | i7-10700 8C/16T | 64GB | 2 (NVMe+HDD) | 1 |
| desk-two | 192.168.0.46 | worker | amd64 | i5-10400 6C/12T | 32GB | 1 (NVMe) | 1 |
| raspi-zero | 192.168.0.182 | worker | arm64 | Cortex-A76 4C/4T | 8GB | 1 (SD) | 1 |
| raspi-one | 192.168.0.33 | worker | arm64 | Cortex-A76 4C/4T | 8GB | 1 (SD) | 1 |
| raspi-nfs | 192.168.0.153 | worker | arm64 | Cortex-A76 4C/4T | 8GB | 2 (SD+USB HDD) | 1 |

### 3.2 desk-zero（control-plane, amd64）

```json
{
  "name": "desk-zero",
  "ip": "192.168.0.130",
  "role": "control-plane",
  "arch": "amd64",
  "os": "Ubuntu 24.04.3 LTS",
  "kernel": "6.8.0-85-generic",
  "uptime": 11145600,

  "cpu": {
    "usage": 38.2,
    "user": 22.5,
    "system": 12.1,
    "iowait": 3.6,
    "idle": 61.8,
    "perCore": [42, 35, 48, 31, 40, 36, 29, 45],
    "load1": 1.85,
    "load5": 1.62,
    "load15": 1.48,
    "model": "Intel(R) Core(TM) i5-8500 CPU @ 3.00GHz",
    "cores": 6,
    "threads": 6,
    "freqMHz": 3000
  },

  "memory": {
    "total": 33554432000,
    "used": 18253611008,
    "available": 15300821000,
    "free": 2147483648,
    "cached": 10737418240,
    "buffers": 1073741824,
    "swapTotal": 4294967296,
    "swapUsed": 268435456
  },

  "disks": [
    {
      "device": "nvme0n1p2",
      "mount": "/",
      "fsType": "ext4",
      "total": 500107862016,
      "used": 185042247680,
      "avail": 315065614336,
      "readPS": 5242880,
      "writePS": 3145728,
      "readIOPS": 420,
      "writeIOPS": 280,
      "ioUtil": 18.5
    }
  ],

  "networks": [
    {
      "iface": "eno1",
      "ip": "192.168.0.130",
      "status": "up",
      "speed": 1000,
      "mtu": 1500,
      "rxPS": 15728640,
      "txPS": 10485760,
      "rxPktsPS": 12500,
      "txPktsPS": 8800,
      "rxErrs": 0,
      "txErrs": 0,
      "rxDrop": 3,
      "txDrop": 0
    },
    {
      "iface": "cni0",
      "ip": "10.42.0.1",
      "status": "up",
      "speed": 10000,
      "mtu": 1450,
      "rxPS": 8388608,
      "txPS": 6291456,
      "rxPktsPS": 6500,
      "txPktsPS": 5200,
      "rxErrs": 0,
      "txErrs": 0,
      "rxDrop": 0,
      "txDrop": 0
    }
  ],

  "temperature": {
    "cpuTemp": 52.0,
    "cpuMax": 100,
    "sensors": [
      { "chip": "coretemp", "label": "Package id 0", "temp": 52.0, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 0", "temp": 50.0, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 1", "temp": 51.5, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 2", "temp": 53.0, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 3", "temp": 49.5, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 4", "temp": 52.5, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 5", "temp": 51.0, "high": 80, "crit": 100 },
      { "chip": "nvme", "label": "Composite", "temp": 38.0, "high": 70, "crit": 80 },
      { "chip": "acpitz", "label": "Mainboard", "temp": 35.0 }
    ]
  },

  "psi": {
    "cpuSome10": 0.85, "cpuSome60": 0.42, "cpuSome300": 0.28,
    "memSome10": 0.12, "memSome60": 0.05, "memSome300": 0.02,
    "memFull10": 0.0, "memFull60": 0.0, "memFull300": 0.0,
    "ioSome10": 2.15, "ioSome60": 1.08, "ioSome300": 0.65,
    "ioFull10": 0.42, "ioFull60": 0.18, "ioFull300": 0.10
  },

  "filefd": { "allocated": 8432, "max": 9223372036854776000 },
  "conntrack": { "entries": 12580, "limit": 131072 },
  "entropy": 3892,

  "tcp": {
    "established": 285, "timeWait": 42, "closeWait": 3,
    "listen": 38, "synRecv": 0, "finWait": 5,
    "socketsAlloc": 412, "socketsUsed": 325, "orphans": 0
  },

  "vmstat": { "pgfault": 125000, "pgmajfault": 12, "pswpin": 0, "pswpout": 85 },
  "timex": { "offsetSec": 0.000125, "synced": true },
  "softnet": { "dropped": 0, "squeezed": 15 }
}
```

### 3.3 desk-one（worker, amd64, 高负载）

```json
{
  "name": "desk-one",
  "ip": "192.168.0.7",
  "role": "worker",
  "arch": "amd64",
  "os": "Ubuntu 24.04.3 LTS",
  "kernel": "6.8.0-85-generic",
  "uptime": 11145600,

  "cpu": {
    "usage": 65.8,
    "user": 45.2,
    "system": 15.3,
    "iowait": 5.3,
    "idle": 34.2,
    "perCore": [72, 58, 78, 62, 55, 71, 68, 60, 74, 56, 66, 63],
    "load1": 5.82,
    "load5": 5.15,
    "load15": 4.68,
    "model": "Intel(R) Core(TM) i7-10700 CPU @ 2.90GHz",
    "cores": 8,
    "threads": 16,
    "freqMHz": 2900
  },

  "memory": {
    "total": 67108864000,
    "used": 52613349376,
    "available": 14495514624,
    "free": 1073741824,
    "cached": 12884901888,
    "buffers": 2147483648,
    "swapTotal": 8589934592,
    "swapUsed": 1073741824
  },

  "disks": [
    {
      "device": "nvme0n1p2", "mount": "/", "fsType": "ext4",
      "total": 1000204886016, "used": 620127363072, "avail": 380077522944,
      "readPS": 31457280, "writePS": 20971520, "readIOPS": 2200, "writeIOPS": 1500, "ioUtil": 45.8
    },
    {
      "device": "sda1", "mount": "/data", "fsType": "xfs",
      "total": 2000398934016, "used": 1200239360409, "avail": 800159573607,
      "readPS": 52428800, "writePS": 36700160, "readIOPS": 3800, "writeIOPS": 2600, "ioUtil": 62.3
    }
  ],

  "networks": [
    {
      "iface": "enp0s31f6", "ip": "192.168.0.7", "status": "up", "speed": 1000, "mtu": 1500,
      "rxPS": 52428800, "txPS": 36700160, "rxPktsPS": 42000, "txPktsPS": 28000,
      "rxErrs": 0, "txErrs": 0, "rxDrop": 28, "txDrop": 5
    }
  ],

  "temperature": {
    "cpuTemp": 68.5, "cpuMax": 100,
    "sensors": [
      { "chip": "coretemp", "label": "Package id 0", "temp": 68.5, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 0", "temp": 66.0, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 1", "temp": 68.0, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 2", "temp": 65.5, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 3", "temp": 70.0, "high": 80, "crit": 100 },
      { "chip": "nvme", "label": "Composite", "temp": 42.0, "high": 70, "crit": 80 }
    ]
  },

  "psi": {
    "cpuSome10": 8.52, "cpuSome60": 5.18, "cpuSome300": 3.85,
    "memSome10": 3.28, "memSome60": 1.92, "memSome300": 1.15,
    "memFull10": 0.45, "memFull60": 0.22, "memFull300": 0.12,
    "ioSome10": 12.85, "ioSome60": 8.42, "ioSome300": 5.65,
    "ioFull10": 3.82, "ioFull60": 2.15, "ioFull300": 1.28
  },

  "filefd": { "allocated": 24856, "max": 9223372036854776000 },
  "conntrack": { "entries": 85420, "limit": 131072 },
  "entropy": 4012,

  "tcp": {
    "established": 1850, "timeWait": 385, "closeWait": 12,
    "listen": 52, "synRecv": 2, "finWait": 28,
    "socketsAlloc": 2580, "socketsUsed": 2150, "orphans": 3
  },

  "vmstat": { "pgfault": 485000, "pgmajfault": 128, "pswpin": 45, "pswpout": 680 },
  "timex": { "offsetSec": -0.000032, "synced": true },
  "softnet": { "dropped": 0, "squeezed": 285 }
}
```

### 3.4 desk-two（worker, amd64, 低负载）

```json
{
  "name": "desk-two",
  "ip": "192.168.0.46",
  "role": "worker",
  "arch": "amd64",
  "os": "Ubuntu 24.04.3 LTS",
  "kernel": "6.8.0-88-generic",
  "uptime": 11145600,

  "cpu": {
    "usage": 28.5,
    "user": 18.2,
    "system": 8.1,
    "iowait": 2.2,
    "idle": 71.5,
    "perCore": [32, 25, 35, 22, 28, 31, 24, 30],
    "load1": 1.42,
    "load5": 1.28,
    "load15": 1.15,
    "model": "Intel(R) Core(TM) i5-10400 CPU @ 2.90GHz",
    "cores": 6,
    "threads": 12,
    "freqMHz": 2900
  },

  "memory": {
    "total": 33554432000,
    "used": 14495514624,
    "available": 19058917376,
    "free": 3221225472,
    "cached": 8589934592,
    "buffers": 1073741824,
    "swapTotal": 4294967296,
    "swapUsed": 0
  },

  "disks": [
    {
      "device": "nvme0n1p2", "mount": "/", "fsType": "ext4",
      "total": 500107862016, "used": 135291469824, "avail": 364816392192,
      "readPS": 2097152, "writePS": 1048576, "readIOPS": 180, "writeIOPS": 120, "ioUtil": 8.2
    }
  ],

  "networks": [
    {
      "iface": "enp0s31f6", "ip": "192.168.0.46", "status": "up", "speed": 1000, "mtu": 1500,
      "rxPS": 10485760, "txPS": 7340032, "rxPktsPS": 8500, "txPktsPS": 6200,
      "rxErrs": 0, "txErrs": 0, "rxDrop": 0, "txDrop": 0
    }
  ],

  "temperature": {
    "cpuTemp": 42.5, "cpuMax": 100,
    "sensors": [
      { "chip": "coretemp", "label": "Package id 0", "temp": 42.5, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 0", "temp": 41.0, "high": 80, "crit": 100 },
      { "chip": "coretemp", "label": "Core 1", "temp": 43.0, "high": 80, "crit": 100 },
      { "chip": "nvme", "label": "Composite", "temp": 35.0, "high": 70, "crit": 80 }
    ]
  },

  "psi": {
    "cpuSome10": 0.15, "cpuSome60": 0.08, "cpuSome300": 0.05,
    "memSome10": 0.0, "memSome60": 0.0, "memSome300": 0.0,
    "memFull10": 0.0, "memFull60": 0.0, "memFull300": 0.0,
    "ioSome10": 0.42, "ioSome60": 0.18, "ioSome300": 0.08,
    "ioFull10": 0.05, "ioFull60": 0.02, "ioFull300": 0.01
  },

  "filefd": { "allocated": 4256, "max": 9223372036854776000 },
  "conntrack": { "entries": 5280, "limit": 131072 },
  "entropy": 4096,

  "tcp": {
    "established": 125, "timeWait": 18, "closeWait": 0,
    "listen": 28, "synRecv": 0, "finWait": 2,
    "socketsAlloc": 185, "socketsUsed": 152, "orphans": 0
  },

  "vmstat": { "pgfault": 62000, "pgmajfault": 2, "pswpin": 0, "pswpout": 0 },
  "timex": { "offsetSec": 0.000008, "synced": true },
  "softnet": { "dropped": 0, "squeezed": 5 }
}
```

### 3.5 raspi-zero（worker, arm64, 内存压力）

```json
{
  "name": "raspi-zero",
  "ip": "192.168.0.182",
  "role": "worker",
  "arch": "arm64",
  "os": "Ubuntu 24.04.3 LTS",
  "kernel": "6.8.0-1043-raspi",
  "uptime": 11145600,

  "cpu": {
    "usage": 42.1,
    "user": 28.5,
    "system": 10.2,
    "iowait": 3.4,
    "idle": 57.9,
    "perCore": [48, 38, 45, 37],
    "load1": 1.68,
    "load5": 1.52,
    "load15": 1.35,
    "model": "Cortex-A76",
    "cores": 4,
    "threads": 4,
    "freqMHz": 2400
  },

  "memory": {
    "total": 8388608000,
    "used": 5905580032,
    "available": 2483027968,
    "free": 536870912,
    "cached": 1610612736,
    "buffers": 268435456,
    "swapTotal": 2147483648,
    "swapUsed": 536870912
  },

  "disks": [
    {
      "device": "mmcblk0p2", "mount": "/", "fsType": "ext4",
      "total": 62277025792, "used": 28991029248, "avail": 33285996544,
      "readPS": 524288, "writePS": 262144, "readIOPS": 85, "writeIOPS": 42, "ioUtil": 12.5
    }
  ],

  "networks": [
    {
      "iface": "eth0", "ip": "192.168.0.182", "status": "up", "speed": 1000, "mtu": 1500,
      "rxPS": 5242880, "txPS": 3145728, "rxPktsPS": 4200, "txPktsPS": 2800,
      "rxErrs": 0, "txErrs": 0, "rxDrop": 0, "txDrop": 0
    }
  ],

  "temperature": {
    "cpuTemp": 55.2, "cpuMax": 85,
    "sensors": [
      { "chip": "cpu_thermal", "label": "CPU", "temp": 55.2, "high": 80, "crit": 85 }
    ]
  },

  "psi": {
    "cpuSome10": 2.85, "cpuSome60": 1.52, "cpuSome300": 0.95,
    "memSome10": 5.42, "memSome60": 3.18, "memSome300": 2.05,
    "memFull10": 1.28, "memFull60": 0.65, "memFull300": 0.38,
    "ioSome10": 4.15, "ioSome60": 2.82, "ioSome300": 1.65,
    "ioFull10": 1.85, "ioFull60": 1.12, "ioFull300": 0.68
  },

  "filefd": { "allocated": 3128, "max": 9223372036854776000 },
  "conntrack": { "entries": 3850, "limit": 65536 },
  "entropy": 256,

  "tcp": {
    "established": 85, "timeWait": 12, "closeWait": 1,
    "listen": 22, "synRecv": 0, "finWait": 1,
    "socketsAlloc": 128, "socketsUsed": 105, "orphans": 0
  },

  "vmstat": { "pgfault": 185000, "pgmajfault": 85, "pswpin": 120, "pswpout": 2850 },
  "timex": { "offsetSec": 0.000285, "synced": true },
  "softnet": { "dropped": 0, "squeezed": 42 }
}
```

### 3.6 raspi-one（worker, arm64, 高内存+Swap 压力）

```json
{
  "name": "raspi-one",
  "ip": "192.168.0.33",
  "role": "worker",
  "arch": "arm64",
  "os": "Ubuntu 24.04.3 LTS",
  "kernel": "6.8.0-1040-raspi",
  "uptime": 11145600,

  "cpu": {
    "usage": 55.8,
    "user": 38.2,
    "system": 12.8,
    "iowait": 4.8,
    "idle": 44.2,
    "perCore": [62, 52, 58, 51],
    "load1": 2.25,
    "load5": 2.05,
    "load15": 1.82,
    "model": "Cortex-A76",
    "cores": 4,
    "threads": 4,
    "freqMHz": 2400
  },

  "memory": {
    "total": 8388608000,
    "used": 6710886400,
    "available": 1677721600,
    "free": 268435456,
    "cached": 1073741824,
    "buffers": 134217728,
    "swapTotal": 2147483648,
    "swapUsed": 1073741824
  },

  "disks": [
    {
      "device": "mmcblk0p2", "mount": "/", "fsType": "ext4",
      "total": 62277025792, "used": 38654705664, "avail": 23622320128,
      "readPS": 786432, "writePS": 524288, "readIOPS": 125, "writeIOPS": 85, "ioUtil": 22.8
    }
  ],

  "networks": [
    {
      "iface": "eth0", "ip": "192.168.0.33", "status": "up", "speed": 1000, "mtu": 1500,
      "rxPS": 8388608, "txPS": 5242880, "rxPktsPS": 6800, "txPktsPS": 4500,
      "rxErrs": 0, "txErrs": 0, "rxDrop": 5, "txDrop": 0
    }
  ],

  "temperature": {
    "cpuTemp": 62.8, "cpuMax": 85,
    "sensors": [
      { "chip": "cpu_thermal", "label": "CPU", "temp": 62.8, "high": 80, "crit": 85 }
    ]
  },

  "psi": {
    "cpuSome10": 5.85, "cpuSome60": 3.42, "cpuSome300": 2.15,
    "memSome10": 12.5, "memSome60": 8.82, "memSome300": 5.65,
    "memFull10": 4.28, "memFull60": 2.85, "memFull300": 1.72,
    "ioSome10": 6.85, "ioSome60": 4.52, "ioSome300": 2.85,
    "ioFull10": 2.82, "ioFull60": 1.85, "ioFull300": 1.12
  },

  "filefd": { "allocated": 2856, "max": 9223372036854776000 },
  "conntrack": { "entries": 4280, "limit": 65536 },
  "entropy": 215,

  "tcp": {
    "established": 95, "timeWait": 28, "closeWait": 2,
    "listen": 22, "synRecv": 0, "finWait": 3,
    "socketsAlloc": 158, "socketsUsed": 128, "orphans": 1
  },

  "vmstat": { "pgfault": 245000, "pgmajfault": 320, "pswpin": 850, "pswpout": 5200 },
  "timex": { "offsetSec": -0.000185, "synced": true },
  "softnet": { "dropped": 2, "squeezed": 128 }
}
```

### 3.7 raspi-nfs（worker, arm64, NFS 服务器, 低负载）

```json
{
  "name": "raspi-nfs",
  "ip": "192.168.0.153",
  "role": "worker",
  "arch": "arm64",
  "os": "Ubuntu 24.04.3 LTS",
  "kernel": "6.8.0-1043-raspi",
  "uptime": 4406400,

  "cpu": {
    "usage": 18.5,
    "user": 10.2,
    "system": 5.8,
    "iowait": 2.5,
    "idle": 81.5,
    "perCore": [22, 15, 20, 17],
    "load1": 0.72,
    "load5": 0.65,
    "load15": 0.58,
    "model": "Cortex-A76",
    "cores": 4,
    "threads": 4,
    "freqMHz": 2400
  },

  "memory": {
    "total": 8388608000,
    "used": 3758096384,
    "available": 4630511616,
    "free": 1073741824,
    "cached": 1610612736,
    "buffers": 536870912,
    "swapTotal": 2147483648,
    "swapUsed": 0
  },

  "disks": [
    {
      "device": "mmcblk0p2", "mount": "/", "fsType": "ext4",
      "total": 62277025792, "used": 18253611008, "avail": 44023414784,
      "readPS": 262144, "writePS": 131072, "readIOPS": 42, "writeIOPS": 22, "ioUtil": 3.5
    },
    {
      "device": "sda1", "mount": "/mnt/nfs", "fsType": "ext4",
      "total": 2000398934016, "used": 850169499238, "avail": 1150229434778,
      "readPS": 1048576, "writePS": 2097152, "readIOPS": 85, "writeIOPS": 165, "ioUtil": 15.2
    }
  ],

  "networks": [
    {
      "iface": "eth0", "ip": "192.168.0.153", "status": "up", "speed": 1000, "mtu": 1500,
      "rxPS": 2097152, "txPS": 4194304, "rxPktsPS": 1800, "txPktsPS": 3500,
      "rxErrs": 0, "txErrs": 0, "rxDrop": 0, "txDrop": 0
    }
  ],

  "temperature": {
    "cpuTemp": 45.0, "cpuMax": 85,
    "sensors": [
      { "chip": "cpu_thermal", "label": "CPU", "temp": 45.0, "high": 80, "crit": 85 }
    ]
  },

  "psi": {
    "cpuSome10": 0.08, "cpuSome60": 0.03, "cpuSome300": 0.01,
    "memSome10": 0.0, "memSome60": 0.0, "memSome300": 0.0,
    "memFull10": 0.0, "memFull60": 0.0, "memFull300": 0.0,
    "ioSome10": 0.85, "ioSome60": 0.42, "ioSome300": 0.22,
    "ioFull10": 0.15, "ioFull60": 0.08, "ioFull300": 0.03
  },

  "filefd": { "allocated": 1856, "max": 9223372036854776000 },
  "conntrack": { "entries": 1250, "limit": 65536 },
  "entropy": 312,

  "tcp": {
    "established": 42, "timeWait": 5, "closeWait": 0,
    "listen": 18, "synRecv": 0, "finWait": 0,
    "socketsAlloc": 72, "socketsUsed": 58, "orphans": 0
  },

  "vmstat": { "pgfault": 35000, "pgmajfault": 0, "pswpin": 0, "pswpout": 0 },
  "timex": { "offsetSec": 0.000012, "synced": true },
  "softnet": { "dropped": 0, "squeezed": 2 }
}
```

---

## 4. node_exporter 指标映射

每个字段与 node_exporter Prometheus 指标的对应关系（OTel Collector 添加 `otel_` 前缀）：

### 4.1 节点基础信息

| 字段 | node_exporter 指标 | 说明 |
|------|--------------------|------|
| os | `node_os_info{pretty_name}` | OS 版本 |
| kernel | `node_uname_info{release}` | 内核版本 |
| uptime | `node_boot_time_seconds` | 当前时间 - boot_time |
| arch | `node_uname_info{machine}` | 架构 |

### 4.2 CPU

| 字段 | node_exporter 指标 | 计算方式 |
|------|--------------------|----------|
| usage | `node_cpu_seconds_total` | `100 - idle%`（rate 计算） |
| user | `node_cpu_seconds_total{mode="user"}` | rate → 占比 |
| system | `node_cpu_seconds_total{mode="system"}` | rate → 占比 |
| iowait | `node_cpu_seconds_total{mode="iowait"}` | rate → 占比 |
| idle | `node_cpu_seconds_total{mode="idle"}` | rate → 占比 |
| perCore | `node_cpu_seconds_total{cpu="N"}` | 每核 rate → 占比 |
| load1/5/15 | `node_load1`, `node_load5`, `node_load15` | gauge 直读 |
| model | `node_cpu_info{model_name}` | label 直读 |
| cores | `node_cpu_info` | 去重统计 `core` label |
| threads | `node_cpu_seconds_total` | 统计 `cpu` label 数量 |
| freqMHz | `node_cpu_frequency_hertz` 或 `node_cpu_info` | Hz → MHz |

### 4.3 内存

| 字段 | node_exporter 指标 |
|------|-------------------|
| total | `node_memory_MemTotal_bytes` |
| free | `node_memory_MemFree_bytes` |
| available | `node_memory_MemAvailable_bytes` |
| cached | `node_memory_Cached_bytes` |
| buffers | `node_memory_Buffers_bytes` |
| used | `total - free - cached - buffers` |
| swapTotal | `node_memory_SwapTotal_bytes` |
| swapUsed | `SwapTotal - node_memory_SwapFree_bytes` |

### 4.4 磁盘

| 字段 | node_exporter 指标 | 计算方式 |
|------|--------------------|----------|
| device | `node_filesystem_size_bytes{device}` | label |
| mount | `node_filesystem_size_bytes{mountpoint}` | label |
| fsType | `node_filesystem_size_bytes{fstype}` | label |
| total | `node_filesystem_size_bytes` | gauge |
| avail | `node_filesystem_avail_bytes` | gauge |
| used | `size - avail` | 计算 |
| readPS | `node_disk_read_bytes_total` | rate |
| writePS | `node_disk_written_bytes_total` | rate |
| readIOPS | `node_disk_reads_completed_total` | rate |
| writeIOPS | `node_disk_writes_completed_total` | rate |
| ioUtil | `node_disk_io_time_seconds_total` | rate × 100 |

### 4.5 网络

| 字段 | node_exporter 指标 | 计算方式 |
|------|--------------------|----------|
| iface | `node_network_info{device}` | label |
| status | `node_network_up` | 1=up, 0=down |
| speed | `node_network_speed_bytes` | bytes/s → Mbps |
| mtu | `node_network_mtu_bytes` | gauge |
| rxPS | `node_network_receive_bytes_total` | rate |
| txPS | `node_network_transmit_bytes_total` | rate |
| rxPktsPS | `node_network_receive_packets_total` | rate |
| txPktsPS | `node_network_transmit_packets_total` | rate |
| rxErrs | `node_network_receive_errs_total` | rate |
| txErrs | `node_network_transmit_errs_total` | rate |
| rxDrop | `node_network_receive_drop_total` | rate |
| txDrop | `node_network_transmit_drop_total` | rate |

### 4.6 温度

| 字段 | node_exporter 指标 |
|------|-------------------|
| sensors[].temp | `node_hwmon_temp_celsius` |
| sensors[].high | `node_hwmon_temp_max_celsius` |
| sensors[].crit | `node_hwmon_temp_crit_celsius` |
| sensors[].chip | `node_hwmon_temp_celsius{chip}` label |
| sensors[].label | `node_hwmon_temp_celsius{sensor}` label |

### 4.7 PSI（Pressure Stall Information）

| 字段 | node_exporter 指标 |
|------|-------------------|
| cpuSome10/60/300 | `node_pressure_cpu_waiting_seconds_total` → rate，或 `node_pressure_cpu_some_avg10/60/300` |
| memSome10/60/300 | `node_pressure_memory_waiting_seconds_total` |
| memFull10/60/300 | `node_pressure_memory_stalled_seconds_total` |
| ioSome10/60/300 | `node_pressure_io_waiting_seconds_total` |
| ioFull10/60/300 | `node_pressure_io_stalled_seconds_total` |

> 注意：CPU PSI 没有 "full" 指标（Linux 内核不提供）。

### 4.8 系统资源

| 字段 | node_exporter 指标 |
|------|-------------------|
| filefd.allocated | `node_filefd_allocated` |
| filefd.max | `node_filefd_maximum` |
| conntrack.entries | `node_nf_conntrack_entries` |
| conntrack.limit | `node_nf_conntrack_entries_limit` |
| entropy | `node_entropy_available_bits` |

### 4.9 TCP

| 字段 | node_exporter 指标 |
|------|-------------------|
| established | `node_netstat_Tcp_CurrEstab` |
| timeWait | `node_sockstat_TCP_tw` |
| closeWait | `node_tcp_connection_states{state="close_wait"}` |
| listen | `node_tcp_connection_states{state="listen"}` |
| synRecv | `node_tcp_connection_states{state="syn_recv"}` |
| finWait | `node_tcp_connection_states{state="fin_wait1"}` + `fin_wait2` |
| socketsAlloc | `node_sockstat_TCP_alloc` |
| socketsUsed | `node_sockstat_sockets_used` |
| orphans | `node_sockstat_TCP_orphan` |

### 4.10 VMStat

| 字段 | node_exporter 指标 | 计算方式 |
|------|--------------------|----------|
| pgfault | `node_vmstat_pgfault` | rate |
| pgmajfault | `node_vmstat_pgmajfault` | rate |
| pswpin | `node_vmstat_pswpin` | rate |
| pswpout | `node_vmstat_pswpout` | rate |

### 4.11 NTP / Softnet

| 字段 | node_exporter 指标 |
|------|-------------------|
| timex.offsetSec | `node_timex_offset_seconds` |
| timex.synced | `node_timex_sync_status` (1=synced) |
| softnet.dropped | `node_softnet_dropped_total` (rate) |
| softnet.squeezed | `node_softnet_times_squeezed_total` (rate) |

---

## 5. 前端阈值规则

用于 TDD 验证前端颜色/告警展示是否正确。

### 5.1 通用使用率

```
>=80% → 红色 (text-red-500)
>=60% → 黄色 (text-yellow-500)
<60%  → 绿色 (text-green-500)
```

适用于：CPU usage, Memory usage, Disk usage, IO Util, Conntrack %, FD %

### 5.2 PSI 压力

```
>=25% → 红色 (严重压力)
>=10% → 黄色 (中等压力)
>=1%  → 蓝色 (轻微压力)
<1%   → 绿色 (无压力)
```

### 5.3 温度

```
>=95% of cpuMax → 红色 (临界)
>=85% of cpuMax → 黄色 (警告)
<85% of cpuMax  → 绿色 (正常)
```

### 5.4 TCP 连接

```
TIME_WAIT > 200   → 黄色
CLOSE_WAIT > 5    → 红色
SYN_RECV > 10     → 红色（可能 SYN 洪泛）
Orphans > 0       → 黄色
```

### 5.5 VMStat

```
pgmajfault > 100   → 红色 (内存抖动)
pswpin > 0 或 pswpout > 0 → 黄色 (活跃 Swap)
```

### 5.6 其他

```
Entropy < 256          → 红色 (加密操作可能阻塞)
NTP synced = false     → 红色
Softnet dropped > 0    → 红色
Softnet squeezed > 50  → 黄色
```

---

## 6. 测试场景覆盖

Mock 数据特意覆盖了以下边界场景，确保 TDD 测试充分：

| 场景 | 覆盖节点 | 验证点 |
|------|----------|--------|
| 正常低负载 | desk-two, raspi-nfs | 所有指标绿色 |
| 中等负载 | desk-zero, raspi-zero | 部分黄色告警 |
| 高负载 | desk-one | CPU/IO 黄色，Conntrack 接近 65% |
| 高内存+Swap 压力 | raspi-one | Memory >80%，Swap 50%，PSI mem full >4% |
| 多磁盘 | desk-one (NVMe+HDD), raspi-nfs (SD+USB) | 磁盘数组正确渲染 |
| 多网口 | desk-zero (eno1+cni0) | 网络数组正确渲染 |
| amd64 多传感器 | desk-zero (9 sensors) | 传感器列表滚动 |
| arm64 单传感器 | raspi-* (1 sensor) | 传感器简洁展示 |
| Swap 未使用 | desk-two, raspi-nfs | Swap 条不显示 |
| 高 major fault | raspi-one (320/s) | 红色告警 + 提示文案 |
| 活跃 Swap IO | raspi-one (850 in + 5200 out) | 黄色告警 + 提示文案 |
| Softnet dropped | raspi-one (dropped=2) | 红色标记 |
| 低熵 | raspi-one (215 bits) | 红色 + 阻塞警告 |
| NTP 全部同步 | 所有节点 | 绿色 "Synced" |
| Conntrack 低限制 | raspi-* (limit=65536) | 占比计算正确 |
| Conntrack 高限制 | desk-* (limit=131072) | 占比计算正确 |

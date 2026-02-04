// types/node-metrics.ts
// 节点硬件指标类型定义

// ============================================================================
// CPU 指标
// ============================================================================
export interface CPUMetrics {
  usagePercent: number;       // 总使用率
  coreCount: number;          // 物理核心数
  threadCount: number;        // 逻辑线程数
  coreUsages: number[];       // 每线程使用率
  loadAvg1: number;           // 1分钟负载
  loadAvg5: number;           // 5分钟负载
  loadAvg15: number;          // 15分钟负载
  model: string;              // CPU 型号
  frequency: number;          // 主频 (MHz)
}

// ============================================================================
// 内存指标
// ============================================================================
export interface MemoryMetrics {
  totalBytes: number;
  usedBytes: number;
  availableBytes: number;
  usagePercent: number;
  swapTotalBytes: number;
  swapUsedBytes: number;
  swapUsagePercent: number;
  cached: number;
  buffers: number;
}

// ============================================================================
// 磁盘指标
// ============================================================================
export interface DiskMetrics {
  device: string;             // 设备名 (sda, nvme0n1)
  mountPoint: string;         // 挂载点
  fsType: string;             // 文件系统类型
  totalBytes: number;
  usedBytes: number;
  availableBytes: number;
  usagePercent: number;
  readBytesPS: number;        // 读取速率 bytes/s
  writeBytesPS: number;       // 写入速率 bytes/s
  iops: number;               // IOPS
  ioUtil: number;             // IO 利用率 %
}

// ============================================================================
// 网络指标
// ============================================================================
export interface NetworkMetrics {
  interface: string;          // 接口名 (eth0, ens192)
  ipAddress: string;          // IP 地址
  macAddress: string;         // MAC 地址
  status: "up" | "down";      // 状态
  speed: number;              // 链路速度 (Mbps)
  rxBytesPS: number;          // 接收速率 bytes/s
  txBytesPS: number;          // 发送速率 bytes/s
  rxPacketsPS: number;        // 接收包数/s
  txPacketsPS: number;        // 发送包数/s
  rxErrors: number;           // 接收错误
  txErrors: number;           // 发送错误
  rxDropped: number;          // 接收丢包
  txDropped: number;          // 发送丢包
}

// ============================================================================
// 温度指标
// ============================================================================
export interface TemperatureMetrics {
  cpuTemp: number;            // CPU 温度 (°C)
  cpuTempMax: number;         // CPU 最高温度
  gpuTemp?: number;           // GPU 温度 (可选)
  sensors: SensorReading[];   // 其他传感器
}

export interface SensorReading {
  name: string;
  label: string;
  temp: number;
  high?: number;
  critical?: number;
}

// ============================================================================
// 进程指标
// ============================================================================
export interface ProcessMetrics {
  pid: number;
  name: string;
  user: string;
  state: string;              // R/S/D/Z/T
  cpuPercent: number;
  memPercent: number;
  memBytes: number;
  threads: number;
  startTime: string;
  command: string;
}

// ============================================================================
// GPU 指标 (可选)
// ============================================================================
export interface GPUMetrics {
  index: number;
  name: string;
  uuid: string;
  temperature: number;
  fanSpeed: number;           // %
  powerUsage: number;         // W
  powerLimit: number;         // W
  memoryTotal: number;        // bytes
  memoryUsed: number;
  gpuUtilization: number;     // %
  memUtilization: number;     // %
  processes: GPUProcess[];
}

export interface GPUProcess {
  pid: number;
  name: string;
  memoryUsed: number;
}

// ============================================================================
// 节点指标快照 (聚合)
// ============================================================================
export interface NodeMetricsSnapshot {
  nodeName: string;
  timestamp: string;
  cpu: CPUMetrics;
  memory: MemoryMetrics;
  disks: DiskMetrics[];
  networks: NetworkMetrics[];
  temperature: TemperatureMetrics;
  topProcesses: ProcessMetrics[];
  gpus?: GPUMetrics[];
}

// ============================================================================
// 历史数据点
// ============================================================================
export interface MetricsDataPoint {
  timestamp: number;          // Unix timestamp
  cpuUsage: number;
  memUsage: number;
  diskUsage: number;          // 磁盘使用率 %
  temperature: number;        // CPU 温度
}

// ============================================================================
// 节点列表项 (用于选择器)
// ============================================================================
export interface NodeListItem {
  name: string;
  status: "Ready" | "NotReady";
  roles: string[];
  hasMetrics: boolean;
}

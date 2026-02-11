/**
 * 节点硬件指标 API
 *
 * 对接 Master V2 Node Metrics API，并转换字段命名（蛇形 -> 驼峰）
 */

import { get } from "./request";
import type {
  NodeMetricsSnapshot,
  MetricsDataPoint,
  CPUMetrics,
  MemoryMetrics,
  DiskMetrics,
  NetworkMetrics,
  TemperatureMetrics,
  ProcessMetrics,
  SensorReading,
  PSIMetrics,
  TCPMetrics,
  SystemMetrics,
  VMStatMetrics,
  NTPMetrics,
  SoftnetMetrics,
} from "@/types/node-metrics";

// ============================================================================
// 后端响应类型定义（蛇形命名）
// ============================================================================

interface BackendClusterMetricsSummary {
  total_nodes: number;
  online_nodes: number;
  offline_nodes: number;
  avg_cpu_usage: number;
  avg_memory_usage: number;
  avg_disk_usage: number;
  max_cpu_usage: number;
  max_memory_usage: number;
  max_disk_usage: number;
  avg_cpu_temp: number;
  max_cpu_temp: number;
  total_memory: number;
  used_memory: number;
  total_disk: number;
  used_disk: number;
  total_network_rx: number;
  total_network_tx: number;
}

interface BackendNodeMetricsSnapshot {
  node_name: string;
  timestamp: string;
  hostname: string;
  os: string;
  kernel: string;
  uptime: number;
  cpu: {
    usage_percent: number;
    user_percent: number;
    system_percent: number;
    idle_percent: number;
    iowait_percent: number;
    per_core: number[];
    load_1: number;
    load_5: number;
    load_15: number;
    model: string;
    cores: number;
    threads: number;
    frequency: number;
  };
  memory: {
    total: number;
    used: number;
    available: number;
    free: number;
    usage_percent: number;
    cached: number;
    buffers: number;
    swap_total: number;
    swap_used: number;
    swap_free: number;
    swap_percent: number;
  };
  disks: Array<{
    device: string;
    mount_point: string;
    fs_type: string;
    total: number;
    used: number;
    available: number;
    usage_percent: number;
    read_bytes: number;
    write_bytes: number;
    read_rate: number;
    write_rate: number;
    read_iops: number;
    write_iops: number;
    io_util: number;
  }>;
  networks: Array<{
    interface: string;
    ip_address: string;
    mac_address: string;
    status: string;
    speed: number;
    mtu: number;
    rx_bytes: number;
    tx_bytes: number;
    rx_packets: number;
    tx_packets: number;
    rx_rate: number;
    tx_rate: number;
    rx_errors: number;
    tx_errors: number;
    rx_dropped: number;
    tx_dropped: number;
  }>;
  temperature: {
    cpu_temp: number;
    cpu_temp_max: number;
    sensors: Array<{
      name: string;
      label: string;
      current: number;
      max: number;
      critical: number;
    }>;
  };
  processes: Array<{
    pid: number;
    name: string;
    cmdline: string;
    user: string;
    status: string;
    cpu_percent: number;
    mem_percent: number;
    mem_rss: number;
    threads: number;
    start_time: number;
  }>;
  psi: {
    cpu_some_percent: number;
    memory_some_percent: number;
    memory_full_percent: number;
    io_some_percent: number;
    io_full_percent: number;
  };
  tcp: {
    curr_estab: number;
    time_wait: number;
    orphan: number;
    alloc: number;
    in_use: number;
    sockets_used: number;
  };
  system: {
    conntrack_entries: number;
    conntrack_limit: number;
    filefd_allocated: number;
    filefd_maximum: number;
    entropy_available: number;
  };
  vmstat: {
    pgfault_ps: number;
    pgmajfault_ps: number;
    pswpin_ps: number;
    pswpout_ps: number;
  };
  ntp: {
    offset_seconds: number;
    synced: boolean;
  };
  softnet: {
    dropped: number;
    squeezed: number;
  };
}

interface BackendMetricsDataPoint {
  timestamp: string;
  node_name: string;
  cpu_usage: number;
  memory_usage: number;
  disk_usage: number;
  disk_io_read: number;
  disk_io_write: number;
  network_rx: number;
  network_tx: number;
  cpu_temp: number;
  load_1: number;
}

interface ClusterNodeMetricsResponse {
  summary: BackendClusterMetricsSummary;
  nodes: BackendNodeMetricsSnapshot[];
}

interface NodeMetricsHistoryResponse {
  node_name: string;
  start: string;
  end: string;
  data: BackendMetricsDataPoint[];
}

// ============================================================================
// 前端类型定义（扩展 summary）
// ============================================================================

export interface ClusterMetricsSummary {
  totalNodes: number;
  onlineNodes: number;
  offlineNodes: number;
  avgCPUUsage: number;
  avgMemoryUsage: number;
  avgDiskUsage: number;
  maxCPUUsage: number;
  maxMemoryUsage: number;
  maxDiskUsage: number;
  avgCPUTemp: number;
  maxCPUTemp: number;
  totalMemory: number;
  usedMemory: number;
  totalDisk: number;
  usedDisk: number;
  totalNetworkRx: number;
  totalNetworkTx: number;
}

export interface ClusterNodeMetricsResult {
  summary: ClusterMetricsSummary;
  nodes: NodeMetricsSnapshot[];
}

export interface NodeMetricsHistoryResult {
  nodeName: string;
  start: Date;
  end: Date;
  data: MetricsDataPoint[];
}

// ============================================================================
// 转换函数
// ============================================================================

function transformSummary(s: BackendClusterMetricsSummary): ClusterMetricsSummary {
  return {
    totalNodes: s.total_nodes,
    onlineNodes: s.online_nodes,
    offlineNodes: s.offline_nodes,
    avgCPUUsage: s.avg_cpu_usage,
    avgMemoryUsage: s.avg_memory_usage,
    avgDiskUsage: s.avg_disk_usage,
    maxCPUUsage: s.max_cpu_usage,
    maxMemoryUsage: s.max_memory_usage,
    maxDiskUsage: s.max_disk_usage,
    avgCPUTemp: s.avg_cpu_temp,
    maxCPUTemp: s.max_cpu_temp,
    totalMemory: s.total_memory,
    usedMemory: s.used_memory,
    totalDisk: s.total_disk,
    usedDisk: s.used_disk,
    totalNetworkRx: s.total_network_rx,
    totalNetworkTx: s.total_network_tx,
  };
}

function transformCPU(cpu: BackendNodeMetricsSnapshot["cpu"]): CPUMetrics {
  return {
    usagePercent: cpu.usage_percent,
    coreCount: cpu.cores,
    threadCount: cpu.threads,
    coreUsages: cpu.per_core || [],
    loadAvg1: cpu.load_1,
    loadAvg5: cpu.load_5,
    loadAvg15: cpu.load_15,
    model: cpu.model,
    frequency: cpu.frequency,
  };
}

function transformMemory(mem: BackendNodeMetricsSnapshot["memory"]): MemoryMetrics {
  return {
    totalBytes: mem.total,
    usedBytes: mem.used,
    availableBytes: mem.available,
    usagePercent: mem.usage_percent,
    swapTotalBytes: mem.swap_total,
    swapUsedBytes: mem.swap_used,
    swapUsagePercent: mem.swap_percent,
    cached: mem.cached,
    buffers: mem.buffers,
  };
}

function transformDisk(disk: BackendNodeMetricsSnapshot["disks"][0]): DiskMetrics {
  return {
    device: disk.device,
    mountPoint: disk.mount_point,
    fsType: disk.fs_type,
    totalBytes: disk.total,
    usedBytes: disk.used,
    availableBytes: disk.available,
    usagePercent: disk.usage_percent,
    readBytesPS: disk.read_rate,
    writeBytesPS: disk.write_rate,
    iops: disk.read_iops + disk.write_iops,
    ioUtil: disk.io_util,
  };
}

function transformNetwork(net: BackendNodeMetricsSnapshot["networks"][0]): NetworkMetrics {
  return {
    interface: net.interface,
    ipAddress: net.ip_address,
    macAddress: net.mac_address,
    status: net.status as "up" | "down",
    speed: net.speed,
    rxBytesPS: net.rx_rate,
    txBytesPS: net.tx_rate,
    rxPacketsPS: 0, // 后端暂不提供 packets/s
    txPacketsPS: 0,
    rxErrors: net.rx_errors,
    txErrors: net.tx_errors,
    rxDropped: net.rx_dropped,
    txDropped: net.tx_dropped,
  };
}

function transformSensor(s: BackendNodeMetricsSnapshot["temperature"]["sensors"][0]): SensorReading {
  return {
    name: s.name,
    label: s.label,
    temp: s.current,
    high: s.max,
    critical: s.critical,
  };
}

function transformTemperature(temp: BackendNodeMetricsSnapshot["temperature"]): TemperatureMetrics {
  return {
    cpuTemp: temp.cpu_temp,
    cpuTempMax: temp.cpu_temp_max,
    sensors: (temp.sensors || []).map(transformSensor),
  };
}

function transformProcess(proc: BackendNodeMetricsSnapshot["processes"][0]): ProcessMetrics {
  return {
    pid: proc.pid,
    name: proc.name,
    user: proc.user,
    state: proc.status,
    cpuPercent: proc.cpu_percent,
    memPercent: proc.mem_percent,
    memBytes: proc.mem_rss,
    threads: proc.threads,
    startTime: new Date(proc.start_time * 1000).toISOString(),
    command: proc.cmdline,
  };
}

function transformPSI(p: BackendNodeMetricsSnapshot["psi"]): PSIMetrics {
  const d = p || {} as BackendNodeMetricsSnapshot["psi"];
  return {
    cpuSomePercent: d.cpu_some_percent || 0,
    memorySomePercent: d.memory_some_percent || 0,
    memoryFullPercent: d.memory_full_percent || 0,
    ioSomePercent: d.io_some_percent || 0,
    ioFullPercent: d.io_full_percent || 0,
  };
}

function transformTCP(t: BackendNodeMetricsSnapshot["tcp"]): TCPMetrics {
  const d = t || {} as BackendNodeMetricsSnapshot["tcp"];
  return {
    currEstab: d.curr_estab || 0,
    timeWait: d.time_wait || 0,
    orphan: d.orphan || 0,
    alloc: d.alloc || 0,
    inUse: d.in_use || 0,
    socketsUsed: d.sockets_used || 0,
  };
}

function transformSystem(s: BackendNodeMetricsSnapshot["system"]): SystemMetrics {
  const d = s || {} as BackendNodeMetricsSnapshot["system"];
  return {
    conntrackEntries: d.conntrack_entries || 0,
    conntrackLimit: d.conntrack_limit || 0,
    filefdAllocated: d.filefd_allocated || 0,
    filefdMaximum: d.filefd_maximum || 0,
    entropyAvailable: d.entropy_available || 0,
  };
}

function transformVMStat(v: BackendNodeMetricsSnapshot["vmstat"]): VMStatMetrics {
  const d = v || {} as BackendNodeMetricsSnapshot["vmstat"];
  return {
    pgfaultPS: d.pgfault_ps || 0,
    pgmajfaultPS: d.pgmajfault_ps || 0,
    pswpinPS: d.pswpin_ps || 0,
    pswpoutPS: d.pswpout_ps || 0,
  };
}

function transformNTP(n: BackendNodeMetricsSnapshot["ntp"]): NTPMetrics {
  const d = n || {} as BackendNodeMetricsSnapshot["ntp"];
  return {
    offsetSeconds: d.offset_seconds || 0,
    synced: d.synced ?? false,
  };
}

function transformSoftnet(s: BackendNodeMetricsSnapshot["softnet"]): SoftnetMetrics {
  const d = s || {} as BackendNodeMetricsSnapshot["softnet"];
  return {
    dropped: d.dropped || 0,
    squeezed: d.squeezed || 0,
  };
}

function transformSnapshot(snapshot: BackendNodeMetricsSnapshot): NodeMetricsSnapshot {
  return {
    nodeName: snapshot.node_name,
    timestamp: snapshot.timestamp,
    cpu: transformCPU(snapshot.cpu),
    memory: transformMemory(snapshot.memory),
    disks: (snapshot.disks || []).map(transformDisk),
    networks: (snapshot.networks || []).map(transformNetwork),
    temperature: transformTemperature(snapshot.temperature),
    topProcesses: (snapshot.processes || []).map(transformProcess),
    gpus: undefined,
    psi: transformPSI(snapshot.psi),
    tcp: transformTCP(snapshot.tcp),
    system: transformSystem(snapshot.system),
    vmstat: transformVMStat(snapshot.vmstat),
    ntp: transformNTP(snapshot.ntp),
    softnet: transformSoftnet(snapshot.softnet),
  };
}

function transformDataPoint(dp: BackendMetricsDataPoint): MetricsDataPoint {
  return {
    timestamp: new Date(dp.timestamp).getTime(),
    cpuUsage: dp.cpu_usage,
    memUsage: dp.memory_usage,
    diskUsage: dp.disk_usage,
    temperature: dp.cpu_temp,
  };
}

// ============================================================================
// API 函数
// ============================================================================

/**
 * 获取集群所有节点指标（含汇总）
 * @param clusterId 集群 ID
 */
export async function getClusterNodeMetrics(clusterId: string): Promise<ClusterNodeMetricsResult> {
  const response = await get<ClusterNodeMetricsResponse>("/api/v2/node-metrics", {
    cluster_id: clusterId,
  });

  const data = response.data;
  return {
    summary: transformSummary(data.summary),
    nodes: (data.nodes || []).map(transformSnapshot),
  };
}

/**
 * 获取单节点详情
 * @param clusterId 集群 ID
 * @param nodeName 节点名称
 */
export async function getNodeMetricsDetail(
  clusterId: string,
  nodeName: string
): Promise<NodeMetricsSnapshot> {
  const response = await get<BackendNodeMetricsSnapshot>(
    `/api/v2/node-metrics/${encodeURIComponent(nodeName)}`,
    { cluster_id: clusterId }
  );
  return transformSnapshot(response.data);
}

/**
 * 获取节点历史数据
 * @param clusterId 集群 ID
 * @param nodeName 节点名称
 * @param hours 时间范围（小时），默认 24
 */
export async function getNodeMetricsHistory(
  clusterId: string,
  nodeName: string,
  hours: number = 24
): Promise<NodeMetricsHistoryResult> {
  const response = await get<NodeMetricsHistoryResponse>(
    `/api/v2/node-metrics/${encodeURIComponent(nodeName)}/history`,
    { cluster_id: clusterId, hours }
  );

  const data = response.data;
  return {
    nodeName: data.node_name,
    start: new Date(data.start),
    end: new Date(data.end),
    data: (data.data || []).map(transformDataPoint),
  };
}

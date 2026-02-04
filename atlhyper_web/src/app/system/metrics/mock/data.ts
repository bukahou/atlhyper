// mock/data.ts
// 硬件指标 Mock 数据

import type {
  NodeListItem,
  NodeMetricsSnapshot,
  MetricsDataPoint,
} from "@/types/node-metrics";

// ============================================================================
// 节点列表
// ============================================================================
export const mockNodeList: NodeListItem[] = [
  { name: "k8s-master-01", status: "Ready", roles: ["control-plane", "master"], hasMetrics: true },
  { name: "k8s-worker-01", status: "Ready", roles: ["worker"], hasMetrics: true },
  { name: "k8s-worker-02", status: "Ready", roles: ["worker"], hasMetrics: true },
  { name: "k8s-worker-03", status: "NotReady", roles: ["worker"], hasMetrics: false },
];

// ============================================================================
// 节点指标快照
// ============================================================================
export const mockNodeMetrics: Record<string, NodeMetricsSnapshot> = {
  "k8s-master-01": {
    nodeName: "k8s-master-01",
    timestamp: new Date().toISOString(),
    cpu: {
      usagePercent: 45.2,
      coreCount: 8,
      threadCount: 16,
      coreUsages: [52.3, 48.1, 42.5, 38.9, 45.6, 51.2, 40.3, 43.8],
      loadAvg1: 2.15,
      loadAvg5: 1.89,
      loadAvg15: 1.72,
      model: "Intel(R) Xeon(R) Gold 6248R CPU @ 3.00GHz",
      frequency: 3000,
    },
    memory: {
      totalBytes: 34359738368,      // 32 GB
      usedBytes: 21474836480,       // 20 GB
      availableBytes: 12884901888,  // 12 GB
      usagePercent: 62.5,
      swapTotalBytes: 4294967296,   // 4 GB
      swapUsedBytes: 536870912,     // 512 MB
      swapUsagePercent: 12.5,
      cached: 8589934592,           // 8 GB
      buffers: 1073741824,          // 1 GB
    },
    disks: [
      {
        device: "nvme0n1p1",
        mountPoint: "/",
        fsType: "ext4",
        totalBytes: 107374182400,   // 100 GB
        usedBytes: 42949672960,     // 40 GB
        availableBytes: 64424509440, // 60 GB
        usagePercent: 40.0,
        readBytesPS: 15728640,      // 15 MB/s
        writeBytesPS: 8388608,      // 8 MB/s
        iops: 1250,
        ioUtil: 35.2,
      },
      {
        device: "sda1",
        mountPoint: "/data",
        fsType: "xfs",
        totalBytes: 1099511627776,  // 1 TB
        usedBytes: 439804651110,    // 410 GB
        availableBytes: 659706976665, // 590 GB
        usagePercent: 40.0,
        readBytesPS: 52428800,      // 50 MB/s
        writeBytesPS: 31457280,     // 30 MB/s
        iops: 3500,
        ioUtil: 58.6,
      },
    ],
    networks: [
      {
        interface: "eth0",
        ipAddress: "192.168.1.101",
        macAddress: "00:50:56:a1:b2:c3",
        status: "up",
        speed: 10000,               // 10 Gbps
        rxBytesPS: 125829120,       // 120 MB/s
        txBytesPS: 83886080,        // 80 MB/s
        rxPacketsPS: 85000,
        txPacketsPS: 62000,
        rxErrors: 0,
        txErrors: 0,
        rxDropped: 12,
        txDropped: 5,
      },
      {
        interface: "cni0",
        ipAddress: "10.244.0.1",
        macAddress: "4a:5b:6c:7d:8e:9f",
        status: "up",
        speed: 10000,
        rxBytesPS: 52428800,        // 50 MB/s
        txBytesPS: 41943040,        // 40 MB/s
        rxPacketsPS: 45000,
        txPacketsPS: 38000,
        rxErrors: 0,
        txErrors: 0,
        rxDropped: 0,
        txDropped: 0,
      },
    ],
    temperature: {
      cpuTemp: 58.5,
      cpuTempMax: 95,
      gpuTemp: undefined,
      sensors: [
        { name: "coretemp", label: "Package id 0", temp: 58.5, high: 80, critical: 95 },
        { name: "coretemp", label: "Core 0", temp: 56.0, high: 80, critical: 95 },
        { name: "coretemp", label: "Core 1", temp: 57.5, high: 80, critical: 95 },
        { name: "coretemp", label: "Core 2", temp: 55.0, high: 80, critical: 95 },
        { name: "coretemp", label: "Core 3", temp: 59.0, high: 80, critical: 95 },
        { name: "acpitz", label: "Mainboard", temp: 42.0 },
      ],
    },
    topProcesses: [
      { pid: 1234, name: "kube-apiserver", user: "root", state: "S", cpuPercent: 12.5, memPercent: 8.2, memBytes: 2818572288, threads: 45, startTime: "2024-01-15 08:00:00", command: "/usr/local/bin/kube-apiserver --advertise-address=..." },
      { pid: 1235, name: "etcd", user: "root", state: "S", cpuPercent: 8.3, memPercent: 5.6, memBytes: 1932735283, threads: 32, startTime: "2024-01-15 08:00:01", command: "/usr/local/bin/etcd --data-dir=/var/lib/etcd" },
      { pid: 1236, name: "kube-controller", user: "root", state: "S", cpuPercent: 6.2, memPercent: 4.1, memBytes: 1409286144, threads: 28, startTime: "2024-01-15 08:00:02", command: "/usr/local/bin/kube-controller-manager" },
      { pid: 1237, name: "kube-scheduler", user: "root", state: "S", cpuPercent: 3.8, memPercent: 2.5, memBytes: 858993459, threads: 18, startTime: "2024-01-15 08:00:03", command: "/usr/local/bin/kube-scheduler" },
      { pid: 2001, name: "containerd", user: "root", state: "S", cpuPercent: 5.1, memPercent: 3.2, memBytes: 1099511627, threads: 52, startTime: "2024-01-15 07:59:58", command: "/usr/bin/containerd" },
      { pid: 2345, name: "kubelet", user: "root", state: "S", cpuPercent: 4.2, memPercent: 2.8, memBytes: 966367641, threads: 35, startTime: "2024-01-15 08:00:05", command: "/usr/bin/kubelet --config=/var/lib/kubelet/config.yaml" },
      { pid: 3456, name: "coredns", user: "65534", state: "S", cpuPercent: 1.5, memPercent: 0.8, memBytes: 274877906, threads: 12, startTime: "2024-01-15 08:01:00", command: "/coredns -conf /etc/coredns/Corefile" },
      { pid: 4567, name: "calico-node", user: "root", state: "S", cpuPercent: 2.1, memPercent: 1.5, memBytes: 515396075, threads: 24, startTime: "2024-01-15 08:00:10", command: "/bin/calico-node -felix" },
    ],
    gpus: undefined,
  },
  "k8s-worker-01": {
    nodeName: "k8s-worker-01",
    timestamp: new Date().toISOString(),
    cpu: {
      usagePercent: 72.8,
      coreCount: 16,
      threadCount: 32,
      coreUsages: [78.2, 82.1, 65.3, 71.5, 68.9, 75.4, 80.2, 69.8, 74.1, 77.3, 66.8, 72.0, 79.5, 68.2, 73.9, 76.1],
      loadAvg1: 8.52,
      loadAvg5: 7.83,
      loadAvg15: 6.91,
      model: "AMD EPYC 7542 32-Core Processor",
      frequency: 2900,
    },
    memory: {
      totalBytes: 68719476736,      // 64 GB
      usedBytes: 54975581389,       // 51.2 GB
      availableBytes: 13743895347,  // 12.8 GB
      usagePercent: 80.0,
      swapTotalBytes: 8589934592,   // 8 GB
      swapUsedBytes: 2147483648,    // 2 GB
      swapUsagePercent: 25.0,
      cached: 12884901888,          // 12 GB
      buffers: 2147483648,          // 2 GB
    },
    disks: [
      {
        device: "nvme0n1p1",
        mountPoint: "/",
        fsType: "ext4",
        totalBytes: 214748364800,   // 200 GB
        usedBytes: 150323855360,    // 140 GB
        availableBytes: 64424509440, // 60 GB
        usagePercent: 70.0,
        readBytesPS: 104857600,     // 100 MB/s
        writeBytesPS: 52428800,     // 50 MB/s
        iops: 8500,
        ioUtil: 72.5,
      },
    ],
    networks: [
      {
        interface: "ens192",
        ipAddress: "192.168.1.111",
        macAddress: "00:50:56:d1:e2:f3",
        status: "up",
        speed: 25000,               // 25 Gbps
        rxBytesPS: 262144000,       // 250 MB/s
        txBytesPS: 157286400,       // 150 MB/s
        rxPacketsPS: 180000,
        txPacketsPS: 120000,
        rxErrors: 0,
        txErrors: 0,
        rxDropped: 45,
        txDropped: 12,
      },
    ],
    temperature: {
      cpuTemp: 68.2,
      cpuTempMax: 90,
      gpuTemp: 52.0,
      sensors: [
        { name: "k10temp", label: "Tctl", temp: 68.2, high: 85, critical: 90 },
        { name: "k10temp", label: "Tdie", temp: 66.5, high: 85, critical: 90 },
        { name: "nvme", label: "NVMe Composite", temp: 45.0, high: 70, critical: 80 },
      ],
    },
    topProcesses: [
      { pid: 5001, name: "java", user: "app", state: "S", cpuPercent: 25.3, memPercent: 18.5, memBytes: 12717908992, threads: 156, startTime: "2024-01-15 10:00:00", command: "java -Xmx12g -jar /opt/app/service.jar" },
      { pid: 5002, name: "python3", user: "ml", state: "R", cpuPercent: 18.2, memPercent: 12.3, memBytes: 8455716864, threads: 24, startTime: "2024-01-15 11:30:00", command: "python3 /opt/ml/train.py --model=resnet50" },
      { pid: 5003, name: "nginx", user: "www-data", state: "S", cpuPercent: 8.5, memPercent: 2.1, memBytes: 1443109683, threads: 8, startTime: "2024-01-15 08:00:00", command: "nginx: worker process" },
      { pid: 2001, name: "containerd", user: "root", state: "S", cpuPercent: 6.8, memPercent: 3.5, memBytes: 2405181685, threads: 68, startTime: "2024-01-15 07:59:58", command: "/usr/bin/containerd" },
      { pid: 2345, name: "kubelet", user: "root", state: "S", cpuPercent: 5.2, memPercent: 2.2, memBytes: 1511828480, threads: 42, startTime: "2024-01-15 08:00:05", command: "/usr/bin/kubelet --config=/var/lib/kubelet/config.yaml" },
      { pid: 6001, name: "postgres", user: "postgres", state: "S", cpuPercent: 4.8, memPercent: 8.5, memBytes: 5841155686, threads: 28, startTime: "2024-01-15 08:05:00", command: "postgres: writer process" },
    ],
    gpus: [
      {
        index: 0,
        name: "NVIDIA GeForce RTX 3090",
        uuid: "GPU-12345678-abcd-efgh-ijkl-mnopqrstuvwx",
        temperature: 52,
        fanSpeed: 45,
        powerUsage: 185,
        powerLimit: 350,
        memoryTotal: 25769803776,   // 24 GB
        memoryUsed: 18253611008,    // 17 GB
        gpuUtilization: 78,
        memUtilization: 71,
        processes: [
          { pid: 5002, name: "python3", memoryUsed: 15032385536 },
          { pid: 5010, name: "jupyter", memoryUsed: 3221225472 },
        ],
      },
    ],
  },
  "k8s-worker-02": {
    nodeName: "k8s-worker-02",
    timestamp: new Date().toISOString(),
    cpu: {
      usagePercent: 35.6,
      coreCount: 8,
      threadCount: 16,
      coreUsages: [32.1, 38.5, 42.3, 28.9, 35.2, 40.1, 31.8, 36.2],
      loadAvg1: 1.85,
      loadAvg5: 2.12,
      loadAvg15: 1.95,
      model: "Intel(R) Xeon(R) E5-2680 v4 @ 2.40GHz",
      frequency: 2400,
    },
    memory: {
      totalBytes: 34359738368,      // 32 GB
      usedBytes: 15461882266,       // 14.4 GB
      availableBytes: 18897856102,  // 17.6 GB
      usagePercent: 45.0,
      swapTotalBytes: 4294967296,   // 4 GB
      swapUsedBytes: 0,
      swapUsagePercent: 0,
      cached: 6442450944,           // 6 GB
      buffers: 1073741824,          // 1 GB
    },
    disks: [
      {
        device: "sda1",
        mountPoint: "/",
        fsType: "ext4",
        totalBytes: 107374182400,   // 100 GB
        usedBytes: 32212254720,     // 30 GB
        availableBytes: 75161927680, // 70 GB
        usagePercent: 30.0,
        readBytesPS: 10485760,      // 10 MB/s
        writeBytesPS: 5242880,      // 5 MB/s
        iops: 850,
        ioUtil: 22.5,
      },
    ],
    networks: [
      {
        interface: "eth0",
        ipAddress: "192.168.1.112",
        macAddress: "00:50:56:g1:h2:i3",
        status: "up",
        speed: 1000,                // 1 Gbps
        rxBytesPS: 31457280,        // 30 MB/s
        txBytesPS: 20971520,        // 20 MB/s
        rxPacketsPS: 25000,
        txPacketsPS: 18000,
        rxErrors: 0,
        txErrors: 0,
        rxDropped: 0,
        txDropped: 0,
      },
    ],
    temperature: {
      cpuTemp: 48.5,
      cpuTempMax: 85,
      sensors: [
        { name: "coretemp", label: "Package id 0", temp: 48.5, high: 75, critical: 85 },
        { name: "coretemp", label: "Core 0", temp: 46.0, high: 75, critical: 85 },
        { name: "coretemp", label: "Core 1", temp: 47.5, high: 75, critical: 85 },
      ],
    },
    topProcesses: [
      { pid: 3001, name: "redis-server", user: "redis", state: "S", cpuPercent: 8.2, memPercent: 12.5, memBytes: 4294967296, threads: 4, startTime: "2024-01-15 08:00:00", command: "redis-server *:6379" },
      { pid: 3002, name: "mongod", user: "mongodb", state: "S", cpuPercent: 6.5, memPercent: 15.2, memBytes: 5222680576, threads: 32, startTime: "2024-01-15 08:00:00", command: "mongod --config /etc/mongod.conf" },
      { pid: 2001, name: "containerd", user: "root", state: "S", cpuPercent: 4.2, memPercent: 2.8, memBytes: 966367641, threads: 45, startTime: "2024-01-15 07:59:58", command: "/usr/bin/containerd" },
      { pid: 2345, name: "kubelet", user: "root", state: "S", cpuPercent: 3.5, memPercent: 2.1, memBytes: 721554432, threads: 32, startTime: "2024-01-15 08:00:05", command: "/usr/bin/kubelet --config=/var/lib/kubelet/config.yaml" },
    ],
  },
};

// ============================================================================
// 历史数据生成器
// ============================================================================
export function generateHistoryData(nodeName: string, hours: number = 1): MetricsDataPoint[] {
  const now = Date.now();
  const interval = 60 * 1000; // 1分钟间隔
  const points = (hours * 60);
  const data: MetricsDataPoint[] = [];

  // 基于节点名的基础值
  const baseValues: Record<string, { cpu: number; mem: number; disk: number; temp: number }> = {
    "k8s-master-01": { cpu: 45, mem: 62, disk: 40, temp: 58 },
    "k8s-worker-01": { cpu: 72, mem: 80, disk: 70, temp: 68 },
    "k8s-worker-02": { cpu: 35, mem: 45, disk: 30, temp: 48 },
  };
  const base = baseValues[nodeName] || { cpu: 50, mem: 50, disk: 50, temp: 55 };

  for (let i = points - 1; i >= 0; i--) {
    const timestamp = now - i * interval;
    // 添加一些随机波动
    const cpuVariation = (Math.random() - 0.5) * 20;
    const memVariation = (Math.random() - 0.5) * 10;
    const diskVariation = (Math.random() - 0.5) * 5;
    const tempVariation = (Math.random() - 0.5) * 8;

    data.push({
      timestamp,
      cpuUsage: Math.max(0, Math.min(100, base.cpu + cpuVariation)),
      memUsage: Math.max(0, Math.min(100, base.mem + memVariation)),
      diskUsage: Math.max(0, Math.min(100, base.disk + diskVariation)),
      temperature: Math.max(30, Math.min(90, base.temp + tempVariation)),
    });
  }

  return data;
}

// ============================================================================
// 格式化工具
// ============================================================================
export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

export function formatBytesPS(bytesPS: number): string {
  return formatBytes(bytesPS) + "/s";
}

export function formatNumber(num: number, decimals = 1): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(decimals) + "M";
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(decimals) + "K";
  }
  return num.toFixed(decimals);
}

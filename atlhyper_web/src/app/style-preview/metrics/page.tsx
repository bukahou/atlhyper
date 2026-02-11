"use client";

import { useState } from "react";
import { Layout } from "@/components/layout/Layout";
import {
  Cpu, HardDrive, Database, Network, Thermometer, Activity,
  ArrowDown, ArrowUp, ArrowDownToLine, ArrowUpFromLine,
  Wifi, WifiOff, AlertTriangle, Gauge, RefreshCw, Server,
  Shield, Zap, FileText, Link2, Timer, MemoryStick,
  ChevronDown, ChevronRight,
} from "lucide-react";

// ============================================================================
// 工具函数
// ============================================================================

const fmt = (bytes: number, d = 2) => {
  if (bytes === 0) return "0 B";
  const k = 1024, s = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(d)) + " " + s[i];
};
const fmtPS = (b: number) => fmt(b) + "/s";
const fmtN = (n: number, d = 1) => n >= 1e6 ? (n / 1e6).toFixed(d) + "M" : n >= 1e3 ? (n / 1e3).toFixed(d) + "K" : n.toFixed(d);
const usageColor = (v: number) => v >= 80 ? "text-red-500" : v >= 60 ? "text-yellow-500" : "text-green-500";
const usageBg = (v: number) => v >= 80 ? "bg-red-500" : v >= 60 ? "bg-yellow-500" : "bg-green-500";
const psiColor = (v: number) => v >= 25 ? "text-red-500" : v >= 10 ? "text-yellow-500" : v >= 1 ? "text-blue-500" : "text-green-500";
const psiBg = (v: number) => v >= 25 ? "bg-red-500" : v >= 10 ? "bg-yellow-500" : v >= 1 ? "bg-blue-500" : "bg-green-500";

// ============================================================================
// Mock 数据 — 模拟真实集群 6 节点 (node_exporter 全量)
// ============================================================================

interface NodeData {
  name: string; ip: string; role: string; arch: string; os: string; kernel: string; uptime: number;
  cpu: { usage: number; user: number; system: number; iowait: number; idle: number; perCore: number[]; load1: number; load5: number; load15: number; model: string; cores: number; threads: number; freqMHz: number; };
  memory: { total: number; used: number; available: number; free: number; cached: number; buffers: number; swapTotal: number; swapUsed: number; };
  disks: { device: string; mount: string; fsType: string; total: number; used: number; avail: number; readPS: number; writePS: number; readIOPS: number; writeIOPS: number; ioUtil: number; }[];
  networks: { iface: string; ip: string; status: "up" | "down"; speed: number; mtu: number; rxPS: number; txPS: number; rxPktsPS: number; txPktsPS: number; rxErrs: number; txErrs: number; rxDrop: number; txDrop: number; }[];
  temperature: { cpuTemp: number; cpuMax: number; sensors: { chip: string; label: string; temp: number; high?: number; crit?: number; }[]; };
  // === OTel node_exporter 能力 ===
  psi: { cpuSomePercent: number; memorySomePercent: number; memoryFullPercent: number; ioSomePercent: number; ioFullPercent: number; };
  system: { conntrackEntries: number; conntrackLimit: number; filefdAllocated: number; filefdMaximum: number; entropyAvailable: number; };
  tcp: { currEstab: number; timeWait: number; orphan: number; alloc: number; inUse: number; socketsUsed: number; };
  vmstat: { pgfaultPS: number; pgmajfaultPS: number; pswpinPS: number; pswpoutPS: number; };
  ntp: { offsetSeconds: number; synced: boolean; };
  softnet: { dropped: number; squeezed: number; };
}

const nodes: NodeData[] = [
  {
    name: "desk-zero", ip: "192.168.0.130", role: "control-plane", arch: "amd64",
    os: "Ubuntu 24.04.3 LTS", kernel: "6.8.0-85-generic", uptime: 11145600,
    cpu: { usage: 38.2, user: 22.5, system: 12.1, iowait: 3.6, idle: 61.8, perCore: [42, 35, 48, 31, 40, 36, 29, 45], load1: 1.85, load5: 1.62, load15: 1.48, model: "", cores: 6, threads: 6, freqMHz: 3000 },
    memory: { total: 33554432000, used: 18253611008, available: 15300821000, free: 2147483648, cached: 10737418240, buffers: 1073741824, swapTotal: 0, swapUsed: 0 },
    disks: [
      { device: "nvme0n1p2", mount: "/", fsType: "ext4", total: 500107862016, used: 185042247680, avail: 315065614336, readPS: 5242880, writePS: 3145728, readIOPS: 420, writeIOPS: 280, ioUtil: 18.5 },
    ],
    networks: [
      { iface: "eno1", ip: "192.168.0.130", status: "up", speed: 1000, mtu: 1500, rxPS: 15728640, txPS: 10485760, rxPktsPS: 12500, txPktsPS: 8800, rxErrs: 0, txErrs: 0, rxDrop: 3, txDrop: 0 },
      { iface: "cni0", ip: "10.42.0.1", status: "up", speed: 10000, mtu: 1450, rxPS: 8388608, txPS: 6291456, rxPktsPS: 6500, txPktsPS: 5200, rxErrs: 0, txErrs: 0, rxDrop: 0, txDrop: 0 },
    ],
    temperature: { cpuTemp: 52.0, cpuMax: 100, sensors: [
      { chip: "coretemp", label: "Package id 0", temp: 52.0, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 0", temp: 50.0, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 1", temp: 51.5, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 2", temp: 53.0, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 3", temp: 49.5, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 4", temp: 52.5, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 5", temp: 51.0, high: 80, crit: 100 },
      { chip: "nvme", label: "Composite", temp: 38.0, high: 70, crit: 80 },
      { chip: "acpitz", label: "Mainboard", temp: 35.0 },
    ]},
    psi: { cpuSomePercent: 0.42, memorySomePercent: 0.05, memoryFullPercent: 0.0, ioSomePercent: 1.08, ioFullPercent: 0.18 },
    system: { conntrackEntries: 12580, conntrackLimit: 131072, filefdAllocated: 8432, filefdMaximum: 9223372036854776000, entropyAvailable: 256 },
    tcp: { currEstab: 285, timeWait: 42, orphan: 0, alloc: 412, inUse: 325, socketsUsed: 370 },
    vmstat: { pgfaultPS: 125000, pgmajfaultPS: 12, pswpinPS: 0, pswpoutPS: 85 },
    ntp: { offsetSeconds: 0.000125, synced: true },
    softnet: { dropped: 0, squeezed: 15 },
  },
  {
    name: "desk-one", ip: "192.168.0.7", role: "worker", arch: "amd64",
    os: "Ubuntu 24.04.3 LTS", kernel: "6.8.0-85-generic", uptime: 11145600,
    cpu: { usage: 65.8, user: 45.2, system: 15.3, iowait: 5.3, idle: 34.2, perCore: [72, 58, 78, 62, 55, 71, 68, 60, 74, 56, 66, 63], load1: 5.82, load5: 5.15, load15: 4.68, model: "", cores: 8, threads: 16, freqMHz: 2900 },
    memory: { total: 67108864000, used: 52613349376, available: 14495514624, free: 1073741824, cached: 12884901888, buffers: 2147483648, swapTotal: 0, swapUsed: 0 },
    disks: [
      { device: "nvme0n1p2", mount: "/", fsType: "ext4", total: 1000204886016, used: 620127363072, avail: 380077522944, readPS: 31457280, writePS: 20971520, readIOPS: 2200, writeIOPS: 1500, ioUtil: 45.8 },
      { device: "sda1", mount: "/data", fsType: "xfs", total: 2000398934016, used: 1200239360409, avail: 800159573607, readPS: 52428800, writePS: 36700160, readIOPS: 3800, writeIOPS: 2600, ioUtil: 62.3 },
    ],
    networks: [
      { iface: "enp0s31f6", ip: "192.168.0.7", status: "up", speed: 1000, mtu: 1500, rxPS: 52428800, txPS: 36700160, rxPktsPS: 42000, txPktsPS: 28000, rxErrs: 0, txErrs: 0, rxDrop: 28, txDrop: 5 },
    ],
    temperature: { cpuTemp: 68.5, cpuMax: 100, sensors: [
      { chip: "coretemp", label: "Package id 0", temp: 68.5, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 0", temp: 66.0, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 1", temp: 68.0, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 2", temp: 65.5, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 3", temp: 70.0, high: 80, crit: 100 },
      { chip: "nvme", label: "Composite", temp: 42.0, high: 70, crit: 80 },
    ]},
    psi: { cpuSomePercent: 5.18, memorySomePercent: 1.92, memoryFullPercent: 0.22, ioSomePercent: 8.42, ioFullPercent: 2.15 },
    system: { conntrackEntries: 85420, conntrackLimit: 131072, filefdAllocated: 24856, filefdMaximum: 9223372036854776000, entropyAvailable: 256 },
    tcp: { currEstab: 1850, timeWait: 385, orphan: 3, alloc: 2580, inUse: 2150, socketsUsed: 2400 },
    vmstat: { pgfaultPS: 485000, pgmajfaultPS: 128, pswpinPS: 45, pswpoutPS: 680 },
    ntp: { offsetSeconds: -0.000032, synced: true },
    softnet: { dropped: 0, squeezed: 285 },
  },
  {
    name: "desk-two", ip: "192.168.0.46", role: "worker", arch: "amd64",
    os: "Ubuntu 24.04.3 LTS", kernel: "6.8.0-88-generic", uptime: 11145600,
    cpu: { usage: 28.5, user: 18.2, system: 8.1, iowait: 2.2, idle: 71.5, perCore: [32, 25, 35, 22, 28, 31, 24, 30], load1: 1.42, load5: 1.28, load15: 1.15, model: "", cores: 6, threads: 12, freqMHz: 2900 },
    memory: { total: 33554432000, used: 14495514624, available: 19058917376, free: 3221225472, cached: 8589934592, buffers: 1073741824, swapTotal: 0, swapUsed: 0 },
    disks: [
      { device: "nvme0n1p2", mount: "/", fsType: "ext4", total: 500107862016, used: 135291469824, avail: 364816392192, readPS: 2097152, writePS: 1048576, readIOPS: 180, writeIOPS: 120, ioUtil: 8.2 },
    ],
    networks: [
      { iface: "enp0s31f6", ip: "192.168.0.46", status: "up", speed: 1000, mtu: 1500, rxPS: 10485760, txPS: 7340032, rxPktsPS: 8500, txPktsPS: 6200, rxErrs: 0, txErrs: 0, rxDrop: 0, txDrop: 0 },
    ],
    temperature: { cpuTemp: 42.5, cpuMax: 100, sensors: [
      { chip: "coretemp", label: "Package id 0", temp: 42.5, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 0", temp: 41.0, high: 80, crit: 100 },
      { chip: "coretemp", label: "Core 1", temp: 43.0, high: 80, crit: 100 },
      { chip: "nvme", label: "Composite", temp: 35.0, high: 70, crit: 80 },
    ]},
    psi: { cpuSomePercent: 0.08, memorySomePercent: 0.0, memoryFullPercent: 0.0, ioSomePercent: 0.18, ioFullPercent: 0.02 },
    system: { conntrackEntries: 5280, conntrackLimit: 131072, filefdAllocated: 4256, filefdMaximum: 9223372036854776000, entropyAvailable: 256 },
    tcp: { currEstab: 125, timeWait: 18, orphan: 0, alloc: 185, inUse: 152, socketsUsed: 170 },
    vmstat: { pgfaultPS: 62000, pgmajfaultPS: 2, pswpinPS: 0, pswpoutPS: 0 },
    ntp: { offsetSeconds: 0.000008, synced: true },
    softnet: { dropped: 0, squeezed: 5 },
  },
  {
    name: "raspi-zero", ip: "192.168.0.182", role: "worker", arch: "arm64",
    os: "Ubuntu 24.04.3 LTS", kernel: "6.8.0-1043-raspi", uptime: 11145600,
    cpu: { usage: 42.1, user: 28.5, system: 10.2, iowait: 3.4, idle: 57.9, perCore: [48, 38, 45, 37], load1: 1.68, load5: 1.52, load15: 1.35, model: "", cores: 4, threads: 4, freqMHz: 2400 },
    memory: { total: 8388608000, used: 5905580032, available: 2483027968, free: 536870912, cached: 1610612736, buffers: 268435456, swapTotal: 0, swapUsed: 0 },
    disks: [
      { device: "mmcblk0p2", mount: "/", fsType: "ext4", total: 62277025792, used: 28991029248, avail: 33285996544, readPS: 524288, writePS: 262144, readIOPS: 85, writeIOPS: 42, ioUtil: 12.5 },
    ],
    networks: [
      { iface: "eth0", ip: "192.168.0.182", status: "up", speed: 1000, mtu: 1500, rxPS: 5242880, txPS: 3145728, rxPktsPS: 4200, txPktsPS: 2800, rxErrs: 0, txErrs: 0, rxDrop: 0, txDrop: 0 },
    ],
    temperature: { cpuTemp: 55.2, cpuMax: 85, sensors: [
      { chip: "cpu_thermal", label: "CPU", temp: 55.2, high: 80, crit: 85 },
    ]},
    psi: { cpuSomePercent: 1.52, memorySomePercent: 3.18, memoryFullPercent: 0.65, ioSomePercent: 2.82, ioFullPercent: 1.12 },
    system: { conntrackEntries: 3850, conntrackLimit: 65536, filefdAllocated: 3128, filefdMaximum: 9223372036854776000, entropyAvailable: 256 },
    tcp: { currEstab: 85, timeWait: 12, orphan: 0, alloc: 128, inUse: 105, socketsUsed: 115 },
    vmstat: { pgfaultPS: 185000, pgmajfaultPS: 85, pswpinPS: 120, pswpoutPS: 2850 },
    ntp: { offsetSeconds: 0.000285, synced: true },
    softnet: { dropped: 0, squeezed: 42 },
  },
  {
    name: "raspi-one", ip: "192.168.0.33", role: "worker", arch: "arm64",
    os: "Ubuntu 24.04.3 LTS", kernel: "6.8.0-1040-raspi", uptime: 11145600,
    cpu: { usage: 55.8, user: 38.2, system: 12.8, iowait: 4.8, idle: 44.2, perCore: [62, 52, 58, 51], load1: 2.25, load5: 2.05, load15: 1.82, model: "", cores: 4, threads: 4, freqMHz: 2400 },
    memory: { total: 8388608000, used: 6710886400, available: 1677721600, free: 268435456, cached: 1073741824, buffers: 134217728, swapTotal: 0, swapUsed: 0 },
    disks: [
      { device: "mmcblk0p2", mount: "/", fsType: "ext4", total: 62277025792, used: 38654705664, avail: 23622320128, readPS: 786432, writePS: 524288, readIOPS: 125, writeIOPS: 85, ioUtil: 22.8 },
    ],
    networks: [
      { iface: "eth0", ip: "192.168.0.33", status: "up", speed: 1000, mtu: 1500, rxPS: 8388608, txPS: 5242880, rxPktsPS: 6800, txPktsPS: 4500, rxErrs: 0, txErrs: 0, rxDrop: 5, txDrop: 0 },
    ],
    temperature: { cpuTemp: 62.8, cpuMax: 85, sensors: [
      { chip: "cpu_thermal", label: "CPU", temp: 62.8, high: 80, crit: 85 },
    ]},
    psi: { cpuSomePercent: 3.42, memorySomePercent: 8.82, memoryFullPercent: 2.85, ioSomePercent: 4.52, ioFullPercent: 1.85 },
    system: { conntrackEntries: 4280, conntrackLimit: 65536, filefdAllocated: 2856, filefdMaximum: 9223372036854776000, entropyAvailable: 256 },
    tcp: { currEstab: 95, timeWait: 28, orphan: 1, alloc: 158, inUse: 128, socketsUsed: 145 },
    vmstat: { pgfaultPS: 245000, pgmajfaultPS: 320, pswpinPS: 850, pswpoutPS: 5200 },
    ntp: { offsetSeconds: -0.000185, synced: true },
    softnet: { dropped: 2, squeezed: 128 },
  },
  {
    name: "raspi-nfs", ip: "192.168.0.153", role: "worker", arch: "arm64",
    os: "Ubuntu 24.04.3 LTS", kernel: "6.8.0-1043-raspi", uptime: 4406400,
    cpu: { usage: 18.5, user: 10.2, system: 5.8, iowait: 2.5, idle: 81.5, perCore: [22, 15, 20, 17], load1: 0.72, load5: 0.65, load15: 0.58, model: "", cores: 4, threads: 4, freqMHz: 2400 },
    memory: { total: 8388608000, used: 3758096384, available: 4630511616, free: 1073741824, cached: 1610612736, buffers: 536870912, swapTotal: 0, swapUsed: 0 },
    disks: [
      { device: "mmcblk0p2", mount: "/", fsType: "ext4", total: 62277025792, used: 18253611008, avail: 44023414784, readPS: 262144, writePS: 131072, readIOPS: 42, writeIOPS: 22, ioUtil: 3.5 },
      { device: "sda1", mount: "/mnt/nfs", fsType: "ext4", total: 2000398934016, used: 850169499238, avail: 1150229434778, readPS: 1048576, writePS: 2097152, readIOPS: 85, writeIOPS: 165, ioUtil: 15.2 },
    ],
    networks: [
      { iface: "eth0", ip: "192.168.0.153", status: "up", speed: 1000, mtu: 1500, rxPS: 2097152, txPS: 4194304, rxPktsPS: 1800, txPktsPS: 3500, rxErrs: 0, txErrs: 0, rxDrop: 0, txDrop: 0 },
    ],
    temperature: { cpuTemp: 45.0, cpuMax: 85, sensors: [
      { chip: "cpu_thermal", label: "CPU", temp: 45.0, high: 80, crit: 85 },
    ]},
    psi: { cpuSomePercent: 0.03, memorySomePercent: 0.0, memoryFullPercent: 0.0, ioSomePercent: 0.42, ioFullPercent: 0.08 },
    system: { conntrackEntries: 1250, conntrackLimit: 65536, filefdAllocated: 1856, filefdMaximum: 9223372036854776000, entropyAvailable: 256 },
    tcp: { currEstab: 42, timeWait: 5, orphan: 0, alloc: 72, inUse: 58, socketsUsed: 65 },
    vmstat: { pgfaultPS: 35000, pgmajfaultPS: 0, pswpinPS: 0, pswpoutPS: 0 },
    ntp: { offsetSeconds: 0.000012, synced: true },
    softnet: { dropped: 0, squeezed: 2 },
  },
];

const uptimeStr = (s: number) => {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600);
  return d > 0 ? `${d}d ${h}h` : `${h}h`;
};

// ============================================================================
// 组件
// ============================================================================

// ---- Cluster Summary ----
function ClusterSummary({ data }: { data: NodeData[] }) {
  const avgCPU = data.reduce((a, n) => a + n.cpu.usage, 0) / data.length;
  const avgMem = data.reduce((a, n) => a + (n.memory.used / n.memory.total) * 100, 0) / data.length;
  const maxTemp = Math.max(...data.map(n => n.temperature.cpuTemp));
  const totalConns = data.reduce((a, n) => a + n.tcp.currEstab, 0);
  const maxConntrack = data.reduce((a, n) => Math.max(a, n.system.conntrackLimit > 0 ? n.system.conntrackEntries / n.system.conntrackLimit * 100 : 0), 0);
  const warns = data.filter(n => n.cpu.usage >= 70 || (n.memory.used / n.memory.total) * 100 >= 80 || n.temperature.cpuTemp >= 70).length;

  const cards = [
    { label: "Nodes", value: `${data.length}`, sub: `${warns} warnings`, icon: Server, color: "text-indigo-500", bg: "bg-indigo-500/10" },
    { label: "Avg CPU", value: `${avgCPU.toFixed(1)}%`, sub: `Max: ${Math.max(...data.map(n => n.cpu.usage)).toFixed(1)}%`, icon: Cpu, color: "text-orange-500", bg: "bg-orange-500/10" },
    { label: "Avg Memory", value: `${avgMem.toFixed(1)}%`, sub: `Max: ${Math.max(...data.map(n => (n.memory.used / n.memory.total) * 100)).toFixed(1)}%`, icon: MemoryStick, color: "text-green-500", bg: "bg-green-500/10" },
    { label: "Max Temp", value: `${maxTemp.toFixed(1)}°C`, sub: `Avg: ${(data.reduce((a, n) => a + n.temperature.cpuTemp, 0) / data.length).toFixed(1)}°C`, icon: Thermometer, color: "text-cyan-500", bg: "bg-cyan-500/10" },
    { label: "TCP Conns", value: fmtN(totalConns, 0), sub: `TW: ${data.reduce((a, n) => a + n.tcp.timeWait, 0)}`, icon: Link2, color: "text-blue-500", bg: "bg-blue-500/10" },
    { label: "Conntrack", value: `${maxConntrack.toFixed(1)}%`, sub: "Max table usage", icon: Shield, color: maxConntrack >= 80 ? "text-red-500" : "text-emerald-500", bg: maxConntrack >= 80 ? "bg-red-500/10" : "bg-emerald-500/10" },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
      {cards.map(c => (
        <div key={c.label} className="bg-card rounded-xl border border-[var(--border-color)] p-3">
          <div className="flex items-center gap-2 mb-2">
            <div className={`p-1.5 rounded-lg ${c.bg}`}><c.icon className={`w-4 h-4 ${c.color}`} /></div>
            <span className="text-xs text-muted">{c.label}</span>
          </div>
          <div className="text-xl font-bold text-default">{c.value}</div>
          <div className="text-[10px] text-muted mt-0.5">{c.sub}</div>
        </div>
      ))}
    </div>
  );
}

// ---- PSI Pressure Card (OTel node_exporter) ----
function PSICard({ data }: { data: NodeData }) {
  const resources = [
    { name: "CPU", some: data.psi.cpuSomePercent, full: undefined as number | undefined, icon: Cpu, color: "text-orange-500" },
    { name: "Memory", some: data.psi.memorySomePercent, full: data.psi.memoryFullPercent, icon: MemoryStick, color: "text-green-500" },
    { name: "I/O", some: data.psi.ioSomePercent, full: data.psi.ioFullPercent, icon: Database, color: "text-purple-500" },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-amber-500/10 rounded-lg">
          <Timer className="w-4 h-4 sm:w-5 sm:h-5 text-amber-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">Pressure Stall Information</h3>
          <p className="text-[10px] sm:text-xs text-muted">% of time tasks stalled waiting for resources</p>
        </div>
      </div>

      <div className="space-y-3">
        {resources.map(r => (
          <div key={r.name} className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
            <div className="flex items-center gap-2 mb-2">
              <r.icon className={`w-3.5 h-3.5 ${r.color}`} />
              <span className="text-xs sm:text-sm font-medium text-default">{r.name}</span>
              <span className={`text-xs font-bold ml-auto ${psiColor(r.some)}`}>{r.some.toFixed(2)}%</span>
            </div>
            <div className="h-1.5 bg-[var(--card-bg)] rounded-full overflow-hidden mb-1">
              <div className={`h-full rounded-full ${psiBg(r.some)}`} style={{ width: `${Math.min(100, r.some * 2)}%` }} />
            </div>
            <div className="flex items-center justify-between text-[10px] text-muted">
              <span>some (at least one task stalled)</span>
            </div>
            {r.full !== undefined && (
              <div className="mt-1.5 pt-1.5 border-t border-[var(--border-color)] flex items-center justify-between text-[10px]">
                <span className="text-muted">full (all tasks stalled)</span>
                <span className={psiColor(r.full)}>{r.full.toFixed(2)}%</span>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

// ---- System Resources Card (FD / Conntrack / Entropy / NTP) ----
function SystemResourcesCard({ data }: { data: NodeData }) {
  const fdPct = data.system.filefdMaximum > 0 ? (data.system.filefdAllocated / data.system.filefdMaximum) * 100 : 0;
  const ctPct = data.system.conntrackLimit > 0 ? (data.system.conntrackEntries / data.system.conntrackLimit) * 100 : 0;
  const entropyOk = data.system.entropyAvailable >= 256;

  const items = [
    { label: "File Descriptors", value: fmtN(data.system.filefdAllocated, 0), pct: fdPct, max: `Max: ${fmtN(data.system.filefdMaximum, 0)}`, icon: FileText },
    { label: "Conntrack Table", value: fmtN(data.system.conntrackEntries, 0), pct: ctPct, max: `Limit: ${fmtN(data.system.conntrackLimit, 0)}`, icon: Shield },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-emerald-500/10 rounded-lg">
          <Activity className="w-4 h-4 sm:w-5 sm:h-5 text-emerald-500" />
        </div>
        <h3 className="text-sm sm:text-base font-semibold text-default">System Resources</h3>
      </div>

      <div className="space-y-3">
        {items.map(it => (
          <div key={it.label} className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
            <div className="flex items-center justify-between mb-1.5">
              <div className="flex items-center gap-1.5">
                <it.icon className="w-3.5 h-3.5 text-muted" />
                <span className="text-xs sm:text-sm text-default">{it.label}</span>
              </div>
              <span className={`text-xs sm:text-sm font-bold ${usageColor(it.pct)}`}>{it.value}</span>
            </div>
            <div className="h-1.5 bg-[var(--card-bg)] rounded-full overflow-hidden mb-1">
              <div className={`h-full rounded-full ${usageBg(it.pct)}`} style={{ width: `${Math.min(100, it.pct)}%` }} />
            </div>
            <div className="flex justify-between text-[10px] text-muted">
              <span>{it.pct.toFixed(2)}% used</span>
              <span>{it.max}</span>
            </div>
          </div>
        ))}

        {/* Entropy */}
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5 text-muted" />
              <span className="text-xs sm:text-sm text-default">Entropy Pool</span>
            </div>
            <span className={`text-xs sm:text-sm font-bold ${entropyOk ? "text-green-500" : "text-red-500"}`}>
              {data.system.entropyAvailable} bits
            </span>
          </div>
          <div className="text-[10px] text-muted mt-1">
            {entropyOk ? "Sufficient for cryptographic operations" : "Low entropy - crypto ops may block"}
          </div>
        </div>

        {/* NTP Sync */}
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <RefreshCw className="w-3.5 h-3.5 text-muted" />
              <span className="text-xs sm:text-sm text-default">NTP Sync</span>
            </div>
            <div className="flex items-center gap-2">
              <span className={`text-xs font-medium ${data.ntp.synced ? "text-green-500" : "text-red-500"}`}>
                {data.ntp.synced ? "Synced" : "Not synced"}
              </span>
              <span className="text-[10px] text-muted">
                offset: {(data.ntp.offsetSeconds * 1000).toFixed(3)}ms
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// ---- TCP / Network Stack Card ----
function TCPCard({ data }: { data: NodeData }) {
  const tcp = data.tcp;
  const states = [
    { label: "ESTABLISHED", value: tcp.currEstab, color: "text-green-500" },
    { label: "TIME_WAIT", value: tcp.timeWait, color: tcp.timeWait > 200 ? "text-yellow-500" : "text-default" },
    { label: "ORPHAN", value: tcp.orphan, color: tcp.orphan > 0 ? "text-yellow-500" : "text-default" },
  ];

  const total = tcp.currEstab + tcp.timeWait + tcp.orphan;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-blue-500/10 rounded-lg">
            <Network className="w-4 h-4 sm:w-5 sm:h-5 text-blue-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">TCP / Network Stack</h3>
            <p className="text-[10px] sm:text-xs text-muted">{total} active connections</p>
          </div>
        </div>
      </div>

      {/* TCP States */}
      <div className="grid grid-cols-3 gap-2 mb-3">
        {states.map(s => (
          <div key={s.label} className="p-2 bg-[var(--background)] rounded-lg text-center">
            <div className={`text-base sm:text-lg font-bold ${s.color}`}>{s.value}</div>
            <div className="text-[9px] sm:text-[10px] text-muted leading-tight mt-0.5">{s.label}</div>
          </div>
        ))}
      </div>

      {/* Stacked bar */}
      {total > 0 && (
        <div className="h-2.5 rounded-full overflow-hidden flex mb-3">
          {tcp.currEstab > 0 && <div className="h-full bg-green-500" style={{ width: `${(tcp.currEstab / total) * 100}%` }} />}
          {tcp.timeWait > 0 && <div className="h-full bg-yellow-500" style={{ width: `${(tcp.timeWait / total) * 100}%` }} />}
          {tcp.orphan > 0 && <div className="h-full bg-red-500" style={{ width: `${(tcp.orphan / total) * 100}%` }} />}
        </div>
      )}

      {/* Socket stats */}
      <div className="grid grid-cols-3 gap-2 text-xs">
        <div className="p-2 bg-[var(--background)] rounded-lg">
          <div className="text-muted text-[10px]">Alloc</div>
          <div className="font-medium text-default">{tcp.alloc}</div>
        </div>
        <div className="p-2 bg-[var(--background)] rounded-lg">
          <div className="text-muted text-[10px]">In Use</div>
          <div className="font-medium text-default">{tcp.inUse}</div>
        </div>
        <div className="p-2 bg-[var(--background)] rounded-lg">
          <div className="text-muted text-[10px]">Sockets Used</div>
          <div className="font-medium text-default">{tcp.socketsUsed}</div>
        </div>
      </div>

      {/* Softnet */}
      <div className="mt-3 pt-3 border-t border-[var(--border-color)] grid grid-cols-2 gap-2 text-xs">
        <div className="flex items-center justify-between p-2 bg-[var(--background)] rounded-lg">
          <span className="text-muted">Softnet Dropped</span>
          <span className={`font-medium ${data.softnet.dropped > 0 ? "text-red-500" : "text-green-500"}`}>{data.softnet.dropped}</span>
        </div>
        <div className="flex items-center justify-between p-2 bg-[var(--background)] rounded-lg">
          <span className="text-muted">Softnet Squeezed</span>
          <span className={`font-medium ${data.softnet.squeezed > 50 ? "text-yellow-500" : "text-default"}`}>{data.softnet.squeezed}</span>
        </div>
      </div>
    </div>
  );
}

// ---- VMStat Card (Page Faults / Swap Activity) ----
function VMStatCard({ data }: { data: NodeData }) {
  const vm = data.vmstat;
  const swapActive = vm.pswpinPS > 0 || vm.pswpoutPS > 0;
  const majorFaultWarn = vm.pgmajfaultPS > 100;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-violet-500/10 rounded-lg">
          <MemoryStick className="w-4 h-4 sm:w-5 sm:h-5 text-violet-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">Virtual Memory</h3>
          <p className="text-[10px] sm:text-xs text-muted">Page faults & swap activity (/sec)</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:gap-3">
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Page Faults (minor)</div>
          <div className="text-base sm:text-lg font-bold text-default">{fmtN(vm.pgfaultPS)}</div>
          <div className="text-[10px] text-muted">per second</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Major Faults</div>
          <div className={`text-base sm:text-lg font-bold ${majorFaultWarn ? "text-red-500" : "text-default"}`}>{fmtN(vm.pgmajfaultPS)}</div>
          <div className="text-[10px] text-muted">per second (disk read)</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Swap In</div>
          <div className={`text-base sm:text-lg font-bold ${vm.pswpinPS > 0 ? "text-yellow-500" : "text-default"}`}>{fmtN(vm.pswpinPS)}</div>
          <div className="text-[10px] text-muted">pages/sec</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Swap Out</div>
          <div className={`text-base sm:text-lg font-bold ${vm.pswpoutPS > 0 ? "text-yellow-500" : "text-default"}`}>{fmtN(vm.pswpoutPS)}</div>
          <div className="text-[10px] text-muted">pages/sec</div>
        </div>
      </div>

      {(swapActive || majorFaultWarn) && (
        <div className={`mt-3 pt-3 border-t border-[var(--border-color)] flex items-center gap-2 ${majorFaultWarn ? "text-red-500" : "text-yellow-500"}`}>
          <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
          <span className="text-xs">
            {majorFaultWarn ? "High major page faults — possible memory thrashing" : "Active swap I/O — memory pressure detected"}
          </span>
        </div>
      )}
    </div>
  );
}

// ---- Compact Core Metrics (CPU + Memory + Disk + Network + Temp in one row per node) ----
function CoreMetricsGrid({ data }: { data: NodeData }) {
  const memPct = (data.memory.used / data.memory.total) * 100;
  const swapPct = data.memory.swapTotal > 0 ? (data.memory.swapUsed / data.memory.swapTotal) * 100 : 0;
  const primaryDisk = data.disks[0];
  const primaryNet = data.networks[0];
  const c = data.cpu;

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
      {/* CPU */}
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-orange-500/10 rounded-lg"><Cpu className="w-4 h-4 text-orange-500" /></div>
            <div>
              <h3 className="text-sm font-semibold text-default">CPU</h3>
              <p className="text-[10px] text-muted">{c.cores}C/{c.threads}T @ {c.freqMHz} MHz</p>
            </div>
          </div>
          <div className={`text-xl font-bold ${usageColor(c.usage)}`}>{c.usage.toFixed(1)}%</div>
        </div>
        {/* Per-core bars */}
        <div className="grid grid-cols-4 sm:grid-cols-6 gap-1.5 mb-3">
          {c.perCore.map((v, i) => (
            <div key={i} className="flex flex-col items-center">
              <div className="w-full h-8 bg-[var(--background)] rounded-sm overflow-hidden flex flex-col-reverse">
                <div className={`w-full ${usageBg(v)}`} style={{ height: `${Math.max(v, 2)}%` }} />
              </div>
              <span className="text-[8px] text-muted mt-0.5">{i}</span>
            </div>
          ))}
        </div>
        {/* CPU breakdown + Load */}
        <div className="grid grid-cols-5 gap-1.5 text-[10px] mb-2">
          <div className="p-1.5 bg-[var(--background)] rounded text-center"><div className="text-muted">user</div><div className="font-medium text-default">{c.user.toFixed(1)}%</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded text-center"><div className="text-muted">sys</div><div className="font-medium text-default">{c.system.toFixed(1)}%</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded text-center"><div className="text-muted">iowait</div><div className={`font-medium ${c.iowait > 5 ? "text-yellow-500" : "text-default"}`}>{c.iowait.toFixed(1)}%</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded text-center"><div className="text-muted">idle</div><div className="font-medium text-default">{c.idle.toFixed(1)}%</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded text-center"><div className="text-muted">load1</div><div className="font-medium text-default">{c.load1.toFixed(2)}</div></div>
        </div>
        <div className="text-[10px] text-muted truncate" title={c.model}>{c.model}</div>
      </div>

      {/* Memory */}
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-green-500/10 rounded-lg"><HardDrive className="w-4 h-4 text-green-500" /></div>
            <div>
              <h3 className="text-sm font-semibold text-default">Memory</h3>
              <p className="text-[10px] text-muted">Total: {fmt(data.memory.total)}</p>
            </div>
          </div>
          <div className={`text-xl font-bold ${usageColor(memPct)}`}>{memPct.toFixed(1)}%</div>
        </div>
        {/* Segmented bar */}
        <div className="h-3 bg-[var(--background)] rounded-full overflow-hidden flex mb-2">
          <div className="h-full bg-green-500" style={{ width: `${((data.memory.used - data.memory.cached - data.memory.buffers) / data.memory.total) * 100}%` }} />
          <div className="h-full bg-blue-500" style={{ width: `${(data.memory.cached / data.memory.total) * 100}%` }} />
          <div className="h-full bg-purple-500" style={{ width: `${(data.memory.buffers / data.memory.total) * 100}%` }} />
        </div>
        <div className="flex gap-3 text-[10px] text-muted mb-3">
          <span className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-full bg-green-500" />Used</span>
          <span className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-full bg-blue-500" />Cached</span>
          <span className="flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-full bg-purple-500" />Buffers</span>
        </div>
        <div className="grid grid-cols-4 gap-1.5 text-[10px]">
          <div className="p-1.5 bg-[var(--background)] rounded"><div className="text-muted">Used</div><div className="font-medium text-default">{fmt(data.memory.used)}</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded"><div className="text-muted">Avail</div><div className="font-medium text-default">{fmt(data.memory.available)}</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded"><div className="text-muted">Cached</div><div className="font-medium text-blue-500">{fmt(data.memory.cached)}</div></div>
          <div className="p-1.5 bg-[var(--background)] rounded"><div className="text-muted">Buffers</div><div className="font-medium text-purple-500">{fmt(data.memory.buffers)}</div></div>
        </div>
        {swapPct > 0 && (
          <div className="mt-2 pt-2 border-t border-[var(--border-color)] flex items-center gap-2 text-[10px]">
            <RefreshCw className="w-3 h-3 text-muted" />
            <span className="text-muted">Swap:</span>
            <span className="text-default">{fmt(data.memory.swapUsed)} / {fmt(data.memory.swapTotal)}</span>
            <span className={usageColor(swapPct)}>({swapPct.toFixed(1)}%)</span>
          </div>
        )}
      </div>

      {/* Disk */}
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-purple-500/10 rounded-lg"><Database className="w-4 h-4 text-purple-500" /></div>
            <h3 className="text-sm font-semibold text-default">Disk</h3>
          </div>
          <div className="flex gap-2 text-xs">
            <span className="flex items-center gap-0.5"><ArrowDown className="w-3 h-3 text-blue-500" />{fmtPS(data.disks.reduce((a, d) => a + d.readPS, 0))}</span>
            <span className="flex items-center gap-0.5"><ArrowUp className="w-3 h-3 text-orange-500" />{fmtPS(data.disks.reduce((a, d) => a + d.writePS, 0))}</span>
          </div>
        </div>
        <div className="space-y-2">
          {data.disks.map(d => {
            const pct = (d.used / d.total) * 100;
            return (
              <div key={d.mount} className="p-2 bg-[var(--background)] rounded-lg">
                <div className="flex justify-between text-[10px] mb-1">
                  <span className="text-default font-medium">{d.mount} <span className="text-muted">({d.device}, {d.fsType})</span></span>
                  <span className={usageColor(pct)}>{pct.toFixed(1)}%</span>
                </div>
                <div className="h-1.5 bg-[var(--card-bg)] rounded-full overflow-hidden mb-1.5">
                  <div className={`h-full rounded-full ${usageBg(pct)}`} style={{ width: `${pct}%` }} />
                </div>
                <div className="grid grid-cols-4 gap-1.5 text-[10px]">
                  <div><span className="text-muted">R: </span><span className="text-default">{fmtPS(d.readPS)}</span></div>
                  <div><span className="text-muted">W: </span><span className="text-default">{fmtPS(d.writePS)}</span></div>
                  <div><span className="text-muted">IOPS: </span><span className="text-default">{d.readIOPS + d.writeIOPS}</span></div>
                  <div><span className="text-muted">Util: </span><span className={usageColor(d.ioUtil)}>{d.ioUtil.toFixed(1)}%</span></div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Network */}
      <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-4">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-blue-500/10 rounded-lg"><Network className="w-4 h-4 text-blue-500" /></div>
            <h3 className="text-sm font-semibold text-default">Network</h3>
          </div>
          <div className="flex gap-2 text-xs">
            <span className="flex items-center gap-0.5"><ArrowDownToLine className="w-3 h-3 text-green-500" />{fmtPS(data.networks.reduce((a, n) => a + n.rxPS, 0))}</span>
            <span className="flex items-center gap-0.5"><ArrowUpFromLine className="w-3 h-3 text-blue-500" />{fmtPS(data.networks.reduce((a, n) => a + n.txPS, 0))}</span>
          </div>
        </div>
        <div className="space-y-2">
          {data.networks.map(n => (
            <div key={n.iface} className="p-2 bg-[var(--background)] rounded-lg">
              <div className="flex items-center gap-1.5 mb-1.5 text-[10px]">
                {n.status === "up" ? <Wifi className="w-3 h-3 text-green-500" /> : <WifiOff className="w-3 h-3 text-red-500" />}
                <span className="text-default font-medium">{n.iface}</span>
                <span className="text-muted">{n.ip}</span>
                <span className="text-muted ml-auto">{n.speed >= 10000 ? `${n.speed / 1000}G` : `${n.speed}M`} MTU:{n.mtu}</span>
              </div>
              <div className="grid grid-cols-2 gap-2 mb-1.5">
                <div className="text-[10px]">
                  <div className="flex justify-between mb-0.5"><span className="text-muted">Rx</span><span className="text-green-500">{fmtPS(n.rxPS)}</span></div>
                  <div className="h-1 bg-[var(--card-bg)] rounded-full overflow-hidden"><div className="h-full bg-green-500 rounded-full" style={{ width: `${Math.min(100, (n.rxPS / (n.speed * 125000)) * 100)}%` }} /></div>
                </div>
                <div className="text-[10px]">
                  <div className="flex justify-between mb-0.5"><span className="text-muted">Tx</span><span className="text-blue-500">{fmtPS(n.txPS)}</span></div>
                  <div className="h-1 bg-[var(--card-bg)] rounded-full overflow-hidden"><div className="h-full bg-blue-500 rounded-full" style={{ width: `${Math.min(100, (n.txPS / (n.speed * 125000)) * 100)}%` }} /></div>
                </div>
              </div>
              <div className="grid grid-cols-4 gap-1.5 text-[10px]">
                <div><span className="text-muted">RxPkt: </span><span className="text-default">{fmtN(n.rxPktsPS)}/s</span></div>
                <div><span className="text-muted">TxPkt: </span><span className="text-default">{fmtN(n.txPktsPS)}/s</span></div>
                <div><span className="text-muted">Err: </span><span className={n.rxErrs + n.txErrs > 0 ? "text-red-500" : "text-default"}>{n.rxErrs + n.txErrs}</span></div>
                <div><span className="text-muted">Drop: </span><span className={n.rxDrop + n.txDrop > 0 ? "text-yellow-500" : "text-default"}>{n.rxDrop + n.txDrop}</span></div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ---- Temperature (compact) ----
function TempCard({ data }: { data: NodeData }) {
  const t = data.temperature;
  const isWarn = t.cpuTemp >= t.cpuMax * 0.85;
  const isCrit = t.cpuTemp >= t.cpuMax * 0.95;
  const tColor = isCrit ? "text-red-500" : isWarn ? "text-yellow-500" : "text-green-500";

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`p-1.5 rounded-lg ${isCrit ? "bg-red-500/10" : isWarn ? "bg-yellow-500/10" : "bg-cyan-500/10"}`}>
            <Thermometer className={`w-4 h-4 ${tColor}`} />
          </div>
          <h3 className="text-sm font-semibold text-default">Temperature</h3>
        </div>
        <div className={`text-xl font-bold ${tColor}`}>{t.cpuTemp.toFixed(1)}°C</div>
      </div>
      <div className="relative h-2.5 bg-[var(--background)] rounded-full overflow-hidden mb-2">
        <div className="absolute inset-0 flex">
          <div className="flex-[7] bg-gradient-to-r from-blue-500/20 to-green-500/20" />
          <div className="flex-[2] bg-yellow-500/20" />
          <div className="flex-1 bg-red-500/20" />
        </div>
        <div className={`h-full rounded-full ${isCrit ? "bg-red-500" : isWarn ? "bg-yellow-500" : "bg-green-500"}`} style={{ width: `${Math.min(100, (t.cpuTemp / t.cpuMax) * 100)}%`, opacity: 0.8 }} />
      </div>
      <div className="space-y-1 max-h-32 overflow-y-auto">
        {t.sensors.map((s, i) => (
          <div key={i} className="flex items-center justify-between text-[10px] px-1.5 py-1 bg-[var(--background)] rounded">
            <span className="text-muted truncate flex-1">{s.label} <span className="hidden sm:inline text-muted/60">({s.chip})</span></span>
            <span className={`font-medium ml-2 ${s.crit && s.temp >= s.crit ? "text-red-500" : s.high && s.temp >= s.high ? "text-yellow-500" : "text-default"}`}>
              {s.temp.toFixed(1)}°C
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ============================================================================
// Node Detail (expandable)
// ============================================================================

function NodeDetail({ node }: { node: NodeData }) {
  const [expanded, setExpanded] = useState(true);
  const memPct = (node.memory.used / node.memory.total) * 100;
  const diskPct = node.disks[0] ? (node.disks[0].used / node.disks[0].total) * 100 : 0;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      {/* Header */}
      <button
        className="w-full flex items-center justify-between p-3 sm:p-4 hover:bg-[var(--background)] transition-colors"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-3">
          {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4 text-indigo-500" />
            <span className="text-sm font-semibold text-default">{node.name}</span>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-indigo-500/10 text-indigo-500">{node.role}</span>
            <span className="text-[10px] text-muted">{node.arch}</span>
          </div>
        </div>
        <div className="flex items-center gap-3 sm:gap-5 text-xs">
          <span><span className="text-muted">CPU </span><span className={usageColor(node.cpu.usage)}>{node.cpu.usage.toFixed(1)}%</span></span>
          <span><span className="text-muted">Mem </span><span className={usageColor(memPct)}>{memPct.toFixed(1)}%</span></span>
          <span className="hidden sm:inline"><span className="text-muted">Disk </span><span className={usageColor(diskPct)}>{diskPct.toFixed(1)}%</span></span>
          <span className="hidden sm:inline"><span className="text-muted">Temp </span><span className={`${node.temperature.cpuTemp >= 70 ? "text-yellow-500" : "text-default"}`}>{node.temperature.cpuTemp.toFixed(1)}°C</span></span>
          <span className="hidden lg:inline text-muted">up {uptimeStr(node.uptime)}</span>
        </div>
      </button>

      {/* Expanded content */}
      {expanded && (
        <div className="px-3 sm:px-4 pb-3 sm:pb-4 space-y-3">
          {/* System info bar */}
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-muted px-1">
            <span>{node.os}</span>
            <span>{node.kernel}</span>
            <span>{node.ip}</span>
            <span>Uptime: {uptimeStr(node.uptime)}</span>
          </div>

          {/* Core metrics */}
          <CoreMetricsGrid data={node} />

          {/* Temperature */}
          <TempCard data={node} />

          {/* NEW: node_exporter exclusive capabilities */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            <PSICard data={node} />
            <TCPCard data={node} />
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            <SystemResourcesCard data={node} />
            <VMStatCard data={node} />
          </div>
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Main Page
// ============================================================================

export default function MetricsPreviewPage() {
  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const displayed = selectedNode ? nodes.filter(n => n.name === selectedNode) : nodes;

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-6">
        {/* Title */}
        <div>
          <h1 className="text-lg sm:text-xl font-bold text-default">Node Metrics — node_exporter Full Preview</h1>
          <p className="text-xs sm:text-sm text-muted mt-1">
            style-preview: mock data showing all node_exporter capabilities including PSI, TCP stack, conntrack, vmstat
          </p>
        </div>

        {/* Cluster Summary */}
        <ClusterSummary data={nodes} />

        {/* Node Filter */}
        <div className="flex flex-wrap gap-2">
          <button
            className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${!selectedNode ? "bg-indigo-500 text-white border-indigo-500" : "bg-card text-muted border-[var(--border-color)] hover:text-default"}`}
            onClick={() => setSelectedNode(null)}
          >
            All Nodes ({nodes.length})
          </button>
          {nodes.map(n => (
            <button
              key={n.name}
              className={`px-3 py-1.5 text-xs rounded-lg border transition-colors ${selectedNode === n.name ? "bg-indigo-500 text-white border-indigo-500" : "bg-card text-muted border-[var(--border-color)] hover:text-default"}`}
              onClick={() => setSelectedNode(selectedNode === n.name ? null : n.name)}
            >
              {n.name}
            </button>
          ))}
        </div>

        {/* Node Details */}
        <div className="space-y-3">
          {displayed.map(node => (
            <NodeDetail key={node.name} node={node} />
          ))}
        </div>
      </div>
    </Layout>
  );
}

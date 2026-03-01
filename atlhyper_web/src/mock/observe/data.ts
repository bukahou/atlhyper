/**
 * Observe Landing Page Mock 数据
 *
 * 6 个服务覆盖 healthy / warning / critical 三种状态
 * 每个服务包含 K8s 资源 + 可观测信号 + 基础设施
 */

import type { LandingPageResponse, ServiceHealth } from "@/types/model/observe";

const mockServices: ServiceHealth[] = [
  // critical — 低成功率
  {
    name: "geass-gateway",
    namespace: "geass",
    status: "critical",
    deployment: {
      name: "geass-gateway",
      replicas: "3/3",
      image: "geass-gateway:v1.2.3",
      strategy: "RollingUpdate",
      age: "30d",
    },
    pods: [
      { name: "geass-gateway-abc12-x9k", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-one", age: "5d", cpuUsage: "120m", memoryUsage: "256Mi" },
      { name: "geass-gateway-abc12-m3p", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-one", age: "5d", cpuUsage: "95m", memoryUsage: "230Mi" },
      { name: "geass-gateway-abc12-q7w", phase: "Running", ready: "1/1", restarts: 2, nodeName: "desk-two", age: "3d", cpuUsage: "110m", memoryUsage: "248Mi" },
    ],
    k8sService: { type: "ClusterIP", clusterIP: "10.43.128.15", ports: "80/TCP→8080, 443/TCP→8443" },
    ingresses: [
      { name: "geass-gateway", hosts: ["api.geass.dev"], tlsEnabled: true, paths: [{ path: "/api", serviceName: "geass-gateway", port: 80 }] },
      { name: "geass-ws", hosts: ["ws.geass.dev"], tlsEnabled: true, paths: [{ path: "/ws", serviceName: "geass-gateway", port: 8080 }] },
    ],
    apm: {
      rps: 245.8,
      successRate: 0.932,
      errorRate: 0.068,
      p99Ms: 892,
      avgMs: 156,
      spanCount: 88480,
      errorCount: 6016,
    },
    slo: {
      meshSuccessRate: 0.941,
      meshRps: 245.8,
      meshP99Ms: 850,
      mtlsEnabled: true,
      ingressDomains: [
        { domain: "api.geass.dev", successRate: 0.938, rps: 180.2, p99Ms: 920 },
        { domain: "ws.geass.dev", successRate: 0.965, rps: 65.6, p99Ms: 340 },
      ],
    },
    logs: { errorCount: 1247, warnCount: 3892, totalCount: 52340 },
    infra: {
      podCount: 3,
      nodes: [
        { name: "desk-one", cpuPct: 72.5, memPct: 68.3 },
        { name: "desk-two", cpuPct: 65.1, memPct: 71.2 },
      ],
    },
  },
  // warning — 成功率 95-99%
  {
    name: "geass-auth",
    namespace: "geass",
    status: "warning",
    deployment: {
      name: "geass-auth",
      replicas: "2/2",
      image: "geass-auth:v2.1.0",
      strategy: "RollingUpdate",
      age: "15d",
    },
    pods: [
      { name: "geass-auth-def34-a1b", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-one", age: "7d", cpuUsage: "80m", memoryUsage: "192Mi" },
      { name: "geass-auth-def34-c2d", phase: "Running", ready: "1/1", restarts: 1, nodeName: "desk-one", age: "7d", cpuUsage: "75m", memoryUsage: "188Mi" },
    ],
    k8sService: { type: "ClusterIP", clusterIP: "10.43.128.22", ports: "80/TCP→8080" },
    ingresses: [],
    apm: {
      rps: 89.3,
      successRate: 0.985,
      errorRate: 0.015,
      p99Ms: 245,
      avgMs: 42,
      spanCount: 32148,
      errorCount: 482,
    },
    slo: {
      meshSuccessRate: 0.987,
      meshRps: 89.3,
      meshP99Ms: 230,
      mtlsEnabled: true,
    },
    logs: { errorCount: 89, warnCount: 456, totalCount: 18920 },
    infra: {
      podCount: 2,
      nodes: [{ name: "desk-one", cpuPct: 72.5, memPct: 68.3 }],
    },
  },
  // healthy
  {
    name: "geass-media",
    namespace: "geass",
    status: "healthy",
    deployment: {
      name: "geass-media",
      replicas: "2/2",
      image: "geass-media:v1.8.5",
      strategy: "RollingUpdate",
      age: "22d",
    },
    pods: [
      { name: "geass-media-ghi56-e3f", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-two", age: "10d", cpuUsage: "150m", memoryUsage: "512Mi" },
      { name: "geass-media-ghi56-g4h", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-two", age: "10d", cpuUsage: "140m", memoryUsage: "498Mi" },
    ],
    k8sService: { type: "ClusterIP", clusterIP: "10.43.128.35", ports: "80/TCP→8080" },
    ingresses: [
      { name: "geass-media", hosts: ["media.geass.dev"], tlsEnabled: true, paths: [{ path: "/", serviceName: "geass-media", port: 80 }] },
    ],
    apm: {
      rps: 156.2,
      successRate: 0.998,
      errorRate: 0.002,
      p99Ms: 180,
      avgMs: 35,
      spanCount: 56232,
      errorCount: 112,
    },
    slo: {
      meshSuccessRate: 0.999,
      meshRps: 156.2,
      meshP99Ms: 175,
      mtlsEnabled: true,
      ingressDomains: [
        { domain: "media.geass.dev", successRate: 0.997, rps: 156.2, p99Ms: 185 },
      ],
    },
    logs: { errorCount: 12, warnCount: 78, totalCount: 24560 },
    infra: {
      podCount: 2,
      nodes: [{ name: "desk-two", cpuPct: 65.1, memPct: 71.2 }],
    },
  },
  {
    name: "geass-favorites",
    namespace: "geass",
    status: "healthy",
    deployment: {
      name: "geass-favorites",
      replicas: "1/1",
      image: "geass-favorites:v1.3.2",
      strategy: "RollingUpdate",
      age: "45d",
    },
    pods: [
      { name: "geass-favorites-jkl78-i5j", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-one", age: "12d", cpuUsage: "30m", memoryUsage: "128Mi" },
    ],
    k8sService: { type: "ClusterIP", clusterIP: "10.43.128.41", ports: "80/TCP→8080" },
    ingresses: [],
    apm: {
      rps: 42.7,
      successRate: 0.999,
      errorRate: 0.001,
      p99Ms: 95,
      avgMs: 18,
      spanCount: 15372,
      errorCount: 15,
    },
    slo: {
      meshSuccessRate: 0.999,
      meshRps: 42.7,
      meshP99Ms: 90,
      mtlsEnabled: true,
    },
    logs: { errorCount: 3, warnCount: 24, totalCount: 8940 },
    infra: {
      podCount: 1,
      nodes: [{ name: "desk-one", cpuPct: 72.5, memPct: 68.3 }],
    },
  },
  {
    name: "geass-history",
    namespace: "geass",
    status: "healthy",
    deployment: {
      name: "geass-history",
      replicas: "1/1",
      image: "geass-history:v1.5.0",
      strategy: "RollingUpdate",
      age: "38d",
    },
    pods: [
      { name: "geass-history-mno90-k6l", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-two", age: "8d", cpuUsage: "55m", memoryUsage: "180Mi" },
    ],
    k8sService: { type: "ClusterIP", clusterIP: "10.43.128.48", ports: "80/TCP→8080" },
    ingresses: [],
    apm: {
      rps: 67.4,
      successRate: 0.997,
      errorRate: 0.003,
      p99Ms: 120,
      avgMs: 28,
      spanCount: 24264,
      errorCount: 72,
    },
    slo: {
      meshSuccessRate: 0.998,
      meshRps: 67.4,
      meshP99Ms: 115,
      mtlsEnabled: true,
    },
    logs: { errorCount: 8, warnCount: 45, totalCount: 12780 },
    infra: {
      podCount: 1,
      nodes: [{ name: "desk-two", cpuPct: 65.1, memPct: 71.2 }],
    },
  },
  {
    name: "geass-user",
    namespace: "geass",
    status: "healthy",
    deployment: {
      name: "geass-user",
      replicas: "2/2",
      image: "geass-user:v2.0.1",
      strategy: "RollingUpdate",
      age: "20d",
    },
    pods: [
      { name: "geass-user-pqr12-m7n", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-one", age: "6d", cpuUsage: "65m", memoryUsage: "200Mi" },
      { name: "geass-user-pqr12-o8p", phase: "Running", ready: "1/1", restarts: 0, nodeName: "desk-two", age: "6d", cpuUsage: "60m", memoryUsage: "195Mi" },
    ],
    k8sService: { type: "ClusterIP", clusterIP: "10.43.128.55", ports: "80/TCP→8080" },
    ingresses: [],
    apm: {
      rps: 78.1,
      successRate: 0.996,
      errorRate: 0.004,
      p99Ms: 135,
      avgMs: 32,
      spanCount: 28116,
      errorCount: 112,
    },
    slo: {
      meshSuccessRate: 0.997,
      meshRps: 78.1,
      meshP99Ms: 130,
      mtlsEnabled: true,
    },
    logs: { errorCount: 15, warnCount: 67, totalCount: 15340 },
    infra: {
      podCount: 2,
      nodes: [
        { name: "desk-one", cpuPct: 72.5, memPct: 68.3 },
        { name: "desk-two", cpuPct: 65.1, memPct: 71.2 },
      ],
    },
  },
];

export function mockGetObserveHealth(): LandingPageResponse {
  const services = mockServices;
  const healthy = services.filter((s) => s.status === "healthy").length;
  const warning = services.filter((s) => s.status === "warning").length;
  const critical = services.filter((s) => s.status === "critical").length;

  const totalRps = services.reduce((sum, s) => sum + (s.apm?.rps ?? 0), 0);
  const avgSuccessRate =
    services.reduce((sum, s) => sum + (s.apm?.successRate ?? 1), 0) / services.length;
  const totalErrorCount = services.reduce((sum, s) => sum + (s.logs?.errorCount ?? 0), 0);

  return {
    overview: {
      totalServices: services.length,
      healthyServices: healthy,
      warningServices: warning,
      criticalServices: critical,
      totalRps: Math.round(totalRps * 10) / 10,
      avgSuccessRate: Math.round(avgSuccessRate * 10000) / 10000,
      sloCompliance: (healthy + warning) / services.length,
      totalNodes: 6,
      onlineNodes: 6,
      avgCpuPct: 45.2,
      avgMemPct: 52.8,
      totalErrorCount,
    },
    services,
  };
}

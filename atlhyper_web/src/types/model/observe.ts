/**
 * Observe Landing Page 数据模型 — 对齐 model_v3/observe/health.go
 *
 * 以 Service 为核心，关联 K8s 资源（ClusterSnapshot）+ 可观测信号（OTelSnapshot）
 */

import type { HealthStatus } from "./apm";

// ============================================================================
// 可观测信号（OTelSnapshot）
// ============================================================================

export interface ServiceAPM {
  rps: number;
  successRate: number;
  errorRate: number;
  p99Ms: number;
  avgMs: number;
  spanCount: number;
  errorCount: number;
}

export interface IngressBrief {
  domain: string;
  successRate: number;
  rps: number;
  p99Ms: number;
}

export interface ServiceSLO {
  meshSuccessRate?: number;
  meshRps?: number;
  meshP99Ms?: number;
  mtlsEnabled: boolean;
  ingressDomains?: IngressBrief[];
}

export interface ServiceLogs {
  errorCount: number;
  warnCount: number;
  totalCount: number;
}

export interface NodeBrief {
  name: string;
  cpuPct: number;
  memPct: number;
}

export interface ServiceInfra {
  podCount: number;
  nodes: NodeBrief[];
}

// ============================================================================
// K8s 资源（ClusterSnapshot）
// ============================================================================

export interface DeploymentBrief {
  name: string;
  replicas: string;     // "3/3" (ready/desired)
  image: string;        // 第一个容器镜像
  strategy: string;     // "RollingUpdate"
  age: string;
}

export interface PodBrief {
  name: string;
  phase: string;        // "Running" | "Pending" | "Failed"
  ready: string;        // "1/1"
  restarts: number;
  nodeName: string;
  age: string;
  cpuUsage: string;     // "100m"
  memoryUsage: string;  // "256Mi"
}

export interface K8sIngressBrief {
  name: string;
  hosts: string[];
  tlsEnabled: boolean;
  paths: { path: string; serviceName: string; port: number }[];
}

export interface K8sServiceBrief {
  type: string;         // "ClusterIP" | "NodePort" | "LoadBalancer"
  clusterIP: string;
  ports: string;        // "80/TCP→8080, 443/TCP→8443"
}

// ============================================================================
// 服务健康（核心聚合）
// ============================================================================

export interface ServiceHealth {
  name: string;
  namespace: string;
  status: HealthStatus;

  // K8s 资源（ClusterSnapshot）
  deployment?: DeploymentBrief;
  pods?: PodBrief[];
  k8sService?: K8sServiceBrief;
  ingresses?: K8sIngressBrief[];

  // 可观测信号（OTelSnapshot）
  apm?: ServiceAPM;
  slo?: ServiceSLO;
  logs?: ServiceLogs;

  // 基础设施
  infra?: ServiceInfra;
}

export interface HealthOverview {
  totalServices: number;
  healthyServices: number;
  warningServices: number;
  criticalServices: number;
  totalRps: number;
  avgSuccessRate: number;
  sloCompliance: number;
  totalNodes: number;
  onlineNodes: number;
  avgCpuPct: number;
  avgMemPct: number;
  totalErrorCount: number;
}

export interface LandingPageResponse {
  overview: HealthOverview;
  services: ServiceHealth[];
}

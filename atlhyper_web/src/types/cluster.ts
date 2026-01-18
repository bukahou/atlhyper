/**
 * 集群相关类型定义
 */

import type { ClusterRequest, NamespaceRequest, ResourceStatus, NodeStatus } from "./common";

// 集群信息（Master V2 使用 snake_case）
export interface ClusterInfo {
  cluster_id: string;
  status: string;
  last_seen: string;
  node_count: number;
  pod_count: number;
}

// 旧集群信息类型（兼容）
export interface ClusterInfoLegacy {
  ClusterID: string;
  ClusterName: string;
}

// Pod 概览 - 匹配后端 API 返回格式
export interface PodOverview {
  cards: {
    running: number;
    pending: number;
    failed: number;
    unknown: number;
  };
  pods: PodItem[];
}

// Pod 列表项 - 匹配后端 API 返回格式
export interface PodItem {
  name: string;
  namespace: string;
  deployment: string;
  ready: string;
  phase: string; // Running, Pending, Failed, Unknown
  restarts: number;
  cpu: number;
  cpuPercent: number;
  memory: number;
  memPercent: number;
  cpuText: string;
  cpuPercentText: string;
  memoryText: string;
  memPercentText: string;
  startTime: string;
  node: string;
  age?: string;
}

// Pod 详情请求
export interface PodDetailRequest extends NamespaceRequest {
  PodName: string;
}

// Pod 容器端口
export interface ContainerPort {
  containerPort: number;
  protocol: string;
  name?: string;
}

// Pod 容器环境变量
export interface ContainerEnv {
  name: string;
  value?: string;
}

// Pod 容器挂载
export interface ContainerVolumeMount {
  name: string;
  mountPath: string;
  readOnly?: boolean;
  subPath?: string;
}

// Pod 容器（详情用）
export interface PodContainerDetail {
  name: string;
  image: string;
  imagePullPolicy?: string;
  ports?: ContainerPort[];
  envs?: ContainerEnv[];
  volumeMounts?: ContainerVolumeMount[];
  requests?: Record<string, string>;
  limits?: Record<string, string>;
  readinessProbe?: unknown;
  livenessProbe?: unknown;
  startupProbe?: unknown;
  securityContext?: unknown;
  state?: string;
  restartCount?: number;
  lastTerminatedReason?: string;
}

// Pod 卷
export interface PodVolume {
  name: string;
  type: string;
  sourceBrief?: string;
  sourceRaw?: unknown;
}

// Pod 详情（匹配后端 PodDetailDTO）
export interface PodDetail {
  // 基本信息
  name: string;
  namespace: string;
  controller?: string;
  phase: string;
  ready: string;
  restarts: number;
  startTime: string;
  age?: string;
  node: string;
  podIP?: string;
  qosClass?: string;
  reason?: string;
  message?: string;
  badges?: string[];

  // 调度/策略
  restartPolicy?: string;
  priorityClassName?: string;
  runtimeClassName?: string;
  terminationGracePeriodSeconds?: number;
  tolerations?: unknown;
  affinity?: unknown;
  topologySpreadConstraints?: unknown;
  nodeSelector?: Record<string, string>;

  // 网络
  hostNetwork?: boolean;
  hostIP?: string;
  podIPs?: string[];
  dnsPolicy?: string;
  serviceAccountName?: string;

  // 指标
  cpuUsage?: string;
  cpuLimit?: string;
  cpuUtilPct?: number;
  memUsage?: string;
  memLimit?: string;
  memUtilPct?: number;

  // 容器和卷
  containers: PodContainerDetail[];
  volumes?: PodVolume[];
}

// Pod 事件
export interface PodEvent {
  Type: string;
  Reason: string;
  Message: string;
  LastTimestamp: string;
}

// Pod 日志请求
export interface PodLogsRequest extends NamespaceRequest {
  Pod: string;
  Container?: string;
  TailLines?: number;
  TimeoutSeconds?: number;
}

// Pod 日志响应
export interface PodLogsResponse {
  commandID: string;
  status: string;
  logs: string;
  errorCode?: number;
}

// Pod 操作请求
export interface PodOperationRequest extends NamespaceRequest {
  Pod: string;
}

// Node 概览 - 匹配后端 API 返回格式
export interface NodeOverview {
  cards: {
    totalNodes: number;
    readyNodes: number;
    totalCPU: number;
    totalMemoryGiB: number;
  };
  rows: NodeItem[];
}

// Node 列表项 - 匹配后端 API 返回格式
export interface NodeItem {
  name: string;
  ready: boolean;
  internalIP: string;
  osImage: string;
  architecture: string;
  cpuCores: number;
  memoryGiB: number;
  schedulable: boolean;
}

// Node 详情请求
export interface NodeDetailRequest extends ClusterRequest {
  NodeName: string;
}

// Node 操作请求（cordon/uncordon）
export interface NodeOperationRequest extends ClusterRequest {
  Node: string;
}

// Node Condition
export interface NodeCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  heartbeat?: string;
  changedAt?: string;
}

// Node Taint
export interface NodeTaint {
  key: string;
  value?: string;
  effect: string;
}

// Node 详情（匹配后端 NodeDetailDTO）
export interface NodeDetail {
  // 基本
  name: string;
  roles?: string[];
  ready: boolean;
  schedulable: boolean;
  age?: string;
  createdAt: string;

  // 地址与系统
  hostname?: string;
  internalIP?: string;
  externalIP?: string;
  osImage?: string;
  os?: string;
  architecture?: string;
  kernel?: string;
  cri?: string;
  kubelet?: string;
  kubeProxy?: string;

  // 资源（容量/可分配）
  cpuCapacityCores?: number;
  cpuAllocatableCores?: number;
  memCapacityGiB?: number;
  memAllocatableGiB?: number;
  podsCapacity?: number;
  podsAllocatable?: number;
  ephemeralStorageGiB?: number;

  // 当前指标
  cpuUsageCores?: number;
  cpuUtilPct?: number;
  memUsageGiB?: number;
  memUtilPct?: number;
  podsUsed?: number;
  podsUtilPct?: number;
  pressureMemory?: boolean;
  pressureDisk?: boolean;
  pressurePID?: boolean;
  networkUnavailable?: boolean;

  // 调度相关
  podCIDRs?: string[];
  providerID?: string;

  // 条件/污点/标签
  conditions?: NodeCondition[];
  taints?: NodeTaint[];
  labels?: Record<string, string>;

  // 诊断/徽标
  badges?: string[];
  reason?: string;
  message?: string;
}

// Deployment 概览 - 匹配后端 API 返回格式
export interface DeploymentOverview {
  cards: {
    totalDeployments: number;
    namespaces: number;
    totalReplicas: number;
    readyReplicas: number;
  };
  rows: DeploymentItem[];
}

// Deployment 列表项 - 匹配后端 API 返回格式
export interface DeploymentItem {
  name: string;
  namespace: string;
  image: string;
  replicas: string; // "1/1" 格式
  labelCount: number;
  annoCount: number;
  createdAt: string;
}

// Deployment 详情请求
export interface DeploymentDetailRequest extends NamespaceRequest {
  Name: string;
}

// Workload 扩缩容请求（匹配后端 /ops/workload/scale）
export interface WorkloadScaleRequest extends NamespaceRequest {
  Name: string;
  Kind?: string; // 默认 Deployment
  Replicas: number;
}

// Workload 更新镜像请求（匹配后端 /ops/workload/updateImage）
export interface WorkloadUpdateImageRequest extends NamespaceRequest {
  Name: string;
  Kind?: string; // 默认 Deployment
  NewImage: string;
  OldImage?: string;
}

// 容器端口
export interface ContainerPortSpec {
  name?: string;
  containerPort: number;
  protocol?: string;
  hostPort?: number;
}

// 环境变量
export interface EnvVarSpec {
  name: string;
  value?: string;
  valueFrom?: string;
}

// 卷挂载
export interface VolumeMountSpec {
  name: string;
  mountPath: string;
  subPath?: string;
  readOnly?: boolean;
}

// 探针
export interface ProbeSpec {
  type: string; // httpGet, tcpSocket, exec
  path?: string;
  port?: number;
  command?: string;
  initialDelaySeconds?: number;
  periodSeconds?: number;
  timeoutSeconds?: number;
  successThreshold?: number;
  failureThreshold?: number;
}

// Deployment 容器信息
export interface DeploymentContainer {
  name: string;
  image: string;
  imagePullPolicy?: string;
  command?: string[];
  args?: string[];
  workingDir?: string;
  ports?: ContainerPortSpec[];
  envs?: EnvVarSpec[];
  volumeMounts?: VolumeMountSpec[];
  requests?: Record<string, string>;
  limits?: Record<string, string>;
  livenessProbe?: ProbeSpec;
  readinessProbe?: ProbeSpec;
  startupProbe?: ProbeSpec;
}

// Deployment Condition
export interface DeploymentCondition {
  type: string;
  status: string;
  reason?: string;
  message?: string;
  lastUpdateTime?: string;
  lastTransitionTime?: string;
}

// Deployment ReplicaSet 信息
export interface DeploymentReplicaSet {
  name: string;
  replicas: number;
  ready: number;
  available: number;
  revision?: string;
  image?: string;
  createdAt?: string;
}

// Deployment Spec DTO
export interface DeploymentSpec {
  replicas?: number;
  minReadySeconds?: number;
  revisionHistoryLimit?: number;
  progressDeadlineSeconds?: number;
  strategyType?: string;
  maxUnavailable?: string;
  maxSurge?: string;
  matchLabels?: Record<string, string>;
}

// 容忍
export interface TolerationSpec {
  key?: string;
  operator?: string;
  value?: string;
  effect?: string;
  tolerationSeconds?: number;
}

// 亲和性（简化）
export interface AffinitySpec {
  nodeAffinity?: string;
  podAffinity?: string;
  podAntiAffinity?: string;
}

// 卷定义
export interface VolumeSpec {
  name: string;
  type: string;
  source?: string;
}

// Deployment Template DTO
export interface DeploymentTemplate {
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  containers: DeploymentContainer[];
  volumes?: VolumeSpec[];
  serviceAccountName?: string;
  nodeSelector?: Record<string, string>;
  tolerations?: TolerationSpec[];
  affinity?: AffinitySpec;
  hostNetwork?: boolean;
  dnsPolicy?: string;
  runtimeClassName?: string;
  imagePullSecrets?: string[];
}

// Deployment Status DTO
export interface DeploymentStatus {
  observedGeneration?: number;
  replicas?: number;
  updatedReplicas?: number;
  readyReplicas?: number;
  availableReplicas?: number;
  unavailableReplicas?: number;
  collisionCount?: number;
}

// Deployment 详情（匹配后端 DeploymentDetailDTO）
export interface DeploymentDetail {
  // 基本信息
  name: string;
  namespace: string;
  strategy: string;
  replicas: number;
  updated: number;
  ready: number;
  available: number;
  unavailable?: number;
  paused?: boolean;
  selector?: string;
  createdAt: string;
  age?: string;

  // 嵌套结构
  spec: DeploymentSpec;
  template: DeploymentTemplate;
  status: DeploymentStatus;

  // Conditions
  conditions?: DeploymentCondition[];

  // Rollout
  rollout?: {
    phase: string;
    message?: string;
    badges?: string[];
  };

  // ReplicaSets
  replicaSets?: DeploymentReplicaSet[];

  // Labels & Annotations
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

// Service 概览 - 匹配后端 API 返回格式
export interface ServiceOverview {
  cards: {
    totalServices: number;
    externalServices: number;
    internalServices: number;
    headlessServices: number;
  };
  rows: ServiceItem[];
}

// Service 列表项 - 匹配后端 API 返回格式
export interface ServiceItem {
  name: string;
  namespace: string;
  type: string;
  clusterIP: string;
  ports: string;
  protocol: string;
  selector: string;
  createdAt: string;
}

// Service 详情请求
export interface ServiceDetailRequest {
  ClusterID: string;
  Namespace: string;
  Name: string;
}

// Service 端口 DTO
export interface ServicePort {
  name?: string;
  protocol: string;
  port: number;
  targetPort: string;
  nodePort?: number;
  appProtocol?: string;
}

// Service 后端端点 DTO
export interface BackendEndpoint {
  address: string;
  ready: boolean;
  nodeName?: string;
  zone?: string;
  targetRef?: {
    kind?: string;
    namespace?: string;
    name?: string;
    uid?: string;
  };
}

// Service 后端 DTO
export interface ServiceBackends {
  ready: number;
  notReady: number;
  total: number;
  slices?: number;
  updated?: string;
  ports?: { name?: string; port: number; protocol: string; appProtocol?: string }[];
  endpoints?: BackendEndpoint[];
}

// Service 详情 DTO
export interface ServiceDetail {
  // 基本信息
  name: string;
  namespace: string;
  type: string;
  createdAt: string;
  age?: string;

  // 选择器 & 端口
  selector?: Record<string, string>;
  ports?: ServicePort[];

  // 网络信息
  clusterIPs?: string[];
  externalIPs?: string[];
  loadBalancerIngress?: string[];

  // 重要 spec
  sessionAffinity?: string;
  sessionAffinityTimeoutSeconds?: number;
  externalTrafficPolicy?: string;
  internalTrafficPolicy?: string;
  ipFamilies?: string[];
  ipFamilyPolicy?: string;
  loadBalancerClass?: string;
  loadBalancerSourceRanges?: string[];
  allocateLoadBalancerNodePorts?: boolean;
  healthCheckNodePort?: number;
  externalName?: string;

  // 端点聚合
  backends?: ServiceBackends;

  // 徽标
  badges?: string[];
}

// Namespace 概览 - 匹配后端 API 返回格式
export interface NamespaceOverview {
  cards: {
    totalNamespaces: number;
    activeCount: number;
    terminating: number;
    totalPods: number;
  };
  rows: NamespaceItem[];
}

// Namespace 列表项 - 匹配后端 API 返回格式
export interface NamespaceItem {
  name: string;
  status: string;
  podCount: number;
  labelCount: number;
  annotationCount: number;
  createdAt: string;
}

// Namespace 详情请求
export interface NamespaceDetailRequest {
  ClusterID: string;
  Namespace: string;
}

// ResourceQuota DTO
export interface ResourceQuotaDTO {
  name: string;
  scopes?: string[];
  hard?: Record<string, string>;
  used?: Record<string, string>;
}

// LimitRange DTO
export interface LimitRangeDTO {
  name: string;
  items: {
    type: string;
    max?: Record<string, string>;
    min?: Record<string, string>;
    default?: Record<string, string>;
    defaultRequest?: Record<string, string>;
    maxLimitRequestRatio?: Record<string, string>;
  }[];
}

// Namespace Metrics DTO
export interface NamespaceMetricsDTO {
  cpu: {
    usage: string;
    requests?: string;
    limits?: string;
    utilPct?: number;
    utilBasis?: string;
    quotaHard?: string;
    quotaUsed?: string;
  };
  memory: {
    usage: string;
    requests?: string;
    limits?: string;
    utilPct?: number;
    utilBasis?: string;
    quotaHard?: string;
    quotaUsed?: string;
  };
}

// Namespace 详情 DTO
export interface NamespaceDetail {
  // 基本信息
  name: string;
  phase: string;
  createdAt: string;
  age?: string;

  // 标签和注解
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  labelCount: number;
  annotationCount: number;

  // 资源计数
  pods: number;
  podsRunning: number;
  podsPending: number;
  podsFailed: number;
  podsSucceeded: number;
  deployments: number;
  statefulSets: number;
  daemonSets: number;
  jobs: number;
  cronJobs: number;
  services: number;
  ingresses: number;
  configMaps: number;
  secrets: number;
  persistentVolumeClaims: number;
  networkPolicies: number;
  serviceAccounts: number;

  // 配额和限制
  quotas?: ResourceQuotaDTO[];
  limitRanges?: LimitRangeDTO[];

  // 指标
  metrics?: NamespaceMetricsDTO;

  // 徽标
  badges?: string[];
}

// ConfigMap 请求
export interface ConfigMapRequest {
  ClusterID: string;
  Namespace: string;
}

// ConfigMap 数据条目
export interface ConfigMapDataEntry {
  key: string;
  size: number;
  preview?: string;
  truncated?: boolean;
}

// ConfigMap 二进制条目
export interface ConfigMapBinaryEntry {
  key: string;
  size: number;
}

// ConfigMap DTO
export interface ConfigMapDTO {
  name: string;
  namespace: string;
  createdAt?: string;
  age?: string;
  immutable?: boolean;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  keys: number;
  binaryKeys: number;
  totalSizeBytes: number;
  binaryTotalSizeBytes: number;
  data?: ConfigMapDataEntry[];
  binary?: ConfigMapBinaryEntry[];
}

// Secret DTO - 匹配后端 model_v2.Secret
export interface SecretDTO {
  name: string;
  namespace: string;
  uid?: string;
  createdAt?: string;
  age?: string;
  type: string; // Opaque, kubernetes.io/tls, etc.
  dataKeys?: string[]; // 只有键名，不含值
}

// Ingress 概览 - 匹配后端 API 返回格式
export interface IngressOverview {
  cards: {
    totalIngresses: number;
    usedHosts: number;
    tlsCerts: number;
    totalPaths: number;
  };
  rows: IngressItem[];
}

// Ingress 列表项 - 匹配后端 API 返回格式
export interface IngressItem {
  name: string;
  namespace: string;
  host: string;
  path: string;
  serviceName: string;
  servicePort: string;
  tls: string;
  createdAt: string;
}

// Ingress 详情请求
export interface IngressDetailRequest {
  ClusterID: string;
  Namespace: string;
  Name: string;
}

// Ingress 详情
export interface IngressDetail {
  name: string;
  namespace: string;
  class?: string;
  controller?: string;
  hosts?: string[];
  tlsEnabled: boolean;
  loadBalancer?: string[];
  createdAt: string;
  age?: string;
  spec: IngressSpecDTO;
  status: IngressStatusDTO;
  annotations?: Record<string, string>;
}

export interface IngressSpecDTO {
  ingressClassName?: string;
  loadBalancerSourceRanges?: string[];
  defaultBackend?: IngressBackendRef;
  rules?: IngressRuleDTO[];
  tls?: IngressTLSDTO[];
}

export interface IngressRuleDTO {
  host?: string;
  paths: IngressPathDTO[];
}

export interface IngressPathDTO {
  path?: string;
  pathType?: string;
  backend: IngressBackendRef;
}

export interface IngressBackendRef {
  type: string; // "Service" | "Resource"
  service?: {
    name: string;
    portName?: string;
    portNumber?: number;
  };
  resource?: {
    apiGroup?: string;
    kind: string;
    name: string;
    namespace?: string;
  };
}

export interface IngressTLSDTO {
  secretName: string;
  hosts?: string[];
}

export interface IngressStatusDTO {
  loadBalancer?: string[];
}

// 事件日志 - 匹配后端 EventLog 结构
export interface EventLog {
  ClusterID: string;
  Category: string;
  EventTime: string;
  Kind: string;
  Message: string;
  Name: string;
  Namespace: string;
  Node: string;
  Reason: string;
  Severity: string;
  Time: string;
}

// 事件日志请求
export interface EventLogRequest {
  clusterID: string;
  withinDays: number;
}

// 事件统计卡片
export interface EventCards {
  totalAlerts: number;
  totalEvents: number;
  warning: number;
  info: number;
  error: number;
  categoriesCount: number;
  kindsCount: number;
}

// 事件概览响应
export interface EventOverview {
  cards: EventCards;
  rows: EventLog[];
}

// ==================== 主机指标 ====================

// 指标概览卡片
export interface MetricsOverviewCards {
  avgCPUPercent: number;
  avgMemPercent: number;
  peakTempC: number;
  peakTempNode: string;
  peakDiskPercent: number;
  peakDiskNode: string;
}

// 节点指标行
export interface NodeMetricsRow {
  node: string;
  cpuPercent: number;
  memPercent: number;
  cpuTempC: number;
  diskUsedPercent: number;
  eth0TxKBps: number;
  eth0RxKBps: number;
  topCPUProcess: string;
  timestamp: string;
}

// 指标概览响应
export interface MetricsOverview {
  cards: MetricsOverviewCards;
  rows: NodeMetricsRow[];
}

// 节点指标时序
export interface NodeMetricsSeries {
  at: string[];
  cpuPct: number[];
  memPct: number[];
  tempC: number[];
  diskPct: number[];
  eth0TxKBps: number[];
  eth0RxKBps: number[];
}

// Top CPU 进程
export interface TopCPUProcess {
  pid: number;
  user: string;
  command: string;
  cpuPercent: number;
  cpuUsage: string;
}

// 节点指标详情请求
export interface MetricsDetailRequest {
  clusterID: string;
  nodeID: string;
}

// 节点指标详情响应
export interface NodeMetricsDetail {
  node: string;
  latest: NodeMetricsRow;
  series: NodeMetricsSeries;
  processes: TopCPUProcess[];
  timeRange: {
    since: string;
    until: string;
  };
}

// AI 分析详情
export interface AnalysisDetail {
  eventId: string;
  analysis: string;
  suggestions: string[];
  severity: "low" | "medium" | "high" | "critical";
}

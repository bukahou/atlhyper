/**
 * 查询键工厂
 *
 * 键的形状固定为 [域, 资源, 范围]。页面不手写键，一律用这里的工厂：
 * - clusterId 进键：切换集群即切换缓存，旧集群的数据不会显示在新集群下
 * - mode 进键：真实数据与演示数据永不共用缓存
 */
import type { DataSourceMode } from "@/config/data-source";
import type { CorrelationSubject, TraceFilter } from "@/datasource/apm";
import type { LogHistogramParams, LogQueryParams } from "@/datasource/logs";
import type { TimeRangeParams } from "@/lib/time-range";

export const sessionKeys = {
  clusterList: () => ["session", "cluster-list"] as const,
};

// 其余域的键按域分文件，统一从这里导出：使用方只认 keys.ts 这一个入口
export * from "./keys-aiops";
export * from "./keys-management";

export const overviewKeys = {
  cluster: (clusterId: string, mode: DataSourceMode) => ["overview", "cluster", { clusterId, mode }] as const,
};

/**
 * 集群资源的种类。
 *
 * 取值与数据源模块的 key（config/data-source.ts 的注册表）一致，
 * 所以同一个值既用来拼键，也用来读该模块当前是真实数据还是演示数据。
 */
export type ClusterResourceKind =
  | "pod"
  | "node"
  | "deployment"
  | "service"
  | "namespace"
  | "ingress"
  | "statefulset"
  | "daemonset"
  | "job"
  | "cronjob"
  | "pv"
  | "pvc"
  | "netpol"
  | "quota"
  | "limit"
  | "sa";

/** 带 namespace 的资源里的某一个对象（详情弹窗指向的那一行） */
export interface NamespacedTarget {
  namespace: string;
  name: string;
}

/** 某个资源自己的事件时间线指向的对象；集群级资源没有 namespace */
export interface EventSubject {
  kind: string;
  namespace?: string;
  name: string;
}

/** 一路日志：哪个 Pod 的哪个容器、取最后多少行 */
export interface PodLogStream {
  namespace: string;
  podName: string;
  /** 空串 = 不指定容器，由后端取默认容器 */
  containerName: string;
  tailLines: number;
}

/** 某类资源的全部列表查询（不分集群与数据源）。只用来按前缀失效 */
const resourceLists = (kind: ClusterResourceKind) => ["cluster", `${kind}-list`] as const;
/** 某类资源的全部详情查询（不分集群与数据源）。只用来按前缀失效 */
const resourceDetails = (kind: ClusterResourceKind) => ["cluster", `${kind}-detail`] as const;

export const clusterKeys = {
  events: (clusterId: string, mode: DataSourceMode) => ["cluster", "events", { clusterId, mode }] as const,

  resourceLists,
  resourceList: (kind: ClusterResourceKind, clusterId: string, mode: DataSourceMode) =>
    ["cluster", `${kind}-list`, { clusterId, mode }] as const,

  resourceDetails,
  /** 详情的键带 namespace 与 name：连续点不同行时，显示的一定是最后点的那一行 */
  resourceDetail: (
    kind: ClusterResourceKind,
    clusterId: string,
    mode: DataSourceMode,
    target: { namespace?: string; name: string },
  ) => ["cluster", `${kind}-detail`, { clusterId, mode, namespace: target.namespace ?? "", name: target.name }] as const,

  /** 写操作之后要重新取数的范围：该类资源的全部列表与详情 */
  resourceScope: (kind: ClusterResourceKind) => [resourceLists(kind), resourceDetails(kind)] as const,

  configMapList: (clusterId: string, mode: DataSourceMode, namespace: string) =>
    ["cluster", "configmap-list", { clusterId, mode, namespace }] as const,
  secretList: (clusterId: string, mode: DataSourceMode, namespace: string) =>
    ["cluster", "secret-list", { clusterId, mode, namespace }] as const,
  configMapData: (clusterId: string, mode: DataSourceMode, target: NamespacedTarget) =>
    ["cluster", "configmap-data", { clusterId, mode, namespace: target.namespace, name: target.name }] as const,
  secretData: (clusterId: string, mode: DataSourceMode, target: NamespacedTarget) =>
    ["cluster", "secret-data", { clusterId, mode, namespace: target.namespace, name: target.name }] as const,

  resourceEvents: (clusterId: string, mode: DataSourceMode, subject: EventSubject) =>
    [
      "cluster",
      "resource-events",
      { clusterId, mode, kind: subject.kind, namespace: subject.namespace ?? "", name: subject.name },
    ] as const,

  /** 行数进键：换行数就是另一份数据。stream 为 null 表示查看器关着，没有要取的日志 */
  podLogs: (clusterId: string, mode: DataSourceMode, stream: PodLogStream | null) =>
    [
      "cluster",
      "pod-logs",
      {
        clusterId,
        mode,
        namespace: stream?.namespace ?? "",
        name: stream?.podName ?? "",
        container: stream?.containerName ?? "",
        tailLines: stream?.tailLines ?? 0,
      },
    ] as const,
};

// ============================================================
// 观测信号（Metrics / Logs / APM / SLO）
//
// 每个信号一个域，`all` 是该域的前缀：页头的手动刷新与写操作后的失效按它匹配。
// 时间范围进键时用 TimeRangeParams —— 相对范围只有描述（"1h"），没有现算的时间戳，键才稳定。
// ============================================================

export const observeKeys = {
  /** 各信号的数据新鲜度。不分信号：四个观测页共用同一份 */
  signalFreshness: (clusterId: string, mode: DataSourceMode) =>
    ["observe", "signal-freshness", { clusterId, mode }] as const,
};

export const metricsKeys = {
  all: ["metrics"] as const,
  snapshot: (clusterId: string, mode: DataSourceMode) => ["metrics", "snapshot", { clusterId, mode }] as const,
  hardware: (clusterId: string, mode: DataSourceMode) => ["metrics", "hardware", { clusterId, mode }] as const,
  /** 节点与小时数进键：每个「节点 + 时间范围」各自缓存 */
  nodeHistory: (clusterId: string, mode: DataSourceMode, nodeName: string, hours: number) =>
    ["metrics", "node-history", { clusterId, mode, nodeName, hours }] as const,
};

export const logsKeys = {
  all: ["logs"] as const,
  list: (clusterId: string, mode: DataSourceMode, params: LogQueryParams) =>
    ["logs", "list", { clusterId, mode, ...params }] as const,
  histogram: (clusterId: string, mode: DataSourceMode, params: LogHistogramParams) =>
    ["logs", "histogram", { clusterId, mode, ...params }] as const,
  /** 某条链路关联的日志（链路详情里的「日志」标签页） */
  traceLogs: (clusterId: string, mode: DataSourceMode, traceId: string, limit: number) =>
    ["logs", "trace-logs", { clusterId, mode, traceId, limit }] as const,
};

export const apmKeys = {
  all: ["apm"] as const,
  services: (clusterId: string, mode: DataSourceMode, range: TimeRangeParams) =>
    ["apm", "services", { clusterId, mode, ...range }] as const,
  topology: (clusterId: string, mode: DataSourceMode, range: TimeRangeParams) =>
    ["apm", "topology", { clusterId, mode, ...range }] as const,
  operations: (clusterId: string, mode: DataSourceMode, range: TimeRangeParams) =>
    ["apm", "operations", { clusterId, mode, ...range }] as const,
  traces: (clusterId: string, mode: DataSourceMode, filter: TraceFilter, range: TimeRangeParams) =>
    ["apm", "traces", { clusterId, mode, ...filter, ...range }] as const,
  /** 某个操作的链路样本序列（链路详情里前后翻样本用） */
  operationSamples: (clusterId: string, mode: DataSourceMode, filter: TraceFilter, range: TimeRangeParams) =>
    ["apm", "operation-samples", { clusterId, mode, ...filter, ...range }] as const,
  /** traceId 进键：快速切换链路时，显示的一定是最后点的那条 */
  traceDetail: (clusterId: string, mode: DataSourceMode, traceId: string) =>
    ["apm", "trace-detail", { clusterId, mode, traceId }] as const,
  serviceSeries: (clusterId: string, mode: DataSourceMode, serviceName: string, range: TimeRangeParams) =>
    ["apm", "service-series", { clusterId, mode, serviceName, ...range }] as const,
  httpStats: (clusterId: string, mode: DataSourceMode, serviceName: string, range: TimeRangeParams) =>
    ["apm", "http-stats", { clusterId, mode, serviceName, ...range }] as const,
  dbStats: (clusterId: string, mode: DataSourceMode, serviceName: string, range: TimeRangeParams) =>
    ["apm", "db-stats", { clusterId, mode, serviceName, ...range }] as const,
  correlations: (clusterId: string, mode: DataSourceMode, subject: CorrelationSubject, range: TimeRangeParams) =>
    [
      "apm",
      "correlations",
      {
        clusterId,
        mode,
        serviceName: subject.serviceName,
        correlationMode: subject.correlationMode,
        operation: subject.operation ?? "",
        ...range,
      },
    ] as const,
};

export const sloKeys = {
  all: ["slo"] as const,
  /** window 是 SLO 的预聚合窗口（1h / 6h / 24h / 3d / 7d），不是任意时间范围 */
  domains: (clusterId: string, mode: DataSourceMode, window: string) =>
    ["slo", "domains", { clusterId, mode, window }] as const,
  domainHistory: (clusterId: string, mode: DataSourceMode, host: string, window: string) =>
    ["slo", "domain-history", { clusterId, mode, host, window }] as const,
  latency: (clusterId: string, mode: DataSourceMode, domain: string, window: string) =>
    ["slo", "latency", { clusterId, mode, domain, window }] as const,
};

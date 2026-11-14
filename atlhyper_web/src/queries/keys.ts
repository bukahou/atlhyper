/**
 * 查询键工厂
 *
 * 键的形状固定为 [域, 资源, 范围]。页面不手写键，一律用这里的工厂：
 * - clusterId 进键：切换集群即切换缓存，旧集群的数据不会显示在新集群下
 * - mode 进键：真实数据与演示数据永不共用缓存
 */
import type { DataSourceMode } from "@/config/data-source";

export const sessionKeys = {
  clusterList: () => ["session", "cluster-list"] as const,
};

export const adminKeys = {
  permissionTable: () => ["admin", "permission-table"] as const,
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

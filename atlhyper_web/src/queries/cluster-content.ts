/**
 * 集群资源的取数描述 —— 经指令通道读取的内容（Pod 日志、ConfigMap / Secret 的值）
 *
 * 这几样不在快照里：每次请求都是一条下发给 Agent 的指令，由 Agent 现场读取后同步返回。
 * 所以只在用户明确要看时才发 —— 失败不自动重试，切回标签页也不重取。
 */
import { queryOptions, skipToken } from "@tanstack/react-query";
import { getDataSourceMode } from "@/config/data-source";
import { getConfigMapData, getPodLogs, getSecretData } from "@/datasource/cluster";
import { clusterKeys, type NamespacedTarget, type PodLogStream } from "./keys";
import { clusterDataAvailable } from "./availability";

const COMMAND_BACKED_READ = { retry: false, refetchOnWindowFocus: false } as const;

/**
 * Pod 日志。stream 为 null 表示查看器关着，不取数。
 *
 * 同一个容器换行数时先沿用上一份日志（界面不闪成空白）；换了 Pod 或容器就不沿用，
 * 否则会把别的容器的日志当成这一个的显示出来。
 */
export function podLogsQuery(clusterId: string, stream: PodLogStream | null) {
  const queryKey = clusterKeys.podLogs(clusterId, getDataSourceMode("pod"), stream);
  return queryOptions({
    queryKey,
    queryFn:
      stream === null
        ? skipToken
        : ({ signal }) =>
            getPodLogs(
              {
                ClusterID: clusterId,
                Namespace: stream.namespace,
                Pod: stream.podName,
                Container: stream.containerName || undefined,
                TailLines: stream.tailLines,
              },
              signal,
            ),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("pod")) && stream !== null && stream.namespace !== "" && stream.podName !== "",
    placeholderData: (previousLogs, previousQuery) => {
      const previous = previousQuery?.queryKey[2];
      const current = queryKey[2];
      const sameContainer =
        previous !== undefined &&
        previous.clusterId === current.clusterId &&
        previous.namespace === current.namespace &&
        previous.name === current.name &&
        previous.container === current.container;
      return sameContainer ? previousLogs : undefined;
    },
    ...COMMAND_BACKED_READ,
  });
}

/**
 * ConfigMap 的键值内容（用户展开某一项时才取）。
 *
 * 取到后在显示期间不再重取（staleTime 无限）；没有组件显示它时立刻从缓存丢弃（gcTime 0），
 * 内容只留在正在看它的界面里。
 */
export function configMapDataQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.configMapData(clusterId, getDataSourceMode("namespace"), target),
    queryFn: ({ signal }) =>
      getConfigMapData({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("namespace")) && target.namespace !== "" && target.name !== "",
    staleTime: Infinity,
    gcTime: 0,
    ...COMMAND_BACKED_READ,
  });
}

/** Secret 的键值内容（用户展开某一项时才取）。缓存策略同 configMapDataQuery */
export function secretDataQuery(clusterId: string, target: NamespacedTarget) {
  return queryOptions({
    queryKey: clusterKeys.secretData(clusterId, getDataSourceMode("namespace"), target),
    queryFn: ({ signal }) =>
      getSecretData({ ClusterID: clusterId, Namespace: target.namespace, Name: target.name }, signal),
    enabled: clusterDataAvailable(clusterId, getDataSourceMode("namespace")) && target.namespace !== "" && target.name !== "",
    staleTime: Infinity,
    gcTime: 0,
    ...COMMAND_BACKED_READ,
  });
}

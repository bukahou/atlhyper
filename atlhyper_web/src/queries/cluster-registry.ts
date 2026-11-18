/**
 * 集群注册表（集群管理页）的取数与写操作描述
 *
 * 管理域没有 datasource：工厂直接调用 api 层。没有权限时页面整页显示演示数据，
 * 演示数据由页面直接取，不经过这里的查询。
 */
import { mutationOptions, queryOptions, type QueryClient } from "@tanstack/react-query";
import {
  getClusterRegistry,
  issueClusterToken,
  registerCluster,
  unregisterCluster,
  updateClusterProfile,
  type ClusterProfile,
  type ClusterRegistration,
} from "@/api/cluster-registry";
import { adminKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 让注册表重新取数。不等它取完：写操作已经成功，按钮不必为了刷新继续转圈 */
function invalidateRegistry(client: QueryClient): void {
  void client.invalidateQueries({ queryKey: adminKeys.clusterRegistry() });
}

/**
 * 集群列表：已登记的 + 正在上报但未登记的。
 *
 * canManageClusters 由调用方用 useSessionReady 与 useCan("clusters.manage") 得到：
 * 没有权限就不发请求，而不是发出去再被后端拒绝。
 * 慢速轮询：在线状态、最近上报时间、Agent 是否带了令牌都会随时间变化。
 */
export function clusterRegistryQuery(canManageClusters: boolean) {
  return queryOptions({
    queryKey: adminKeys.clusterRegistry(),
    queryFn: async ({ signal }) => (await getClusterRegistry(signal)).data.data ?? [],
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
    enabled: canManageClusters,
  });
}

/** 登记集群；返回登记后的那一行。集群 ID 已登记时后端返回 409，错误原样抛给调用方 */
export function registerClusterMutation() {
  return mutationOptions({
    mutationFn: async (registration: ClusterRegistration) => (await registerCluster(registration)).data.data,
    onSuccess: (_registered, _registration, _onMutateResult, context) => invalidateRegistry(context.client),
  });
}

export interface ClusterProfileChange extends ClusterProfile {
  clusterId: string;
}

/** 修改某个集群的显示名与描述；返回修改后的那一行 */
export function updateClusterProfileMutation() {
  return mutationOptions({
    mutationFn: async ({ clusterId, name, description }: ClusterProfileChange) =>
      (await updateClusterProfile(clusterId, { name, description })).data.data,
    onSuccess: (_updated, _change, _onMutateResult, context) => invalidateRegistry(context.client),
  });
}

/** 删除某个集群的登记（令牌随之失效，历史数据保留） */
export function unregisterClusterMutation() {
  return mutationOptions({
    mutationFn: async (clusterId: string) => {
      await unregisterCluster(clusterId);
    },
    onSuccess: (_nothing, _clusterId, _onMutateResult, context) => invalidateRegistry(context.client),
  });
}

export interface ClusterTokenRequest {
  clusterId: string;
  /**
   * 接收令牌明文。只会被调用一次；调用方把明文放进弹窗的本地状态，关闭弹窗即丢弃。
   */
  deliverToken: (token: string) => void;
}

/** 一次签发里可以留存的部分：前缀与签发时间（列表里本来就显示它们） */
export interface IssuedTokenSummary {
  tokenPrefix: string;
  tokenIssuedAt: string;
}

/**
 * 给某个集群签发令牌；已有令牌时即为轮换，旧令牌立即失效。
 *
 * ⛔ 明文不作为 mutation 的返回值：mutation 的结果会留在 MutationCache 里
 * （只要页面还挂着就一直在），开发者工具与任何拿得到 QueryClient 的代码都读得到。
 * 所以明文在取数函数里直接交给调用方的 deliverToken，缓存里只留前缀与签发时间。
 */
export function issueClusterTokenMutation() {
  return mutationOptions({
    mutationFn: async ({ clusterId, deliverToken }: ClusterTokenRequest): Promise<IssuedTokenSummary> => {
      const { token, tokenPrefix, tokenIssuedAt } = (await issueClusterToken(clusterId)).data.data;
      deliverToken(token);
      return { tokenPrefix, tokenIssuedAt };
    },
    onSuccess: (_summary, _request, _onMutateResult, context) => invalidateRegistry(context.client),
  });
}

/**
 * 会话级数据的取数描述（全站只需要一份）
 */
import { queryOptions } from "@tanstack/react-query";
import { getClusterList } from "@/api/cluster";
import { sessionKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 已接入的集群列表。慢速轮询：新接入的集群与 OTel 可用性变化会自己出现 */
export function clusterListQuery() {
  return queryOptions({
    queryKey: sessionKeys.clusterList(),
    queryFn: async ({ signal }) => (await getClusterList(signal)).data?.clusters ?? [],
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
  });
}

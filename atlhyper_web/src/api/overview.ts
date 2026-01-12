import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { ClusterOverview } from "@/types/overview";

/**
 * 获取集群概览
 */
export function getClusterOverview(data: ClusterRequest) {
  return post<ClusterOverview, ClusterRequest>("/uiapi/overview/cluster/detail", data);
}

import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { ClusterInfo } from "@/types/cluster";

/**
 * 获取集群信息
 */
export function getClusterInfo() {
  return post<{ clusters: ClusterInfo[] }, undefined>("/uiapi/cluster/info", undefined);
}

import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { IngressOverview, IngressDetail, IngressDetailRequest } from "@/types/cluster";

/**
 * 获取 Ingress 概览
 */
export function getIngressOverview(data: ClusterRequest) {
  return post<IngressOverview, ClusterRequest>("/uiapi/cluster/ingress/list", data);
}

/**
 * 获取 Ingress 详情
 */
export function getIngressDetail(data: IngressDetailRequest) {
  return post<IngressDetail, IngressDetailRequest>("/uiapi/cluster/ingress/detail", data);
}

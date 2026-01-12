import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { ServiceOverview, ServiceDetail, ServiceDetailRequest } from "@/types/cluster";

/**
 * 获取 Service 概览
 */
export function getServiceOverview(data: ClusterRequest) {
  return post<ServiceOverview, ClusterRequest>("/uiapi/cluster/service/list", data);
}

/**
 * 获取 Service 详情
 */
export function getServiceDetail(data: ServiceDetailRequest) {
  return post<ServiceDetail, ServiceDetailRequest>("/uiapi/cluster/service/detail", data);
}

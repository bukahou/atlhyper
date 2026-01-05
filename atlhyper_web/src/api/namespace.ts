import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type {
  NamespaceOverview,
  NamespaceDetail,
  NamespaceDetailRequest,
  ConfigMapDTO,
  ConfigMapRequest,
} from "@/types/cluster";

/**
 * 获取 Namespace 概览
 */
export function getNamespaceOverview(data: ClusterRequest) {
  return post<NamespaceOverview, ClusterRequest>("/uiapi/namespace/overview", data);
}

/**
 * 获取 Namespace 详情
 */
export function getNamespaceDetail(data: NamespaceDetailRequest) {
  return post<NamespaceDetail, NamespaceDetailRequest>("/uiapi/namespace/detail", data);
}

/**
 * 获取 ConfigMap 列表（按 Namespace）
 */
export function getConfigMaps(data: ConfigMapRequest) {
  return post<ConfigMapDTO[], ConfigMapRequest>("/uiapi/configmap/detail", data);
}

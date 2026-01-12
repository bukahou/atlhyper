import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type {
  PodOverview,
  PodDetail,
  PodDetailRequest,
  PodLogsRequest,
  PodLogsResponse,
  PodOperationRequest,
} from "@/types/cluster";

/**
 * 获取 Pod 概览
 */
export function getPodOverview(data: ClusterRequest) {
  return post<PodOverview, ClusterRequest>("/uiapi/cluster/pod/list", data);
}

/**
 * 获取 Pod 详情
 */
export function getPodDetail(data: PodDetailRequest) {
  return post<PodDetail, PodDetailRequest>("/uiapi/cluster/pod/detail", data);
}

/**
 * 获取 Pod 日志
 */
export function getPodLogs(data: PodLogsRequest) {
  return post<PodLogsResponse, PodLogsRequest>("/uiapi/ops/pod/logs", data);
}

/**
 * 重启 Pod
 */
export function restartPod(data: PodOperationRequest) {
  return post<{ commandID: string; type: string; target: string }, PodOperationRequest>(
    "/uiapi/ops/pod/restart",
    data
  );
}

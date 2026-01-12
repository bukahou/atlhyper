import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type {
  DeploymentOverview,
  DeploymentDetail,
  DeploymentDetailRequest,
  WorkloadScaleRequest,
  WorkloadUpdateImageRequest,
} from "@/types/cluster";

/**
 * 获取 Deployment 概览
 */
export function getDeploymentOverview(data: ClusterRequest) {
  return post<DeploymentOverview, ClusterRequest>("/uiapi/cluster/deployment/list", data);
}

/**
 * 获取 Deployment 详情
 */
export function getDeploymentDetail(data: DeploymentDetailRequest) {
  return post<DeploymentDetail, DeploymentDetailRequest>("/uiapi/cluster/deployment/detail", data);
}

/**
 * Workload 扩缩容
 */
export function scaleDeployment(data: WorkloadScaleRequest) {
  return post<{ commandID: string; type: string; target: Record<string, string> }, WorkloadScaleRequest>(
    "/uiapi/ops/workload/scale",
    data
  );
}

/**
 * Deployment 重启（通过删除 Pod 触发）
 */
export function restartDeployment(data: DeploymentDetailRequest) {
  return post<string, DeploymentDetailRequest>("/uiapi/ops/deployment/restart", data);
}

/**
 * Workload 更新镜像
 */
export function updateDeploymentImage(data: WorkloadUpdateImageRequest) {
  return post<{ commandID: string; type: string; target: Record<string, string> }, WorkloadUpdateImageRequest>(
    "/uiapi/ops/workload/updateImage",
    data
  );
}

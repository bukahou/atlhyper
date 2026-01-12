import { post } from "./request";
import type { ClusterRequest } from "@/types/common";
import type { NodeOverview, NodeDetail, NodeDetailRequest, NodeOperationRequest } from "@/types/cluster";

/**
 * 获取 Node 概览
 */
export function getNodeOverview(data: ClusterRequest) {
  return post<NodeOverview, ClusterRequest>("/uiapi/cluster/node/list", data);
}

/**
 * 获取 Node 详情
 */
export function getNodeDetail(data: NodeDetailRequest) {
  return post<NodeDetail, NodeDetailRequest>("/uiapi/cluster/node/detail", data);
}

/**
 * Node Cordon（封锁节点）
 */
export function cordonNode(data: NodeOperationRequest) {
  return post<{ commandID: string; type: string; target: Record<string, string> }, NodeOperationRequest>(
    "/uiapi/ops/node/cordon",
    data
  );
}

/**
 * Node Uncordon（解封节点）
 */
export function uncordonNode(data: NodeOperationRequest) {
  return post<{ commandID: string; type: string; target: Record<string, string> }, NodeOperationRequest>(
    "/uiapi/ops/node/uncordon",
    data
  );
}

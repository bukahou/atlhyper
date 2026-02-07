/**
 * 服务网格 API
 */

import { get } from "./request";
import type {
  MeshTopologyResponse,
  MeshServiceDetailResponse,
  MeshTopologyParams,
  MeshServiceDetailParams,
} from "@/types/mesh";

/**
 * 获取服务网格拓扑
 */
export const getMeshTopology = (params?: MeshTopologyParams) => {
  return get<MeshTopologyResponse>("/api/v2/slo/mesh/topology", {
    cluster_id: params?.clusterId,
    time_range: params?.timeRange,
  });
};

/**
 * 获取服务详情
 */
export const getMeshServiceDetail = (params: MeshServiceDetailParams) => {
  return get<MeshServiceDetailResponse>("/api/v2/slo/mesh/service/detail", {
    cluster_id: params.clusterId,
    namespace: params.namespace,
    name: params.name,
    time_range: params.timeRange,
  });
};

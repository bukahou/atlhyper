/**
 * Mesh Mock — 查询函数
 *
 * 模拟 2 个读 API: getMeshTopology / getMeshServiceDetail
 */

import type {
  MeshTopologyResponse,
  MeshServiceDetailResponse,
} from "@/types/mesh";
import { MOCK_MESH_TOPOLOGY, getMockServiceDetail } from "./data";

/** 模拟获取服务网格拓扑 */
export function mockGetMeshTopology(): MeshTopologyResponse {
  return MOCK_MESH_TOPOLOGY;
}

/** 模拟获取服务详情 */
export function mockGetMeshServiceDetail(
  namespace: string,
  name: string,
): MeshServiceDetailResponse | null {
  return getMockServiceDetail(namespace, name);
}

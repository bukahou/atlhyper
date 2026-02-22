/**
 * Mesh 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 * SLO 模块的 mesh 数据共用 slo 的数据源模式
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/mesh";
import * as api from "@/api/mesh";
import type {
  MeshTopologyParams,
  MeshServiceDetailParams,
} from "@/types/mesh";

export async function getMeshTopology(params?: MeshTopologyParams) {
  if (getDataSourceMode("slo") === "mock") {
    return { data: mock.mockGetMeshTopology() };
  }
  return api.getMeshTopology(params);
}

export async function getMeshServiceDetail(params: MeshServiceDetailParams) {
  if (getDataSourceMode("slo") === "mock") {
    return { data: mock.mockGetMeshServiceDetail(params.namespace, params.name) };
  }
  return api.getMeshServiceDetail(params);
}

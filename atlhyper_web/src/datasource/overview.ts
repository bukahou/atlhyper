/**
 * Overview 数据源代理
 *
 * 根据中心配置自动切换 mock / api
 */

import { getDataSourceMode } from "@/config/data-source";
import * as mock from "@/mock/overview";
import * as api from "@/api/overview";

export async function getClusterOverview(params: { cluster_id: string }) {
  if (getDataSourceMode("overview") === "mock") {
    return mock.mockGetClusterOverview();
  }
  return api.getClusterOverview(params);
}

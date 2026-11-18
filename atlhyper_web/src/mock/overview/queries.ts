/**
 * Overview mock queries — 返回与数据源相同的形态（解包后的 ClusterOverview）
 */

import type { ClusterOverview } from "@/types/overview";
import { MOCK_CLUSTER_OVERVIEW } from "./data";

export function mockGetClusterOverview(): ClusterOverview {
  return MOCK_CLUSTER_OVERVIEW;
}

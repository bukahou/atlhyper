/**
 * Overview mock queries — 返回与真实 API 相同的响应结构
 */

import { MOCK_CLUSTER_OVERVIEW } from "./data";

export function mockGetClusterOverview() {
  return { data: { data: MOCK_CLUSTER_OVERVIEW } };
}

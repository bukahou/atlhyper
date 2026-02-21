/**
 * APM Mock — 拓扑图 API
 */

import type { Topology } from "@/types/model/apm";
import { mockTopology } from "./data";

export function mockGetTopology(): Topology {
  return mockTopology;
}

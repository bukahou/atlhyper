/**
 * APM Mock — 服务列表 API
 */

import type { APMService } from "@/types/model/apm";
import { mockAPMServices } from "./data";

export function mockGetAPMServices(): APMService[] {
  return mockAPMServices;
}

import { post } from "./request";
import type { EventOverview, EventLogRequest } from "@/types/cluster";

/**
 * 获取事件日志
 */
export function getEventLogs(data: EventLogRequest) {
  return post<EventOverview, EventLogRequest>("/uiapi/event/logs", data);
}
